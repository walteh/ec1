package oci

import (
	"context"
	"log/slog"
	"os"
	"os/exec"

	"gitlab.com/tozd/go/errors"
)

// SkopeoImageDownloader implements ImageDownloader using skopeo for production use
type SkopeoImageDownloader struct {
	TempDir string // Optional: custom temp directory for downloads
}

// DownloadImage downloads an image using skopeo and returns the OCI layout path
func (d *SkopeoImageDownloader) DownloadImage(ctx context.Context, imageRef string) (string, error) {
	slog.InfoContext(ctx, "downloading image with skopeo", "image", imageRef)

	// Create temp directory for this download
	tempDir := d.TempDir
	if tempDir == "" {
		tempDir = os.TempDir()
	}

	ociLayoutPath, err := os.MkdirTemp(tempDir, "oci-layout-*")
	if err != nil {
		return "", errors.Errorf("creating temp directory: %w", err)
	}

	// Use skopeo to copy the image to OCI layout format
	cmd := exec.CommandContext(ctx, "skopeo", "copy", 
		"docker://"+imageRef, 
		"oci:"+ociLayoutPath)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		// Clean up on failure
		os.RemoveAll(ociLayoutPath)
		return "", errors.Errorf("skopeo copy failed: %w", err)
	}

	slog.InfoContext(ctx, "successfully downloaded image", "image", imageRef, "oci_layout", ociLayoutPath)
	return ociLayoutPath, nil
}
