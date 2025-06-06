package oci

import (
	"context"
	"encoding/json"
	"os"
	"time"

	"gitlab.com/tozd/go/errors"

	v1 "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/walteh/ec1/pkg/units"
)

// Constants for the OCI package
const (
	// Default cache directory name
	DefaultCacheDir = "ec1-oci-cache"

	// Cache subdirectory names
	CacheMetadataFile = "metadata.json"
	CacheRootfsDir    = "rootfs"
	CacheExt4File     = "rootfs.ext4"
	CacheImagePath    = "image.json"

	// Default cache expiration
	DefaultCacheExpiration = 24 * time.Hour

	// File permissions
	CacheDirPerm  = 0755
	CacheFilePerm = 0644

	// Ext4 disk size limit
	DefaultExt4MaxSize = 1 << 30 // 1 GiB
)

// ImageFetcher fetches container images and returns OCI layout directories
// This interface abstracts the source of images (registry, local cache, containerd, etc.)
type ImageFetcher interface {
	// FetchImage fetches an image and returns the path to an OCI layout directory
	// imageRef: image reference like "alpine:3.21" or "docker.io/library/alpine:3.21"
	// Returns: path to the OCI layout directory
	FetchImageToOCILayout(ctx context.Context, imageRef string) (dir string, err error)
}

type ImageFetchConverter interface {
	ImageFetcher
	FilesystemConverter
}

func FetchAndConvertImage(ctx context.Context, fetcher ImageFetchConverter, imageRef string, platform units.Platform) (*Image, error) {
	ociLayoutPath, err := fetcher.FetchImageToOCILayout(ctx, imageRef)
	if err != nil {
		return nil, errors.Errorf("fetching image to OCI layout: %w", err)
	}

	converted, err := fetcher.ConvertOCILayoutToRootfsAndExt4(ctx, ociLayoutPath, platform)
	if err != nil {
		return nil, errors.Errorf("converting OCI layout to rootfs and ext4: %w", err)
	}

	return converted, nil
}

// FilesystemConverter converts OCI layout directories to VM-ready filesystems
// This interface abstracts the conversion process from OCI to rootfs/ext4
type FilesystemConverter interface {
	// ConvertToFilesystem converts an OCI layout to a rootfs and ext4 for a specific platform
	// ociLayoutPath: path to OCI layout directory
	// platform: target platform (e.g., "linux/amd64")
	// destDir: destination directory where rootfs and ext4 will be created
	// Returns: metadata about the converted image
	ConvertOCILayoutToRootfsAndExt4(ctx context.Context, ociLayoutPath string, platform units.Platform) (*Image, error)
}

// ImageCache manages cached container images for fast VM creation
// This is the main interface for the OCI cache system
type ImageCache interface {
	ImageFetcher
	FilesystemConverter
}

// CachedImage represents a cached container image ready for VM use
type Image struct {
	Platform   units.Platform `json:"platform"`
	RootfsPath string         `json:"rootfs_path"`
	Ext4Path   string         `json:"ext4_path"`
	Metadata   *v1.Image      `json:"metadata"`
	CachedAt   time.Time      `json:"cached_at"`
}

func SaveImageToCache(ctx context.Context, file string, i *Image) error {
	metadataBytes, err := json.Marshal(i)
	if err != nil {
		return errors.Errorf("marshaling metadata: %w", err)
	}
	return os.WriteFile(file, metadataBytes, CacheFilePerm)
}

func LoadImageFromCache(ctx context.Context, file string) (*Image, error) {
	metadataBytes, err := os.ReadFile(file)
	if err != nil {
		return nil, errors.Errorf("reading metadata: %w", err)
	}
	var i Image
	if err := json.Unmarshal(metadataBytes, &i); err != nil {
		return nil, errors.Errorf("unmarshaling metadata: %w", err)
	}
	return &i, nil
}
