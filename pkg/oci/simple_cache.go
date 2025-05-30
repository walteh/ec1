package oci

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gitlab.com/tozd/go/errors"

	v1 "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/walteh/ec1/pkg/units"
)

// DefaultSimpleCache creates a cache with default production implementations
func DefaultSimpleCache() *SimpleCache {
	return NewSimpleCache("/tmp/ec1-oci-cache", &SkopeoImageDownloader{}, &OCIImageExtractor{})
}

// ImageDownloader downloads images to OCI layout format
// This is mainly for testing and standalone usage - in containerd integration,
// containerd handles the downloading and provides us with the OCI layout path
type ImageDownloader interface {
	// DownloadImage downloads an image to OCI layout format
	// imageRef: image reference like "alpine:3.21" or "docker.io/library/alpine:3.21"
	// Returns: path to the downloaded OCI layout directory
	DownloadImage(ctx context.Context, imageRef string) (string, error)
}

// SimpleCache manages cached container images for fast VM creation
type SimpleCache struct {
	cacheDir   string
	downloader ImageDownloader // Optional - only used for standalone mode
	extractor  ImageExtractor
	mu         sync.RWMutex
}

// CachedImage represents a cached container image ready for VM use
type CachedImage struct {
	ImageRef   string
	Platform   units.Platform
	RootfsPath string
	Ext4Path   string
	Metadata   *v1.Image
	CachedAt   time.Time
}

// SimpleCacheEntry represents the JSON structure stored in cache metadata
type SimpleCacheEntry struct {
	ImageRef    string    `json:"image_ref"`
	Platform    string    `json:"platform"`
	RootfsPath  string    `json:"rootfs_path"`
	Ext4Path    string    `json:"ext4_path"`
	CachedAt    time.Time `json:"cached_at"`
	MetadataRaw []byte    `json:"metadata_raw"`
}

// NewSimpleCache creates a new cache instance
func NewSimpleCache(cacheDir string, downloader ImageDownloader, extractor ImageExtractor) *SimpleCache {
	return &SimpleCache{
		cacheDir:   cacheDir,
		downloader: downloader,
		extractor:  extractor,
	}
}

// LoadImage loads an image, using cache if available or downloading/extracting if needed
// For containerd integration, use LoadImageFromOCILayout instead
func (c *SimpleCache) LoadImage(ctx context.Context, imageRef string, platform units.Platform) (*CachedImage, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Try to load from cache first
	if cached, err := c.loadFromCache(ctx, imageRef, platform); err == nil {
		slog.InfoContext(ctx, "loaded image from cache", "image", imageRef, "platform", string(platform))
		return cached, nil
	}

	// Download and cache
	return c.downloadAndCache(ctx, imageRef, platform)
}

// LoadImageFromOCILayout loads an image from an existing OCI layout path (containerd integration)
// This is the main method for containerd integration where containerd provides the OCI layout
func (c *SimpleCache) LoadImageFromOCILayout(ctx context.Context, ociLayoutPath string, imageRef string, platform units.Platform) (*CachedImage, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Try to load from cache first
	if cached, err := c.loadFromCache(ctx, imageRef, platform); err == nil {
		slog.InfoContext(ctx, "loaded image from cache", "image", imageRef, "platform", string(platform))
		return cached, nil
	}

	// Extract from provided OCI layout and cache
	return c.extractAndCache(ctx, ociLayoutPath, imageRef, platform)
}

// loadFromCache attempts to load a cached image
func (c *SimpleCache) loadFromCache(ctx context.Context, imageRef string, platform units.Platform) (*CachedImage, error) {
	cacheKey := c.getCacheKey(imageRef, platform)
	metadataPath := filepath.Join(c.cacheDir, cacheKey, "metadata.json")

	data, err := os.ReadFile(metadataPath)
	if err != nil {
		return nil, errors.Errorf("reading cache metadata: %w", err)
	}

	var entry SimpleCacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, errors.Errorf("unmarshaling cache metadata: %w", err)
	}

	// Verify files exist
	if _, err := os.Stat(entry.RootfsPath); err != nil {
		return nil, errors.Errorf("cached rootfs not found: %w", err)
	}
	if _, err := os.Stat(entry.Ext4Path); err != nil {
		return nil, errors.Errorf("cached ext4 not found: %w", err)
	}

	// Unmarshal metadata
	var metadata v1.Image
	if err := json.Unmarshal(entry.MetadataRaw, &metadata); err != nil {
		return nil, errors.Errorf("unmarshaling image metadata: %w", err)
	}

	return &CachedImage{
		ImageRef:   entry.ImageRef,
		Platform:   platform,
		RootfsPath: entry.RootfsPath,
		Ext4Path:   entry.Ext4Path,
		Metadata:   &metadata,
		CachedAt:   entry.CachedAt,
	}, nil
}

// downloadAndCache downloads an image and caches the result (standalone mode)
func (c *SimpleCache) downloadAndCache(ctx context.Context, imageRef string, platform units.Platform) (*CachedImage, error) {
	if c.downloader == nil {
		return nil, errors.New("no downloader configured - use LoadImageFromOCILayout for containerd integration")
	}

	slog.InfoContext(ctx, "downloading image", "image", imageRef, "platform", string(platform))

	// Download to OCI layout
	ociLayoutPath, err := c.downloader.DownloadImage(ctx, imageRef)
	if err != nil {
		return nil, errors.Errorf("downloading image: %w", err)
	}

	return c.extractAndCache(ctx, ociLayoutPath, imageRef, platform)
}

// extractAndCache extracts an OCI layout and caches the result
func (c *SimpleCache) extractAndCache(ctx context.Context, ociLayoutPath string, imageRef string, platform units.Platform) (*CachedImage, error) {
	cacheKey := c.getCacheKey(imageRef, platform)
	cacheDir := filepath.Join(c.cacheDir, cacheKey)

	// Create cache directory
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, errors.Errorf("creating cache directory: %w", err)
	}

	slog.InfoContext(ctx, "extracting image to cache", "image", imageRef, "platform", string(platform), "cache_dir", cacheDir)

	// Extract the image
	metadata, err := c.extractor.ExtractImage(ctx, ociLayoutPath, platform, cacheDir)
	if err != nil {
		return nil, errors.Errorf("extracting image: %w", err)
	}

	// Prepare cache entry
	rootfsPath := filepath.Join(cacheDir, "rootfs")
	ext4Path := filepath.Join(cacheDir, "rootfs.ext4")

	metadataRaw, err := json.Marshal(metadata)
	if err != nil {
		return nil, errors.Errorf("marshaling metadata: %w", err)
	}

	entry := SimpleCacheEntry{
		ImageRef:    imageRef,
		Platform:    string(platform),
		RootfsPath:  rootfsPath,
		Ext4Path:    ext4Path,
		CachedAt:    time.Now(),
		MetadataRaw: metadataRaw,
	}

	// Save cache metadata
	entryData, err := json.Marshal(entry)
	if err != nil {
		return nil, errors.Errorf("marshaling cache entry: %w", err)
	}

	metadataPath := filepath.Join(cacheDir, "metadata.json")
	if err := os.WriteFile(metadataPath, entryData, 0644); err != nil {
		return nil, errors.Errorf("writing cache metadata: %w", err)
	}

	slog.InfoContext(ctx, "image cached successfully", "image", imageRef, "platform", string(platform), "cache_dir", cacheDir)

	return &CachedImage{
		ImageRef:   imageRef,
		Platform:   platform,
		RootfsPath: rootfsPath,
		Ext4Path:   ext4Path,
		Metadata:   metadata,
		CachedAt:   entry.CachedAt,
	}, nil
}

// getCacheKey generates a cache key for an image and platform
func (c *SimpleCache) getCacheKey(imageRef string, platform units.Platform) string {
	// Replace invalid characters for filesystem paths
	key := fmt.Sprintf("%s_%s", imageRef, string(platform))
	key = strings.ReplaceAll(key, "/", "_")
	key = strings.ReplaceAll(key, ":", "_")
	key = strings.ReplaceAll(key, "@", "_")
	return key
}

// ClearCache removes all cached images
func (c *SimpleCache) ClearCache(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := os.RemoveAll(c.cacheDir); err != nil {
		return errors.Errorf("removing cache directory: %w", err)
	}

	return os.MkdirAll(c.cacheDir, 0755)
}

// ListCachedImages returns all cached images
func (c *SimpleCache) ListCachedImages(ctx context.Context) ([]*CachedImage, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var images []*CachedImage

	entries, err := os.ReadDir(c.cacheDir)
	if err != nil {
		if os.IsNotExist(err) {
			return images, nil
		}
		return nil, errors.Errorf("reading cache directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		metadataPath := filepath.Join(c.cacheDir, entry.Name(), "metadata.json")
		data, err := os.ReadFile(metadataPath)
		if err != nil {
			slog.WarnContext(ctx, "skipping invalid cache entry", "entry", entry.Name(), "error", err)
			continue
		}

		var cacheEntry SimpleCacheEntry
		if err := json.Unmarshal(data, &cacheEntry); err != nil {
			slog.WarnContext(ctx, "skipping invalid cache metadata", "entry", entry.Name(), "error", err)
			continue
		}

		var metadata v1.Image
		if err := json.Unmarshal(cacheEntry.MetadataRaw, &metadata); err != nil {
			slog.WarnContext(ctx, "skipping invalid image metadata", "entry", entry.Name(), "error", err)
			continue
		}

		platform, err := units.ParsePlatform(cacheEntry.Platform)
		if err != nil {
			slog.WarnContext(ctx, "skipping invalid platform", "entry", entry.Name(), "platform", cacheEntry.Platform, "error", err)
			continue
		}

		images = append(images, &CachedImage{
			ImageRef:   cacheEntry.ImageRef,
			Platform:   platform,
			RootfsPath: cacheEntry.RootfsPath,
			Ext4Path:   cacheEntry.Ext4Path,
			Metadata:   &metadata,
			CachedAt:   cacheEntry.CachedAt,
		})
	}

	return images, nil
}

// CleanExpiredCache removes cached images older than the specified duration
func (c *SimpleCache) CleanExpiredCache(ctx context.Context, expiration time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	cutoff := time.Now().Add(-expiration)

	entries, err := os.ReadDir(c.cacheDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return errors.Errorf("reading cache directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		metadataPath := filepath.Join(c.cacheDir, entry.Name(), "metadata.json")
		data, err := os.ReadFile(metadataPath)
		if err != nil {
			continue
		}

		var cacheEntry SimpleCacheEntry
		if err := json.Unmarshal(data, &cacheEntry); err != nil {
			continue
		}

		if cacheEntry.CachedAt.Before(cutoff) {
			entryDir := filepath.Join(c.cacheDir, entry.Name())
			if err := os.RemoveAll(entryDir); err != nil {
				slog.WarnContext(ctx, "failed to remove expired cache entry", "entry", entry.Name(), "error", err)
			} else {
				slog.InfoContext(ctx, "removed expired cache entry", "entry", entry.Name(), "cached_at", cacheEntry.CachedAt)
			}
		}
	}

	return nil
}

// GetCacheSize returns the total size of the cache directory in bytes
func (c *SimpleCache) GetCacheSize(ctx context.Context) (int64, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var totalSize int64

	err := filepath.Walk(c.cacheDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})

	return totalSize, err
}

// Global cache instances for convenience
var (
	defaultCache *SimpleCache
	cacheMu      sync.Mutex
)

// SetDefaultCache sets the default cache instance
func SetDefaultCache(cache *SimpleCache) {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	defaultCache = cache
}

// ClearAllCaches clears both default and test caches
func ClearAllCaches(ctx context.Context) error {
	cacheMu.Lock()
	defer cacheMu.Unlock()

	var errs []error

	if defaultCache != nil {
		if err := defaultCache.ClearCache(ctx); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errors.Errorf("clearing caches: %v", errs)
	}

	return nil
}

// ListAllCachedImages lists images from both default and test caches
func ListAllCachedImages(ctx context.Context) ([]*CachedImage, error) {
	cacheMu.Lock()
	defer cacheMu.Unlock()

	var allImages []*CachedImage

	if defaultCache != nil {
		images, err := defaultCache.ListCachedImages(ctx)
		if err != nil {
			return nil, errors.Errorf("listing default cache: %w", err)
		}
		allImages = append(allImages, images...)
	}

	return allImages, nil
}
