package oci

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Microsoft/hcsshim/ext4/tar2ext4"
	"github.com/containers/image/v5/docker"
	"github.com/containers/image/v5/signature"
	"github.com/containers/image/v5/types"
	"github.com/mholt/archives"
	"gitlab.com/tozd/go/errors"

	v1 "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/walteh/ec1/pkg/units"
)

// CacheConfig holds configuration for the cache system
type CacheConfig struct {
	CacheDir   string
	Expiration time.Duration
}

// ImageDownloader interface abstracts image downloading operations for testing
// This is the part that hits the network and can be stubbed out
type ImageDownloader interface {
	DownloadImage(ctx context.Context, imageRef string) (ociLayoutPathDir string, err error)
}

// PlatformFetcher interface abstracts platform fetching operations for testing
type PlatformFetcher interface {
	FetchAvailablePlatforms(ctx context.Context, imageRef string) ([]*PlatformInfo, error)
}

// CacheV2 represents the v2 cache system with dependency injection
type CacheV2 struct {
	config          CacheConfig
	imageDownloader ImageDownloader
}

// NewCacheV2 creates a new cache instance with injected dependencies
func NewCacheV2(config CacheConfig, imageDownloader ImageDownloader) *CacheV2 {
	return &CacheV2{
		config:          config,
		imageDownloader: imageDownloader,
	}
}

// DefaultCacheV2 creates a cache with default implementations
func DefaultCacheV2() *CacheV2 {
	cacheDir, _ := getCacheDirV2()
	config := CacheConfig{
		CacheDir:   cacheDir,
		Expiration: CacheExpirationV2,
	}
	return NewCacheV2(config, &SkopeoImageDownloader{})
}

// ImageCacheMetadata represents metadata for a cached image reference
// This is stored per image reference (e.g., "alpine:latest") and tracks
// all available platforms and their freshness
type ImageCacheMetadata struct {
	ImageRef  string    `json:"image_ref"`
	CachedAt  time.Time `json:"cached_at"`
	ExpiresAt time.Time `json:"expires_at"`
	// Platforms     map[string]*PlatformManifest  `json:"platforms"`      // platform string -> manifest info
	ManifestCache map[string]*ManifestCacheInfo `json:"manifest_cache"` // manifest digest -> cache info
}

// PlatformManifest represents a specific platform's manifest information
type PlatformManifest struct {
	Platform       units.Platform `json:"platform"`
	ManifestDigest string         `json:"manifest_digest"`
	Size           int64          `json:"size"`
	LastAccessed   time.Time      `json:"last_accessed"`
}

// ManifestCacheInfo represents cached filesystem content for a specific manifest
type ManifestCacheInfo struct {
	ManifestDigest string    `json:"manifest_digest"`
	RootfsPath     string    `json:"rootfs_path"`
	OciLayoutPath  string    `json:"oci_layout_path"`
	MetadataPath   string    `json:"metadata_path"`
	RootfsDiskPath string    `json:"rootfs_disk_path"`
	Size           int64     `json:"size"`
	CachedAt       time.Time `json:"cached_at"`
	PlatformString string    `json:"platform_string"`
}

// CachedContainerV2 represents a cached container with v2 cache system
type CachedContainerV2 struct {
	ImageRef         string
	Platform         units.Platform
	ManifestDigest   string
	ReadonlyFSPath   string
	ReadonlyExt4Path string
	Metadata         *v1.Image
}

const (
	// CacheExpirationV2 is how long cached image metadata is valid
	CacheExpirationV2 = 24 * time.Hour

	// ImageMetadataFile is the name of the image metadata file
	ImageMetadataFile = "image-metadata.json"

	// ManifestMetadataFile is the name of the manifest metadata file
	ManifestMetadataFile = "manifest-metadata.json"
)

// getCacheDirV2 returns the base cache directory for v2 cache
func getCacheDirV2() (string, error) {
	userCacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", errors.Errorf("getting user cache dir: %w", err)
	}
	return filepath.Join(userCacheDir, "ec1", "cache", "oci-v2"), nil
}

// getImageCacheDir returns the cache directory for a specific image
func (c *CacheV2) getImageCacheDir(imageRef string) (string, error) {
	// Create a safe directory name from the image reference
	safeName := strings.ReplaceAll(imageRef, "/", "_")
	safeName = strings.ReplaceAll(safeName, ":", "_")

	imageDir := filepath.Join(c.config.CacheDir, "images", safeName)
	if err := os.MkdirAll(imageDir, 0755); err != nil {
		return "", errors.Errorf("creating image cache directory: %w", err)
	}

	return imageDir, nil
}

// getManifestCacheDir returns the cache directory for a specific manifest
func (c *CacheV2) getManifestCacheDir(manifestDigest string) (string, error) {
	// Replace colons with underscores for filesystem compatibility
	safeDigest := strings.ReplaceAll(manifestDigest, ":", "_")

	manifestDir := filepath.Join(c.config.CacheDir, "manifests", safeDigest)
	if err := os.MkdirAll(manifestDir, 0755); err != nil {
		return "", errors.Errorf("creating manifest cache directory: %w", err)
	}

	return manifestDir, nil
}

// loadImageCacheMetadata loads image cache metadata from disk
func (c *CacheV2) loadImageCacheMetadata(imageRef string) (*ImageCacheMetadata, error) {
	imageDir, err := c.getImageCacheDir(imageRef)
	if err != nil {
		return nil, err
	}

	metadataPath := filepath.Join(imageDir, "metadata.json")
	data, err := os.ReadFile(metadataPath)
	if err != nil {
		return nil, errors.Errorf("reading image metadata: %w", err)
	}

	var metadata ImageCacheMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, errors.Errorf("parsing image metadata: %w", err)
	}

	return &metadata, nil
}

// saveImageCacheMetadata saves image cache metadata to disk
func (c *CacheV2) saveImageCacheMetadata(metadata *ImageCacheMetadata) error {
	imageDir, err := c.getImageCacheDir(metadata.ImageRef)
	if err != nil {
		return err
	}

	metadataPath := filepath.Join(imageDir, "metadata.json")
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return errors.Errorf("marshaling image metadata: %w", err)
	}

	if err := os.WriteFile(metadataPath, data, 0644); err != nil {
		return errors.Errorf("writing image metadata: %w", err)
	}

	return nil
}

// loadManifestCacheInfo loads manifest cache info from disk
func (c *CacheV2) loadManifestCacheInfo(manifestDigest string) (*ManifestCacheInfo, error) {
	manifestDir, err := c.getManifestCacheDir(manifestDigest)
	if err != nil {
		return nil, err
	}

	infoPath := filepath.Join(manifestDir, "info.json")
	data, err := os.ReadFile(infoPath)
	if err != nil {
		return nil, errors.Errorf("reading manifest cache info: %w", err)
	}

	var info ManifestCacheInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, errors.Errorf("parsing manifest cache info: %w", err)
	}

	return &info, nil
}

// saveManifestCacheInfo saves manifest cache info to disk
func (c *CacheV2) saveManifestCacheInfo(info *ManifestCacheInfo) error {
	manifestDir, err := c.getManifestCacheDir(info.ManifestDigest)
	if err != nil {
		return err
	}

	infoPath := filepath.Join(manifestDir, "info.json")
	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return errors.Errorf("marshaling manifest cache info: %w", err)
	}

	if err := os.WriteFile(infoPath, data, 0644); err != nil {
		return errors.Errorf("writing manifest cache info: %w", err)
	}

	return nil
}

// isImageCacheValid checks if image cache metadata is still valid
func (c *CacheV2) isImageCacheValid(metadata *ImageCacheMetadata) bool {
	if metadata == nil {
		return false
	}

	// Check if cache has expired
	if time.Now().After(metadata.ExpiresAt) {
		return false
	}

	// Check if platform support has changed since caching
	// This is handled separately in shouldRefreshForPlatformSupport

	return true
}

// shouldRefreshForPlatformSupport checks if we need to refresh due to platform support changes
func (c *CacheV2) shouldRefreshForPlatformSupport(metadata *ImageCacheMetadata, requestedPlatform units.Platform) bool {
	if metadata == nil {
		return true
	}

	// If the requested platform is not in our cached platforms, we need to refresh
	platformStr := string(requestedPlatform)
	if _, exists := metadata.Platforms[platformStr]; !exists {
		return true
	}

	// Check if any cached platforms are no longer supported
	// This could happen if the supported platforms list changed
	for cachedPlatformStr := range metadata.Platforms {
		cachedPlatform := units.Platform(cachedPlatformStr)
		if !cachedPlatform.IsSupported() {
			return true
		}
	}

	return false
}

// isManifestCacheValid checks if manifest cache info is still valid
func (c *CacheV2) isManifestCacheValid(info *ManifestCacheInfo) bool {
	if info == nil {
		return false
	}

	// Check if all required files exist
	requiredPaths := []string{
		info.RootfsPath,
		info.OciLayoutPath,
		info.MetadataPath,
		info.RootfsDiskPath,
	}

	for _, path := range requiredPaths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return false
		}
	}

	return true
}

// LoadCachedContainerV2 loads a container from the v2 cache system
func (c *CacheV2) LoadCachedContainer(ctx context.Context, imageRef string, platform units.Platform) (*CachedContainerV2, error) {
	slog.InfoContext(ctx, "loading container with v2 cache system",
		"image", imageRef,
		"platform", string(platform),
		"cache_expiration", c.config.Expiration)

	// Early validation: check if requested platform is supported
	if !platform.IsSupported() {
		return nil, errors.Errorf("requested platform %s is not supported", string(platform))
	}

	// Try to load existing image metadata
	imageMetadata, err := c.loadImageCacheMetadata(imageRef)
	cacheExists := err == nil
	cacheValid := cacheExists && c.isImageCacheValid(imageMetadata)

	// Check if we need to refresh due to platform support changes
	needsPlatformRefresh := cacheExists && c.shouldRefreshForPlatformSupport(imageMetadata, platform)

	slog.InfoContext(ctx, "image metadata cache status",
		"image", imageRef,
		"cache_exists", cacheExists,
		"cache_valid", cacheValid,
		"needs_platform_refresh", needsPlatformRefresh)

	// Refresh metadata if cache is invalid, missing, or needs platform refresh
	if !cacheValid || needsPlatformRefresh {
		slog.InfoContext(ctx, "image metadata cache miss, expired, or needs platform refresh - refreshing",
			"image", imageRef,
			"cache_exists", cacheExists,
			"cache_valid", cacheValid,
			"needs_platform_refresh", needsPlatformRefresh)

		imageMetadata, err = c.refreshImageMetadata(ctx, imageRef, platform)
		if err != nil {
			return nil, errors.Errorf("failed to refresh image metadata and no cache available: %w", err)
		}
	}

	// Get platform-specific manifest information
	platformStr := string(platform)
	platformManifest, exists := imageMetadata.Platforms[platformStr]
	if !exists {
		return nil, errors.Errorf("platform %s not found in cached metadata for image %s", platformStr, imageRef)
	}

	// Check if manifest content is cached and valid
	manifestInfo, err := c.loadManifestCacheInfo(platformManifest.ManifestDigest)
	if err != nil || !c.isManifestCacheValid(manifestInfo) {
		slog.InfoContext(ctx, "manifest content cache miss or invalid, downloading",
			"manifest_digest", platformManifest.ManifestDigest,
			"platform", platformStr)

		manifestInfo, err = c.downloadAndCacheManifest(ctx, imageRef, platform, platformManifest.ManifestDigest)
		if err != nil {
			return nil, errors.Errorf("downloading and caching manifest: %w", err)
		}

		// Update last accessed time for the platform
		platformManifest.LastAccessed = time.Now()
		imageMetadata.Platforms[platformStr] = platformManifest
		if err := c.saveImageCacheMetadata(imageMetadata); err != nil {
			slog.WarnContext(ctx, "failed to update platform last accessed time", "error", err)
		}
	}

	// Load image metadata from cached manifest
	metadataPath := manifestInfo.MetadataPath
	metadataBytes, err := os.ReadFile(metadataPath)
	if err != nil {
		return nil, errors.Errorf("reading cached metadata: %w", err)
	}

	var metadata v1.Image
	if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
		return nil, errors.Errorf("parsing cached metadata: %w", err)
	}

	slog.InfoContext(ctx, "successfully loaded container from v2 cache",
		"image", imageRef,
		"platform", platformStr,
		"manifest_digest", platformManifest.ManifestDigest,
		"rootfs_disk_path", manifestInfo.RootfsDiskPath,
		"cached_at", manifestInfo.CachedAt)

	return &CachedContainerV2{
		ImageRef:         imageRef,
		Platform:         platform,
		ManifestDigest:   platformManifest.ManifestDigest,
		ReadonlyFSPath:   manifestInfo.RootfsPath,
		ReadonlyExt4Path: manifestInfo.RootfsDiskPath,
		Metadata:         &metadata,
	}, nil
}

// Backward compatibility function
func LoadCachedContainerV2(ctx context.Context, imageRef string, platform units.Platform) (*CachedContainerV2, error) {
	cache := DefaultCacheV2()
	return cache.LoadCachedContainer(ctx, imageRef, platform)
}

// refreshImageMetadata refreshes image metadata by pulling manifest information for all platforms
func (c *CacheV2) refreshImageMetadata(ctx context.Context, imageRef string, requestedPlatform units.Platform) (*ImageCacheMetadata, error) {
	slog.InfoContext(ctx, "refreshing image metadata", "image", imageRef, "requested_platform", string(requestedPlatform))

	// Create policy context
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

	// Create image source to inspect the manifest
	imgSrc, err := srcRef.NewImageSource(ctx, &types.SystemContext{})
	if err != nil {
		return nil, errors.Errorf("creating image source: %w", err)
	}
	defer imgSrc.Close()

	// Get the raw manifest to check if it's a manifest list
	manifestBlob, manifestType, err := imgSrc.GetManifest(ctx, nil)
	if err != nil {
		return nil, errors.Errorf("getting manifest: %w", err)
	}

	var manifest v1.Manifest
	if err := json.Unmarshal(manifestBlob, &manifest); err != nil {
		return nil, errors.Errorf("unmarshalling manifest: %w", err)
	}

	// Create new metadata
	metadata := &ImageCacheMetadata{
		ImageRef:      imageRef,
		CachedAt:      time.Now(),
		ExpiresAt:     time.Now().Add(c.config.Expiration),
		Platforms:     make(map[string]*PlatformManifest),
		ManifestCache: make(map[string]*ManifestCacheInfo),
	}

	// Add platform information
	for _, platform := range platforms {
		platformStr := string(platform.Platform)
		metadata.Platforms[platformStr] = &PlatformManifest{
			Platform:       platform.Platform,
			ManifestDigest: platform.ManifestDigest,
			Size:           platform.Size,
			LastAccessed:   time.Now(),
		}
	}

	// Save metadata
	if err := c.saveImageCacheMetadata(metadata); err != nil {
		return nil, errors.Errorf("saving image metadata: %w", err)
	}

	slog.InfoContext(ctx, "successfully refreshed image metadata",
		"image", imageRef,
		"platforms_cached", len(metadata.Platforms))

	return metadata, nil
}

// downloadAndCacheManifest downloads and caches manifest content for a specific platform
func (c *CacheV2) downloadAndCacheManifest(ctx context.Context, imageRef string, platform units.Platform, manifestDigest string) (*ManifestCacheInfo, error) {
	slog.InfoContext(ctx, "downloading and caching manifest",
		"image", imageRef,
		"platform", string(platform),
		"manifest_digest", manifestDigest)

	// Get manifest cache directory
	manifestDir, err := c.getManifestCacheDir(manifestDigest)
	if err != nil {
		return nil, err
	}

	// Create subdirectories for different components
	rootfsPath := filepath.Join(manifestDir, "rootfs")
	metadataPath := filepath.Join(manifestDir, "metadata.json")

	// Pull and extract the image
	// ociLayoutPath, err := c.imageDownloader.DownloadImage(ctx, imageRef, []digest.Digest{digest.Digest(manifestDigest)})
	ociLayoutPath, err := c.imageDownloader.DownloadImage(ctx, imageRef)
	if err != nil {
		return nil, errors.Errorf("downloading image: %w", err)
	}

	// Extract metadata from the downloaded image
	metadata, err := ExtractOCIImageToDir(ctx, ociLayoutPath, rootfsPath, platform)
	if err != nil {
		return nil, errors.Errorf("extracting image metadata: %w", err)
	}

	// Save metadata
	metadataBytes, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return nil, errors.Errorf("marshaling metadata: %w", err)
	}

	if err := os.WriteFile(metadataPath, metadataBytes, 0644); err != nil {
		return nil, errors.Errorf("writing metadata: %w", err)
	}

	// Convert to ext4 disk image
	ext4Path, err := bundleToExt4(ctx, ociLayoutPath, manifestDir)
	if err != nil {
		return nil, errors.Errorf("converting to ext4: %w", err)
	}

	// Calculate size
	size, err := calculateDirectorySize(manifestDir)
	if err != nil {
		slog.WarnContext(ctx, "failed to calculate manifest cache size", "error", err)
		size = 0
	}

	// Create manifest cache info
	info := &ManifestCacheInfo{
		ManifestDigest: manifestDigest,
		RootfsPath:     rootfsPath,
		OciLayoutPath:  ociLayoutPath,
		MetadataPath:   metadataPath,
		RootfsDiskPath: ext4Path,
		Size:           size,
		CachedAt:       time.Now(),
		PlatformString: string(platform),
	}

	// Save manifest cache info
	if err := c.saveManifestCacheInfo(info); err != nil {
		return nil, errors.Errorf("saving manifest cache info: %w", err)
	}

	slog.InfoContext(ctx, "successfully cached manifest",
		"manifest_digest", manifestDigest,
		"size_mb", size/1024/1024,
		"ext4_path", ext4Path)

	return info, nil
}

// calculateDirectorySize calculates the total size of a directory
func calculateDirectorySize(dirPath string) (int64, error) {
	var size int64
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
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

// bundleToExt4 bundles a rootfs directory into an ext4 disk image
func bundleToExt4(ctx context.Context, rootfsPath string, cacheDir string) (string, error) {
	rootfsDiskPath := filepath.Join(cacheDir, "fs.ext4.img")

	fles, err := archives.FilesFromDisk(ctx, &archives.FromDiskOptions{}, map[string]string{
		rootfsPath: rootfsDiskPath,
	})
	if err != nil {
		return "", errors.Errorf("getting files from disk: %w", err)
	}

	os.Remove(rootfsDiskPath)

	// Pack the folder into a tar stream
	pr, pw := io.Pipe()
	go func() {
		defer pw.Close()
		err := (&archives.Tar{}).Archive(ctx, pw, fles)
		if err != nil {
			slog.ErrorContext(ctx, "failed to archive files", "error", err)
		}
	}()

	out, err := os.Create(rootfsDiskPath)
	if err != nil {
		return "", errors.Errorf("creating rootfs disk file: %w", err)
	}
	defer out.Close()

	err = tar2ext4.ConvertTarToExt4(pr, out, tar2ext4.MaximumDiskSize(1<<30), tar2ext4.InlineData) // 1 GiB
	if err != nil {
		return "", errors.Errorf("converting tar to ext4: %w", err)
	}

	return rootfsDiskPath, nil
}

// platformToSystemContext converts a units.Platform to a types.SystemContext
func platformToSystemContext(platform units.Platform) *types.SystemContext {
	return &types.SystemContext{
		OSChoice:           platform.OS(),
		ArchitectureChoice: platform.Arch(),
	}
}

// CleanExpiredCacheV2 removes expired cache entries and orphaned manifests
func (c *CacheV2) CleanExpiredCache(ctx context.Context) error {
	slog.InfoContext(ctx, "cleaning expired v2 cache entries")

	// List all cached images
	cachedImages, err := c.ListCachedImages(ctx)
	if err != nil {
		return errors.Errorf("listing cached images: %w", err)
	}

	var removedImageCount int
	var removedManifestCount int
	var reclaimedSize int64

	// Collect all referenced manifest digests
	referencedManifests := make(map[string]bool)

	// Clean expired image metadata and collect referenced manifests
	for _, metadata := range cachedImages {
		if time.Now().After(metadata.ExpiresAt) {
			slog.InfoContext(ctx, "removing expired image metadata",
				"image", metadata.ImageRef,
				"expired_since", time.Since(metadata.ExpiresAt))

			imageCacheDir, err := c.getImageCacheDir(metadata.ImageRef)
			if err != nil {
				slog.WarnContext(ctx, "failed to get image cache dir", "image", metadata.ImageRef, "error", err)
				continue
			}

			if err := os.RemoveAll(imageCacheDir); err != nil {
				slog.WarnContext(ctx, "failed to remove expired image cache", "dir", imageCacheDir, "error", err)
			} else {
				removedImageCount++
			}
		} else {
			// Collect manifest digests that are still referenced
			for _, platform := range metadata.Platforms {
				referencedManifests[platform.ManifestDigest] = true
			}
		}
	}

	// Clean orphaned manifest cache entries
	orphanedManifests, err := c.findOrphanedManifests(ctx, referencedManifests)
	if err != nil {
		slog.WarnContext(ctx, "failed to find orphaned manifests", "error", err)
	} else {
		for _, manifestDigest := range orphanedManifests {
			slog.InfoContext(ctx, "removing orphaned manifest cache",
				"manifest_digest", manifestDigest)

			manifestCacheDir, err := c.getManifestCacheDir(manifestDigest)
			if err != nil {
				slog.WarnContext(ctx, "failed to get manifest cache dir", "digest", manifestDigest, "error", err)
				continue
			}

			// Calculate size before removal
			if manifestInfo, err := c.loadManifestCacheInfo(manifestDigest); err == nil {
				reclaimedSize += manifestInfo.Size
			}

			if err := os.RemoveAll(manifestCacheDir); err != nil {
				slog.WarnContext(ctx, "failed to remove orphaned manifest cache", "dir", manifestCacheDir, "error", err)
			} else {
				removedManifestCount++
			}
		}
	}

	slog.InfoContext(ctx, "cleaned expired cache entries",
		"removed_images", removedImageCount,
		"removed_manifests", removedManifestCount,
		"reclaimed_mb", reclaimedSize/1024/1024)

	return nil
}

// ListCachedImages lists all cached images in the v2 cache
func (c *CacheV2) ListCachedImages(ctx context.Context) ([]*ImageCacheMetadata, error) {
	imagesDir := filepath.Join(c.config.CacheDir, "images")
	entries, err := os.ReadDir(imagesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*ImageCacheMetadata{}, nil
		}
		return nil, errors.Errorf("reading images cache directory: %w", err)
	}

	var cachedImages []*ImageCacheMetadata

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Try to load metadata for this image directory
		metadataPath := filepath.Join(imagesDir, entry.Name(), "metadata.json")
		data, err := os.ReadFile(metadataPath)
		if err != nil {
			slog.WarnContext(ctx, "failed to read image metadata", "path", metadataPath, "error", err)
			continue
		}

		var metadata ImageCacheMetadata
		if err := json.Unmarshal(data, &metadata); err != nil {
			slog.WarnContext(ctx, "failed to parse image metadata", "path", metadataPath, "error", err)
			continue
		}

		cachedImages = append(cachedImages, &metadata)
	}

	return cachedImages, nil
}

// findOrphanedManifests finds manifest cache entries that are no longer referenced by any image metadata
func (c *CacheV2) findOrphanedManifests(ctx context.Context, referencedManifests map[string]bool) ([]string, error) {
	manifestsDir := filepath.Join(c.config.CacheDir, "manifests")
	entries, err := os.ReadDir(manifestsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, errors.Errorf("reading manifests cache directory: %w", err)
	}

	var orphanedManifests []string

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		manifestDigest := entry.Name()

		// Check if this manifest is still referenced
		if !referencedManifests[manifestDigest] {
			// Verify it's actually a manifest cache directory by checking for info file
			manifestCacheDir := filepath.Join(manifestsDir, manifestDigest)
			infoPath := filepath.Join(manifestCacheDir, "info.json")

			if _, err := os.Stat(infoPath); err == nil {
				orphanedManifests = append(orphanedManifests, manifestDigest)
				slog.DebugContext(ctx, "found orphaned manifest", "digest", manifestDigest)
			}
		}
	}

	slog.InfoContext(ctx, "orphaned manifest scan complete",
		"total_manifests", len(entries),
		"referenced_manifests", len(referencedManifests),
		"orphaned_manifests", len(orphanedManifests))

	return orphanedManifests, nil
}

// Backward compatibility functions
func CleanExpiredCacheV2(ctx context.Context) error {
	cache := DefaultCacheV2()
	return cache.CleanExpiredCache(ctx)
}

func ListCachedImagesV2(ctx context.Context) ([]*ImageCacheMetadata, error) {
	cache := DefaultCacheV2()
	return cache.ListCachedImages(ctx)
}

func ClearCacheV2(ctx context.Context) error {
	cache := DefaultCacheV2()
	return cache.ClearCache(ctx)
}

// PlatformInfo represents information about a specific platform manifest
type PlatformInfo struct {
	Platform       units.Platform
	ManifestDigest string
	Size           int64
}

// fetchAvailablePlatforms fetches all available platforms for an image from the registry
func fetchAvailablePlatforms(ctx context.Context, imageRef string) ([]*PlatformInfo, error) {
	slog.InfoContext(ctx, "fetching available platforms from registry", "image", imageRef)

	// Create policy context
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

	// Create image source to inspect the manifest
	imgSrc, err := srcRef.NewImageSource(ctx, &types.SystemContext{})
	if err != nil {
		return nil, errors.Errorf("creating image source: %w", err)
	}
	defer imgSrc.Close()

	// Get the raw manifest to check if it's a manifest list
	manifestBlob, manifestType, err := imgSrc.GetManifest(ctx, nil)
	if err != nil {
		return nil, errors.Errorf("getting manifest: %w", err)
	}

	var platforms []*PlatformInfo

	// Check if this is a manifest list (multi-platform)
	if manifestType == "application/vnd.docker.distribution.manifest.list.v2+json" ||
		manifestType == "application/vnd.oci.image.index.v1+json" {

		slog.InfoContext(ctx, "found multi-platform manifest", "type", manifestType)

		// Parse the manifest list
		var manifestList struct {
			Manifests []struct {
				Digest   string `json:"digest"`
				Size     int64  `json:"size"`
				Platform struct {
					Architecture string `json:"architecture"`
					OS           string `json:"os"`
					Variant      string `json:"variant,omitempty"`
				} `json:"platform"`
			} `json:"manifests"`
		}

		if err := json.Unmarshal(manifestBlob, &manifestList); err != nil {
			return nil, errors.Errorf("parsing manifest list: %w", err)
		}

		// Extract platform information, but only for supported platforms
		for _, manifest := range manifestList.Manifests {
			platformStr := fmt.Sprintf("%s/%s", manifest.Platform.OS, manifest.Platform.Architecture)
			if manifest.Platform.Variant != "" {
				platformStr += "/" + manifest.Platform.Variant
			}

			// Resolve platform variants and check if supported
			resolvedPlatform := units.ResolvePlatformVariant(platformStr)
			if !resolvedPlatform.IsSupported() {
				slog.DebugContext(ctx, "skipping unsupported platform",
					"platform", platformStr,
					"resolved", string(resolvedPlatform))
				continue
			}

			platforms = append(platforms, &PlatformInfo{
				Platform:       resolvedPlatform,
				ManifestDigest: manifest.Digest,
				Size:           manifest.Size,
			})
		}
	} else {
		slog.InfoContext(ctx, "found single-platform manifest", "type", manifestType)

		// Single platform manifest - try to determine the platform
		// Default to host platform if it's supported, otherwise linux/amd64
		defaultPlatform := units.HostPlatform()
		if !defaultPlatform.IsSupported() {
			defaultPlatform = units.PlatformLinuxAMD64
		}

		// Calculate manifest digest
		hasher := sha256.New()
		hasher.Write(manifestBlob)
		manifestDigest := "sha256:" + hex.EncodeToString(hasher.Sum(nil))

		platforms = append(platforms, &PlatformInfo{
			Platform:       defaultPlatform,
			ManifestDigest: manifestDigest,
			Size:           int64(len(manifestBlob)),
		})
	}

	slog.InfoContext(ctx, "discovered supported platforms",
		"count", len(platforms),
		"image", imageRef)
	for _, platform := range platforms {
		slog.DebugContext(ctx, "available supported platform",
			"platform", string(platform.Platform),
			"digest", platform.ManifestDigest,
			"size", platform.Size)
	}

	return platforms, nil
}

// findCompatiblePlatform finds a compatible platform for the requested platform
func findCompatiblePlatform(requestedPlatform units.Platform, availablePlatforms []*PlatformInfo) *PlatformInfo {
	// Ensure the requested platform is supported
	if !requestedPlatform.IsSupported() {
		return nil
	}

	// First, try exact match
	for _, platform := range availablePlatforms {
		if platform.Platform == requestedPlatform {
			return platform
		}
	}

	// Try to find compatible platform with same OS and architecture
	requestedOS := requestedPlatform.OS()
	requestedArch := requestedPlatform.Arch()

	for _, platform := range availablePlatforms {
		if platform.Platform.OS() == requestedOS && platform.Platform.Arch() == requestedArch {
			return platform
		}
	}

	return nil
}

// generateManifestDigest generates a deterministic manifest digest for fallback cases
func generateManifestDigest(imageRef string, platform units.Platform, timestamp time.Time) string {
	hasher := sha256.New()
	hasher.Write([]byte(imageRef + string(platform) + timestamp.Format(time.RFC3339)))
	return "sha256:" + hex.EncodeToString(hasher.Sum(nil))
}

// ClearCacheV2 removes all cached data from v2 cache
func (c *CacheV2) ClearCache(ctx context.Context) error {
	slog.InfoContext(ctx, "clearing v2 OCI image cache", "cache_dir", c.config.CacheDir)

	if err := os.RemoveAll(c.config.CacheDir); err != nil {
		return errors.Errorf("removing cache directory: %w", err)
	}

	slog.InfoContext(ctx, "successfully cleared v2 OCI image cache")
	return nil
}
