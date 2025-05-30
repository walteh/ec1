package oci

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/Microsoft/hcsshim/ext4/tar2ext4"
	"github.com/containers/image/v5/image"
	"github.com/containers/image/v5/oci/layout"
	"github.com/containers/image/v5/types"
	"github.com/mholt/archives"
	"gitlab.com/tozd/go/errors"

	v1 "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/walteh/ec1/pkg/units"
)

// ImageExtractor extracts container images from OCI layout to filesystem
type ImageExtractor interface {
	// ExtractImage extracts an OCI layout to a rootfs for a specific platform
	// ociLayoutPath: path to OCI layout directory
	// platform: target platform (e.g., "linux/amd64")
	// destDir: destination directory where rootfs and ext4 will be created
	// Returns: metadata about the extracted image
	ExtractImage(ctx context.Context, ociLayoutPath string, platform units.Platform, destDir string) (*v1.Image, error)
}

// OCIImageExtractor implements ImageExtractor for OCI layout processing
type OCIImageExtractor struct{}

func (e *OCIImageExtractor) ExtractImage(ctx context.Context, ociLayoutPath string, platform units.Platform, destDir string) (*v1.Image, error) {
	slog.InfoContext(ctx, "extracting image from OCI layout",
		"source", ociLayoutPath,
		"platform", string(platform),
		"dest", destDir)

	// First, check if we need to handle duplicate manifests in the index
	err := e.ensureCleanIndex(ctx, ociLayoutPath)
	if err != nil {
		return nil, errors.Errorf("cleaning index: %w", err)
	}

	// Create image reference from the OCI layout
	imgRef, err := layout.NewReference(ociLayoutPath, "")
	if err != nil {
		return nil, errors.Errorf("creating image reference: %w", err)
	}

	// Create system context with platform information
	sysCtx := &types.SystemContext{
		OSChoice:           platform.OS(),
		ArchitectureChoice: platform.Arch(),
	}

	// Create image source
	imgSrc, err := imgRef.NewImageSource(ctx, sysCtx)
	if err != nil {
		return nil, errors.Errorf("creating image source: %w", err)
	}
	defer imgSrc.Close()

	// Create image from source with better error handling for multi-platform
	img, err := image.FromSource(ctx, sysCtx, imgSrc)
	if err != nil {
		// If we get "more than one image" error, try to be more specific
		if strings.Contains(err.Error(), "more than one image") {
			slog.WarnContext(ctx, "multi-platform image detected, attempting platform-specific selection",
				"platform", string(platform),
				"os", platform.OS(),
				"arch", platform.Arch())
		}
		return nil, errors.Errorf("creating image from source: %w", err)
	}
	defer img.Close()

	// Get OCI configuration
	ociConfig, err := img.OCIConfig(ctx)
	if err != nil {
		return nil, errors.Errorf("getting OCI config: %w", err)
	}

	// Extract the filesystem layers to a rootfs directory
	rootfsPath := filepath.Join(destDir, "rootfs")
	err = e.extractLayersFromOCILayout(ctx, ociLayoutPath, rootfsPath, img)
	if err != nil {
		return nil, errors.Errorf("extracting layers: %w", err)
	}

	// Create ext4 disk image from rootfs
	ext4Path := filepath.Join(destDir, "rootfs.ext4")
	err = e.createExt4FromRootfs(ctx, rootfsPath, ext4Path)
	if err != nil {
		return nil, errors.Errorf("creating ext4 image: %w", err)
	}

	slog.InfoContext(ctx, "successfully extracted container image",
		"dest", destDir,
		"rootfs", rootfsPath,
		"ext4", ext4Path,
		"entrypoint", ociConfig.Config.Entrypoint,
		"cmd", ociConfig.Config.Cmd,
		"platform", ociConfig.Platform)

	return ociConfig, nil
}

// ensureCleanIndex checks for and fixes duplicate manifest entries in index.json
func (e *OCIImageExtractor) ensureCleanIndex(ctx context.Context, ociLayoutPath string) error {
	indexPath := filepath.Join(ociLayoutPath, "index.json")
	
	// Read the current index
	indexData, err := os.ReadFile(indexPath)
	if err != nil {
		return errors.Errorf("reading index.json: %w", err)
	}

	var index v1.Index
	if err := json.Unmarshal(indexData, &index); err != nil {
		return errors.Errorf("unmarshaling index.json: %w", err)
	}

	// Check for duplicate manifests
	seen := make(map[string]bool)
	var uniqueManifests []v1.Descriptor
	duplicatesFound := false

	for _, manifest := range index.Manifests {
		key := manifest.Digest.String() + ":" + manifest.MediaType
		if !seen[key] {
			seen[key] = true
			uniqueManifests = append(uniqueManifests, manifest)
		} else {
			duplicatesFound = true
			slog.WarnContext(ctx, "found duplicate manifest entry, removing",
				"digest", manifest.Digest.String(),
				"media_type", manifest.MediaType)
		}
	}

	// If we found duplicates, write the cleaned index
	if duplicatesFound {
		index.Manifests = uniqueManifests
		
		cleanedData, err := json.Marshal(index)
		if err != nil {
			return errors.Errorf("marshaling cleaned index: %w", err)
		}

		if err := os.WriteFile(indexPath, cleanedData, 0644); err != nil {
			return errors.Errorf("writing cleaned index: %w", err)
		}

		slog.InfoContext(ctx, "cleaned duplicate manifest entries",
			"original_count", len(index.Manifests) + (len(index.Manifests) - len(uniqueManifests)),
			"cleaned_count", len(uniqueManifests))
	}

	return nil
}

// extractLayersFromOCILayout extracts the filesystem layers from an OCI layout to create a rootfs
func (e *OCIImageExtractor) extractLayersFromOCILayout(ctx context.Context, ociLayoutDir, destDir string, img types.Image) error {
	slog.InfoContext(ctx, "extracting layers from OCI layout", "oci_layout", ociLayoutDir, "dest", destDir)

	// Create the destination filesystem directory
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return errors.Errorf("creating filesystem directory: %w", err)
	}

	// Get layer information from the image
	layerInfos := img.LayerInfos()
	slog.InfoContext(ctx, "found layers", "count", len(layerInfos))

	// Extract each layer in order
	for i, layerInfo := range layerInfos {
		// Remove the "sha256:" prefix from digest to get the filename
		layerFile := strings.TrimPrefix(layerInfo.Digest.String(), "sha256:")
		// OCI layout stores blobs in blobs/sha256/ directory
		layerPath := filepath.Join(ociLayoutDir, "blobs", "sha256", layerFile)

		slog.InfoContext(ctx, "extracting layer", "layer", i+1, "total", len(layerInfos), "digest", layerInfo.Digest.String(), "path", layerPath)

		if err := e.extractLayer(ctx, layerPath, destDir); err != nil {
			return errors.Errorf("extracting layer %d: %w", i+1, err)
		}
	}

	slog.InfoContext(ctx, "successfully extracted layers to rootfs", "dest", destDir)
	return nil
}

// extractLayer extracts a single compressed layer tar file to the destination
func (e *OCIImageExtractor) extractLayer(ctx context.Context, layerPath, destDir string) error {
	file, err := os.Open(layerPath)
	if err != nil {
		return errors.Errorf("opening layer file: %w", err)
	}
	defer file.Close()

	// Decompress the gzipped layer
	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return errors.Errorf("creating gzip reader: %w", err)
	}
	defer gzReader.Close()

	// Extract the tar archive
	tarReader := tar.NewReader(gzReader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return errors.Errorf("reading tar header: %w", err)
		}

		// Skip whiteout files (used for deletions in overlay filesystems)
		if strings.HasPrefix(filepath.Base(header.Name), ".wh.") {
			continue
		}

		targetPath := filepath.Join(destDir, header.Name)

		// Ensure the target directory exists
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return errors.Errorf("creating directory: %w", err)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, os.FileMode(header.Mode)); err != nil {
				return errors.Errorf("creating directory %s: %w", targetPath, err)
			}
		case tar.TypeReg:
			if err := e.extractFile(tarReader, targetPath, header); err != nil {
				return errors.Errorf("extracting file %s: %w", targetPath, err)
			}
		case tar.TypeSymlink:
			if err := os.Symlink(header.Linkname, targetPath); err != nil && !os.IsExist(err) {
				return errors.Errorf("creating symlink %s: %w", targetPath, err)
			}
		case tar.TypeLink:
			linkTarget := filepath.Join(destDir, header.Linkname)
			if err := os.Link(linkTarget, targetPath); err != nil && !os.IsExist(err) {
				return errors.Errorf("creating hard link %s: %w", targetPath, err)
			}
		default:
			slog.WarnContext(ctx, "skipping unsupported file type", "type", header.Typeflag, "file", header.Name)
		}
	}

	return nil
}

// extractFile extracts a single file from the tar reader
func (e *OCIImageExtractor) extractFile(tarReader *tar.Reader, targetPath string, header *tar.Header) error {
	file, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
	if err != nil {
		return errors.Errorf("creating file: %w", err)
	}
	defer file.Close()

	if _, err := io.Copy(file, tarReader); err != nil {
		return errors.Errorf("copying file content: %w", err)
	}

	return nil
}

// createExt4FromRootfs creates an ext4 disk image from a rootfs directory
func (e *OCIImageExtractor) createExt4FromRootfs(ctx context.Context, rootfsPath, ext4Path string) error {
	slog.InfoContext(ctx, "creating ext4 disk image", "rootfs", rootfsPath, "ext4", ext4Path)

	// Use tar2ext4 to create a proper ext4 image
	fles, err := archives.FilesFromDisk(ctx, &archives.FromDiskOptions{}, map[string]string{
		rootfsPath: ext4Path,
	})
	if err != nil {
		return errors.Errorf("getting files from disk: %w", err)
	}

	// Remove existing ext4 file if it exists
	os.Remove(ext4Path)

	// Pack the folder into a tar stream
	pr, pw := io.Pipe()
	go func() {
		defer pw.Close()
		err := (&archives.Tar{}).Archive(ctx, pw, fles)
		if err != nil {
			slog.ErrorContext(ctx, "failed to archive files", "error", err)
		}
	}()

	// Create the ext4 file
	out, err := os.Create(ext4Path)
	if err != nil {
		return errors.Errorf("creating ext4 file: %w", err)
	}
	defer out.Close()

	// Convert tar stream to ext4
	err = tar2ext4.ConvertTarToExt4(pr, out, tar2ext4.MaximumDiskSize(1<<30), tar2ext4.InlineData) // 1 GiB
	if err != nil {
		return errors.Errorf("converting tar to ext4: %w", err)
	}

	slog.InfoContext(ctx, "ext4 disk image created", "path", ext4Path)
	return nil
}
