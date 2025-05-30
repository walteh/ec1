package oci

import (
	"archive/tar"
	"compress/gzip"
	"context"
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
)

// pullAndExtractImageSkopeo pulls a container image using the skopeo approach and extracts it to OCI layout
func PullAndExtractImageSkopeo(ctx context.Context, imageRef, destDir string, sysCtx *types.SystemContext) (*v1.Image, error) {
	slog.InfoContext(ctx, "pulling container image using skopeo approach", "image", imageRef)

	// Create policy context (allow all for now - in production this should be more restrictive)
	policyContext, err := signature.NewPolicyContext(&signature.Policy{
		Default: []signature.PolicyRequirement{signature.NewPRInsecureAcceptAnything()},
	})
	if err != nil {
		return nil, errors.Errorf("creating policy context: %w", err)
	}
	defer policyContext.Destroy()

	// Parse source image reference
	srcRef, err := docker.ParseReference("//" + imageRef)
	if err != nil {
		return nil, errors.Errorf("parsing source reference: %w", err)
	}

	// Create OCI layout destination
	ociLayoutDir := filepath.Join(destDir, "oci-layout")
	if err := os.MkdirAll(ociLayoutDir, 0755); err != nil {
		return nil, errors.Errorf("creating oci layout directory: %w", err)
	}

	destRef, err := layout.NewReference(ociLayoutDir, imageRef)
	if err != nil {
		return nil, errors.Errorf("creating oci layout reference: %w", err)
	}

	// Copy image from source to OCI layout
	_, err = copy.Image(ctx, policyContext, destRef, srcRef, &copy.Options{
		SourceCtx:      sysCtx,
		DestinationCtx: sysCtx,
		ReportWriter:   os.Stdout,
	})
	if err != nil {
		return nil, errors.Errorf("copying image: %w", err)
	}

	// Create image source from the copied OCI layout to extract configuration
	imgSrc, err := destRef.NewImageSource(ctx, sysCtx)
	if err != nil {
		return nil, errors.Errorf("creating image source: %w", err)
	}
	defer imgSrc.Close()

	// Create image from source to get configuration
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
	err = extractLayersFromOCILayout(ctx, ociLayoutDir, destDir, img)
	if err != nil {
		return nil, errors.Errorf("extracting layers: %w", err)
	}

	slog.InfoContext(ctx, "successfully extracted container image with metadata",
		"dest", destDir,
		"entrypoint", ociConfig.Config.Entrypoint,
		"cmd", ociConfig.Config.Cmd)

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
