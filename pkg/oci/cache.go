package oci

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/containers/image/v5/types"
	"gitlab.com/tozd/go/errors"

	v1 "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/walteh/ec1/pkg/virtio"
)

// CacheEntry represents a cached container image with metadata
type CacheEntry struct {
	ImageRef     string    `json:"image_ref"`
	Platform     string    `json:"platform"`
	CachedAt     time.Time `json:"cached_at"`
	ExpiresAt    time.Time `json:"expires_at"`
	RootfsPath   string    `json:"rootfs_path"`
	MetadataPath string    `json:"metadata_path"`
	Size         int64     `json:"size"`
}

const (
	// CacheExpiration is how long cached images are valid
	CacheExpiration = 24 * time.Hour

	// CacheMetadataFile is the name of the cache metadata file
	CacheMetadataFile = "cache-metadata.json"
)

// getCacheDir returns the cache directory for OCI images
func getCacheDir() (string, error) {
	userCacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", errors.Errorf("getting user cache dir: %w", err)
	}
	return filepath.Join(userCacheDir, "ec1", "cache", "oci"), nil
}

// getCacheDirForImage returns a cache directory specific to an image and platform
func getCacheDirForImage(imageRef string, platform *types.SystemContext) (string, error) {
	// Create a unique identifier for this image+platform combination
	platformStr := "linux-amd64" // default
	if platform != nil {
		platformStr = fmt.Sprintf("%s-%s", platform.OSChoice, platform.ArchitectureChoice)
	}

	// Hash the image reference and platform to create a unique directory name
	hasher := sha256.New()
	hasher.Write([]byte(imageRef + ":" + platformStr))
	hash := hex.EncodeToString(hasher.Sum(nil))

	// Create a readable directory name
	imageName := strings.ReplaceAll(imageRef, "/", "_")
	imageName = strings.ReplaceAll(imageName, ":", "_")
	if len(imageName) > 50 {
		imageName = imageName[:50]
	}

	dirname := fmt.Sprintf("%s_%s_%s", imageName, platformStr, hash[:16])

	cacheDir, err := getCacheDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(cacheDir, dirname), nil
}

// loadCacheEntry loads cache metadata for an image
func loadCacheEntry(cacheDir string) (*CacheEntry, error) {
	metadataPath := filepath.Join(cacheDir, CacheMetadataFile)

	data, err := os.ReadFile(metadataPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No cache entry exists
		}
		return nil, errors.Errorf("reading cache metadata: %w", err)
	}

	var entry CacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, errors.Errorf("unmarshaling cache metadata: %w", err)
	}

	return &entry, nil
}

// saveCacheEntry saves cache metadata for an image
func saveCacheEntry(cacheDir string, entry *CacheEntry) error {
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return errors.Errorf("creating cache directory: %w", err)
	}

	metadataPath := filepath.Join(cacheDir, CacheMetadataFile)

	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return errors.Errorf("marshaling cache metadata: %w", err)
	}

	if err := os.WriteFile(metadataPath, data, 0644); err != nil {
		return errors.Errorf("writing cache metadata: %w", err)
	}

	return nil
}

// isCacheValid checks if a cache entry is still valid
func isCacheValid(entry *CacheEntry) bool {
	if entry == nil {
		return false
	}

	// Check if cache has expired
	if time.Now().After(entry.ExpiresAt) {
		return false
	}

	// Check if cached files still exist
	if _, err := os.Stat(entry.RootfsPath); err != nil {
		return false
	}

	if _, err := os.Stat(entry.MetadataPath); err != nil {
		return false
	}

	return true
}

// getCachedRootfsSize calculates the size of a cached rootfs directory
func getCachedRootfsSize(rootfsPath string) (int64, error) {
	var size int64

	err := filepath.Walk(rootfsPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})

	return size, err
}

// ContainerToVirtioDeviceCached is a cached version of ContainerToVirtioDevice
// It caches extracted container images for 24 hours and handles offline scenarios gracefully
func ContainerToVirtioDeviceCached(ctx context.Context, opts ContainerToVirtioOptions) (virtio.VirtioDevice, *v1.Image, error) {
	slog.InfoContext(ctx, "converting OCI container to virtio device with caching",
		"image", opts.ImageRef,
		"cache_expiration", CacheExpiration)

	// Get cache directory for this image
	cacheDir, err := getCacheDirForImage(opts.ImageRef, opts.Platform)
	if err != nil {
		slog.WarnContext(ctx, "failed to get cache directory, falling back to non-cached", "error", err)
		return ContainerToVirtioDevice(ctx, opts)
	}

	// Load existing cache entry
	cacheEntry, err := loadCacheEntry(cacheDir)
	if err != nil {
		slog.WarnContext(ctx, "failed to load cache entry, will try to create new one", "error", err)
	}

	// Check if cache is valid
	if isCacheValid(cacheEntry) {
		slog.InfoContext(ctx, "using cached container image",
			"image", opts.ImageRef,
			"cached_at", cacheEntry.CachedAt,
			"expires_at", cacheEntry.ExpiresAt,
			"size_mb", cacheEntry.Size/1024/1024)

		return loadFromCache(ctx, cacheEntry, opts)
	}

	// Cache is invalid or doesn't exist, try to download and cache
	slog.InfoContext(ctx, "cache miss or expired, downloading container image",
		"image", opts.ImageRef,
		"cache_exists", cacheEntry != nil,
		"cache_valid", isCacheValid(cacheEntry))

	// Try to download and extract the image
	device, metadata, err := downloadAndCache(ctx, opts, cacheDir)
	if err != nil {
		// If we have an expired cache entry and download failed (e.g., offline),
		// try to use the expired cache as a fallback
		if cacheEntry != nil {
			slog.WarnContext(ctx, "download failed but found expired cache, using as fallback",
				"error", err,
				"cached_at", cacheEntry.CachedAt,
				"expired_since", time.Since(cacheEntry.ExpiresAt))

			fallbackDevice, fallbackMetadata, fallbackErr := loadFromCache(ctx, cacheEntry, opts)
			if fallbackErr == nil {
				slog.InfoContext(ctx, "successfully loaded from expired cache as fallback")
				return fallbackDevice, fallbackMetadata, nil
			}

			slog.ErrorContext(ctx, "failed to load from expired cache fallback", "fallback_error", fallbackErr)
		}

		return nil, nil, errors.Errorf("failed to download image and no valid cache available: %w", err)
	}

	return device, metadata, nil
}

// downloadAndCache downloads an image and caches it
func downloadAndCache(ctx context.Context, opts ContainerToVirtioOptions, cacheDir string) (virtio.VirtioDevice, *v1.Image, error) {
	// Ensure cache directory exists
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, nil, errors.Errorf("creating cache directory: %w", err)
	}

	// Create a temporary directory for the download
	tempDir, err := os.MkdirTemp(cacheDir, "download-*")
	if err != nil {
		return nil, nil, errors.Errorf("creating temp directory: %w", err)
	}

	// Clean up temp directory on error
	defer func() {
		if err != nil {
			os.RemoveAll(tempDir)
		}
	}()

	// Update opts to use the temp directory
	tempOpts := opts
	tempOpts.OutputDir = tempDir
	tempOpts.MountPoint = filepath.Join(tempDir, "mount")

	// Download and extract the image
	device, metadata, err := ContainerToVirtioDevice(ctx, tempOpts)
	if err != nil {
		return nil, nil, errors.Errorf("downloading container image: %w", err)
	}

	// Move the extracted content to the cache
	cachedRootfsPath := filepath.Join(cacheDir, "rootfs")
	cachedMetadataPath := filepath.Join(cacheDir, "metadata.json")

	// Remove old cache if it exists
	os.RemoveAll(cachedRootfsPath)
	os.Remove(cachedMetadataPath)

	// Find the extracted rootfs in the temp directory
	tempRootfsPath := ""
	if vfsDevice, ok := device.(*virtio.VirtioFs); ok {
		tempRootfsPath = vfsDevice.SharedDir
	} else {
		// Fallback: look for rootfs directory
		tempRootfsPath = filepath.Join(tempDir, "rootfs")
	}

	// Move rootfs to cache
	if err := os.Rename(tempRootfsPath, cachedRootfsPath); err != nil {
		return nil, nil, errors.Errorf("moving rootfs to cache: %w", err)
	}

	// Save metadata to cache
	metadataData, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return nil, nil, errors.Errorf("marshaling metadata: %w", err)
	}

	if err := os.WriteFile(cachedMetadataPath, metadataData, 0644); err != nil {
		return nil, nil, errors.Errorf("writing metadata to cache: %w", err)
	}

	// Calculate cache size
	cacheSize, err := getCachedRootfsSize(cachedRootfsPath)
	if err != nil {
		slog.WarnContext(ctx, "failed to calculate cache size", "error", err)
		cacheSize = 0
	}

	// Create cache entry
	now := time.Now()
	cacheEntry := &CacheEntry{
		ImageRef:     opts.ImageRef,
		Platform:     getPlatformString(opts.Platform),
		CachedAt:     now,
		ExpiresAt:    now.Add(CacheExpiration),
		RootfsPath:   cachedRootfsPath,
		MetadataPath: cachedMetadataPath,
		Size:         cacheSize,
	}

	// Save cache entry metadata
	if err := saveCacheEntry(cacheDir, cacheEntry); err != nil {
		slog.WarnContext(ctx, "failed to save cache metadata", "error", err)
	}

	slog.InfoContext(ctx, "successfully cached container image",
		"image", opts.ImageRef,
		"cache_dir", cacheDir,
		"size_mb", cacheSize/1024/1024,
		"expires_at", cacheEntry.ExpiresAt)

	// Clean up temp directory
	os.RemoveAll(tempDir)

	// Create new device pointing to cached rootfs
	return loadFromCache(ctx, cacheEntry, opts)
}

// loadFromCache loads a virtio device from cached data
func loadFromCache(ctx context.Context, cacheEntry *CacheEntry, opts ContainerToVirtioOptions) (virtio.VirtioDevice, *v1.Image, error) {
	// Load metadata from cache
	metadataData, err := os.ReadFile(cacheEntry.MetadataPath)
	if err != nil {
		return nil, nil, errors.Errorf("reading cached metadata: %w", err)
	}

	var metadata v1.Image
	if err := json.Unmarshal(metadataData, &metadata); err != nil {
		return nil, nil, errors.Errorf("unmarshaling cached metadata: %w", err)
	}

	// Set up mount point if specified
	if opts.MountPoint != "" {
		// Create mount point directory
		if err := os.MkdirAll(opts.MountPoint, 0755); err != nil {
			return nil, nil, errors.Errorf("creating mount point: %w", err)
		}

		// Create mount manager for cached rootfs
		mountMng := NewContainerdMountManager(cacheEntry.RootfsPath, opts.MountPoint, opts.ReadOnly, opts.ImageRef)

		// Mount the cached filesystem
		if err := mountMng.Mount(ctx); err != nil {
			return nil, nil, errors.Errorf("mounting cached filesystem: %w", err)
		}

		// Create virtio device pointing to mount point
		device, err := virtio.VirtioFsNew(opts.MountPoint, "rootfs")
		if err != nil {
			mountMng.Unmount(ctx)
			return nil, nil, errors.Errorf("creating virtio device from mount: %w", err)
		}

		return device, &metadata, nil
	}

	// Create virtio device pointing directly to cached rootfs
	device, err := virtio.VirtioFsNew(cacheEntry.RootfsPath, "rootfs")
	if err != nil {
		return nil, nil, errors.Errorf("creating virtio device from cache: %w", err)
	}

	slog.InfoContext(ctx, "successfully loaded container from cache",
		"image", opts.ImageRef,
		"rootfs_path", cacheEntry.RootfsPath,
		"cached_at", cacheEntry.CachedAt)

	return device, &metadata, nil
}

// getPlatformString converts a SystemContext to a string representation
func getPlatformString(platform *types.SystemContext) string {
	if platform == nil {
		return "linux-amd64"
	}

	os := platform.OSChoice
	if os == "" {
		os = "linux"
	}

	arch := platform.ArchitectureChoice
	if arch == "" {
		arch = "amd64"
	}

	return fmt.Sprintf("%s-%s", os, arch)
}

// ClearCache removes all cached OCI images
func ClearCache(ctx context.Context) error {
	cacheDir, err := getCacheDir()
	if err != nil {
		return errors.Errorf("getting cache directory: %w", err)
	}

	slog.InfoContext(ctx, "clearing OCI image cache", "cache_dir", cacheDir)

	if err := os.RemoveAll(cacheDir); err != nil {
		return errors.Errorf("removing cache directory: %w", err)
	}

	slog.InfoContext(ctx, "successfully cleared OCI image cache")
	return nil
}

// ListCachedImages returns information about all cached images
func ListCachedImages(ctx context.Context) ([]*CacheEntry, error) {
	cacheDir, err := getCacheDir()
	if err != nil {
		return nil, errors.Errorf("getting cache directory: %w", err)
	}

	entries, err := os.ReadDir(cacheDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*CacheEntry{}, nil
		}
		return nil, errors.Errorf("reading cache directory: %w", err)
	}

	var cachedImages []*CacheEntry

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		imageCacheDir := filepath.Join(cacheDir, entry.Name())
		cacheEntry, err := loadCacheEntry(imageCacheDir)
		if err != nil {
			slog.WarnContext(ctx, "failed to load cache entry", "dir", entry.Name(), "error", err)
			continue
		}

		if cacheEntry != nil {
			cachedImages = append(cachedImages, cacheEntry)
		}
	}

	return cachedImages, nil
}

// CleanExpiredCache removes expired cache entries
func CleanExpiredCache(ctx context.Context) error {
	cachedImages, err := ListCachedImages(ctx)
	if err != nil {
		return errors.Errorf("listing cached images: %w", err)
	}

	var removedCount int
	var reclaimedSize int64

	for _, entry := range cachedImages {
		if time.Now().After(entry.ExpiresAt) {
			slog.InfoContext(ctx, "removing expired cache entry",
				"image", entry.ImageRef,
				"expired_since", time.Since(entry.ExpiresAt),
				"size_mb", entry.Size/1024/1024)

			// Remove the entire cache directory for this image
			cacheDir := filepath.Dir(entry.RootfsPath)
			if err := os.RemoveAll(cacheDir); err != nil {
				slog.WarnContext(ctx, "failed to remove expired cache", "dir", cacheDir, "error", err)
			} else {
				removedCount++
				reclaimedSize += entry.Size
			}
		}
	}

	slog.InfoContext(ctx, "cleaned expired cache entries",
		"removed_count", removedCount,
		"reclaimed_mb", reclaimedSize/1024/1024)

	return nil
}
