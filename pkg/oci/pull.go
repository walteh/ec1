package oci

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/containers/image/v5/copy"
	"github.com/containers/image/v5/docker"
	"github.com/containers/image/v5/image"
	"github.com/containers/image/v5/oci/layout"
	"github.com/containers/image/v5/signature"
	"github.com/containers/image/v5/types"
	"gitlab.com/tozd/go/errors"

	v1 "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/walteh/ec1/pkg/units"
)

// SkopeoImageDownloader implements ImageDownloader using skopeo/containers library
type SkopeoImageDownloader struct {
	RootDir string
}

func (d *SkopeoImageDownloader) DownloadImage(ctx context.Context, imageRef string) (string, error) {
	slog.InfoContext(ctx, "downloading container image using skopeo",
		"image", imageRef,
	)

	// get the current main digest of the image

	// Create policy context (allow all for now - in production this should be more restrictive)
	policyContext, err := signature.NewPolicyContext(&signature.Policy{
		Default: []signature.PolicyRequirement{signature.NewPRInsecureAcceptAnything()},
	})
	if err != nil {
		return "", errors.Errorf("creating policy context: %w", err)
	}
	defer policyContext.Destroy()

	// Parse source image reference
	srcRef, err := docker.ParseReference("//" + imageRef)
	if err != nil {
		return "", errors.Errorf("parsing source reference: %w", err)
	}

	// get image index manifest
	srcImg, err := srcRef.NewImage(ctx, &types.SystemContext{})
	if err != nil {
		return "", errors.Errorf("creating source image: %w", err)
	}
	defer srcImg.Close()

	bdig, _, err := srcImg.Manifest(ctx)
	if err != nil {
		return "", errors.Errorf("getting manifest: %w", err)
	}

	// var manifest v1.Manifest
	// if sdig == "application/vnd.oci.image.index.v1+json" {
	// 	var index v1.Index
	// 	if err := json.Unmarshal(bdig, &index); err != nil {
	// 		return "", errors.Errorf("unmarshalling index: %w", err)
	// 	}

	// } else {
	// 	var manifest v1.Manifest
	// 	if err := json.Unmarshal(bdig, &manifest); err != nil {
	// 		return "", errors.Errorf("unmarshalling manifest: %w", err)
	// 	}
	// }

	var manifest v1.Manifest
	if err := json.Unmarshal(bdig, &manifest); err != nil {
		return "", errors.Errorf("unmarshalling manifest: %w", err)
	}
	hashd := sha256.Sum256(bdig)
	destDir := filepath.Join(d.RootDir, hex.EncodeToString(hashd[:]))

	// Create OCI layout destination
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return "", errors.Errorf("creating destination directory: %w", err)
	}

	destRef, err := layout.NewReference(destDir, imageRef)
	if err != nil {
		return "", errors.Errorf("creating oci layout reference: %w", err)
	}

	// Create system context with platform information
	sysCtx := &types.SystemContext{}

	// Copy image from source to OCI layout
	_, err = copy.Image(ctx, policyContext, destRef, srcRef, &copy.Options{
		SourceCtx:      sysCtx,
		DestinationCtx: sysCtx,
		ReportWriter:   os.Stdout,
		// Instances:      digests,
	})
	if err != nil {
		return "", errors.Errorf("copying image: %w", err)
	}

	slog.InfoContext(ctx, "successfully downloaded container image", "dest", destDir)
	return destDir, nil
}

func ExtractOCIImageToDir(ctx context.Context, downloadedPath string, destDir string, platform units.Platform) (*v1.Image, error) {
	slog.InfoContext(ctx, "extracting image from OCI layout",
		"source", downloadedPath,
		"dest", destDir)

	sysCtx := &types.SystemContext{
		OSChoice:           platform.OS(),
		ArchitectureChoice: platform.Arch(),
	}

	// Create image reference from the downloaded OCI layout
	imgRef, err := layout.NewReference(downloadedPath, "")
	if err != nil {
		return nil, errors.Errorf("creating image reference: %w", err)
	}

	// Create image source
	imgSrc, err := imgRef.NewImageSource(ctx, sysCtx)
	if err != nil {
		return nil, errors.Errorf("creating image source: %w", err)
	}
	defer imgSrc.Close()

	// Create image from source
	img, err := image.FromSource(ctx, sysCtx, imgSrc)
	if err != nil {
		return nil, errors.Errorf("creating image from source: %w", err)
	}
	defer img.Close()

	// Get OCI configuration
	ociConfig, err := img.OCIConfig(ctx)
	if err != nil {
		return nil, errors.Errorf("getting OCI config: %w", err)
	}

	// Extract the filesystem layers to a rootfs directory
	err = extractLayersFromOCILayout(ctx, downloadedPath, destDir, img)
	if err != nil {
		return nil, errors.Errorf("extracting layers: %w", err)
	}

	slog.InfoContext(ctx, "successfully extracted container image with metadata",
		"dest", destDir,
		"entrypoint", ociConfig.Config.Entrypoint,
		"cmd", ociConfig.Config.Cmd,
		"platform", ociConfig.Platform)

	return ociConfig, nil
}

// extractLayersFromOCILayout extracts the filesystem layers from an OCI layout to create a rootfs
func extractLayersFromOCILayout(ctx context.Context, ociLayoutDir, destDir string, img types.Image) error {
	slog.InfoContext(ctx, "extracting layers from OCI layout", "oci_layout", ociLayoutDir, "dest", destDir)

	// Create the destination filesystem directory
	fsDir := filepath.Join(destDir, "rootfs")
	if err := os.MkdirAll(fsDir, 0755); err != nil {
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

		if err := extractLayer(ctx, layerPath, fsDir); err != nil {
			return errors.Errorf("extracting layer %d: %w", i+1, err)
		}
	}

	slog.InfoContext(ctx, "successfully extracted layers to rootfs", "dest", fsDir)
	return nil
}

// extractLayer extracts a single compressed layer tar file to the destination
func extractLayer(ctx context.Context, layerPath, destDir string) error {
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
			if err := extractFile(tarReader, targetPath, header); err != nil {
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
func extractFile(tarReader *tar.Reader, targetPath string, header *tar.Header) error {
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

// Legacy function for backward compatibility - now uses the separated approach
func PullAndExtractImageSkopeo(ctx context.Context, imageRef, destDir string, sysCtx *types.SystemContext) (*v1.Image, error) {
	// Extract platform from system context
	platform := units.Platform("linux/amd64") // default
	if sysCtx != nil {
		platform = units.Platform(sysCtx.OSChoice + "/" + sysCtx.ArchitectureChoice)
	}

	// Use the separated approach
	downloader := &SkopeoImageDownloader{}

	// Download the image
	ociLayoutPath, err := downloader.DownloadImage(ctx, imageRef)
	if err != nil {
		return nil, err
	}

	// Extract the image
	return ExtractOCIImageToDir(ctx, ociLayoutPath, destDir, platform)
}
