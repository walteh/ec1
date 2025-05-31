package oci

import (
	"context"

	"gitlab.com/tozd/go/errors"

	"github.com/walteh/ec1/pkg/units"
)

// imageCache implements ImageCache interface
type imageCache struct {
	cacheDir     string
	ImageFetcher // Optional - only used for standalone mode
	FilesystemConverter
}

// NewImageCacheWithFS creates a new cache instance with custom filesystem provider
func NewImageCache(cacheDir string, fetcher ImageFetcher, converter FilesystemConverter) *imageCache {
	return &imageCache{
		cacheDir:            cacheDir,
		ImageFetcher:        NewCachedFetcher(cacheDir, fetcher),
		FilesystemConverter: NewCachedConverter(converter),
	}
}

func (c *imageCache) Preload(ctx context.Context, imageRef string, platform units.Platform) error {
	ociLayoutPath, err := c.ImageFetcher.FetchImageToOCILayout(ctx, imageRef)
	if err != nil {
		return errors.Errorf("fetching image to OCI layout: %w", err)
	}

	_, err = c.FilesystemConverter.ConvertOCILayoutToRootfsAndExt4(ctx, ociLayoutPath, platform)
	if err != nil {
		return errors.Errorf("converting OCI layout to rootfs and ext4: %w", err)
	}
	return nil
}
