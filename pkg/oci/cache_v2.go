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
	"github.com/containers/image/v5/types"
	"github.com/mholt/archives"
	"gitlab.com/tozd/go/errors"

	v1 "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/walteh/ec1/pkg/units"
)

// ImageCacheMetadata represents metadata for a cached image reference
// This is stored per image reference (e.g., "alpine:latest") and tracks
// all available platforms and their freshness
type ImageCacheMetadata struct {
	ImageRef      string                        `json:"image_ref"`
	CachedAt      time.Time                     `json:"cached_at"`
	ExpiresAt     time.Time                     `json:"expires_at"`
	Platforms     map[string]*PlatformManifest  `json:"platforms"`      // platform string -> manifest info
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

// getCacheDirV2 returns the v2 cache directory for OCI images
func getCacheDirV2() (string, error) {
	userCacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", errors.Errorf("getting user cache dir: %w", err)
	}
	return filepath.Join(userCacheDir, "ec1", "cache", "oci-v2"), nil
}

// getImageCacheDir returns the cache directory for image metadata
func getImageCacheDir(imageRef string) (string, error) {
	cacheDir, err := getCacheDirV2()
	if err != nil {
		return "", err
	}

	// Create a safe directory name from image reference
	imageName := strings.ReplaceAll(imageRef, "/", "_")
	imageName = strings.ReplaceAll(imageName, ":", "_")

	// Hash the image reference for uniqueness
	hasher := sha256.New()
	hasher.Write([]byte(imageRef))
	hash := hex.EncodeToString(hasher.Sum(nil))

	if len(imageName) > 50 {
		imageName = imageName[:50]
	}

	dirname := fmt.Sprintf("%s_%s", imageName, hash[:16])
	return filepath.Join(cacheDir, "images", dirname), nil
}

// getManifestCacheDir returns the cache directory for manifest content
func getManifestCacheDir(manifestDigest string) (string, error) {
	cacheDir, err := getCacheDirV2()
	if err != nil {
		return "", err
	}

	// Use manifest digest as directory name (already a hash)
	return filepath.Join(cacheDir, "manifests", manifestDigest), nil
}

// loadImageCacheMetadata loads image cache metadata
func loadImageCacheMetadata(imageRef string) (*ImageCacheMetadata, error) {
	imageCacheDir, err := getImageCacheDir(imageRef)
	if err != nil {
		return nil, err
	}

	metadataPath := filepath.Join(imageCacheDir, ImageMetadataFile)
	data, err := os.ReadFile(metadataPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No cache entry exists
		}
		return nil, errors.Errorf("reading image cache metadata: %w", err)
	}

	var metadata ImageCacheMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, errors.Errorf("unmarshaling image cache metadata: %w", err)
	}

	return &metadata, nil
}

// saveImageCacheMetadata saves image cache metadata
func saveImageCacheMetadata(metadata *ImageCacheMetadata) error {
	imageCacheDir, err := getImageCacheDir(metadata.ImageRef)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(imageCacheDir, 0755); err != nil {
		return errors.Errorf("creating image cache directory: %w", err)
	}

	metadataPath := filepath.Join(imageCacheDir, ImageMetadataFile)
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return errors.Errorf("marshaling image cache metadata: %w", err)
	}

	if err := os.WriteFile(metadataPath, data, 0644); err != nil {
		return errors.Errorf("writing image cache metadata: %w", err)
	}

	return nil
}

// loadManifestCacheInfo loads manifest cache info
func loadManifestCacheInfo(manifestDigest string) (*ManifestCacheInfo, error) {
	manifestCacheDir, err := getManifestCacheDir(manifestDigest)
	if err != nil {
		return nil, err
	}

	metadataPath := filepath.Join(manifestCacheDir, ManifestMetadataFile)
	data, err := os.ReadFile(metadataPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No cache entry exists
		}
		return nil, errors.Errorf("reading manifest cache metadata: %w", err)
	}

	var info ManifestCacheInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, errors.Errorf("unmarshaling manifest cache metadata: %w", err)
	}

	return &info, nil
}

// saveManifestCacheInfo saves manifest cache info
func saveManifestCacheInfo(info *ManifestCacheInfo) error {
	manifestCacheDir, err := getManifestCacheDir(info.ManifestDigest)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(manifestCacheDir, 0755); err != nil {
		return errors.Errorf("creating manifest cache directory: %w", err)
	}

	metadataPath := filepath.Join(manifestCacheDir, ManifestMetadataFile)
	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return errors.Errorf("marshaling manifest cache metadata: %w", err)
	}

	if err := os.WriteFile(metadataPath, data, 0644); err != nil {
		return errors.Errorf("writing manifest cache metadata: %w", err)
	}

	return nil
}

// isImageCacheValid checks if image metadata cache is still valid
func isImageCacheValid(metadata *ImageCacheMetadata) bool {
	if metadata == nil {
		return false
	}

	// Check if cache has expired
	return time.Now().Before(metadata.ExpiresAt)
}

// isManifestCacheValid checks if manifest content cache is still valid
func isManifestCacheValid(info *ManifestCacheInfo) bool {
	if info == nil {
		return false
	}

	// Check if cached files still exist
	if info.RootfsDiskPath != "" {
		if _, err := os.Stat(info.RootfsDiskPath); err != nil {
			return false
		}
	}

	if _, err := os.Stat(info.RootfsPath); err != nil {
		return false
	}

	if _, err := os.Stat(info.MetadataPath); err != nil {
		return false
	}

	return true
}

// LoadCachedContainerV2 loads a container using the v2 cache system
func LoadCachedContainerV2(ctx context.Context, imageRef string, platform units.Platform) (*CachedContainerV2, error) {
	slog.InfoContext(ctx, "loading container with v2 cache system",
		"image", imageRef,
		"platform", platform,
		"cache_expiration", CacheExpirationV2)

	// Load image metadata
	imageMetadata, err := loadImageCacheMetadata(imageRef)
	if err != nil {
		slog.WarnContext(ctx, "failed to load image cache metadata", "error", err)
	}

	// Check if we need to refresh image metadata
	needsRefresh := !isImageCacheValid(imageMetadata)

	if needsRefresh {
		slog.InfoContext(ctx, "image metadata cache miss or expired, refreshing",
			"image", imageRef,
			"cache_exists", imageMetadata != nil,
			"cache_valid", isImageCacheValid(imageMetadata))

		// Try to refresh image metadata (pull all platforms if possible)
		newMetadata, err := refreshImageMetadata(ctx, imageRef, platform)
		if err != nil {
			if imageMetadata != nil {
				slog.WarnContext(ctx, "failed to refresh metadata but have expired cache, using as fallback", "error", err)
			} else {
				return nil, errors.Errorf("failed to refresh image metadata and no cache available: %w", err)
			}
		} else {
			imageMetadata = newMetadata
		}
	}

	// Find the requested platform
	platformStr := string(platform)
	platformManifest, exists := imageMetadata.Platforms[platformStr]
	if !exists {
		return nil, errors.Errorf("platform %s not available for image %s", platformStr, imageRef)
	}

	// Update last accessed time
	platformManifest.LastAccessed = time.Now()
	imageMetadata.Platforms[platformStr] = platformManifest

	// Save updated metadata
	if err := saveImageCacheMetadata(imageMetadata); err != nil {
		slog.WarnContext(ctx, "failed to save updated image metadata", "error", err)
	}

	// Load manifest cache info
	manifestInfo, err := loadManifestCacheInfo(platformManifest.ManifestDigest)
	if err != nil {
		return nil, errors.Errorf("loading manifest cache info: %w", err)
	}

	// Check if manifest content is cached and valid
	if !isManifestCacheValid(manifestInfo) {
		slog.InfoContext(ctx, "manifest content cache miss or invalid, downloading",
			"manifest_digest", platformManifest.ManifestDigest,
			"platform", platformStr)

		// Download and cache the manifest content
		manifestInfo, err = downloadAndCacheManifest(ctx, imageRef, platform, platformManifest.ManifestDigest)
		if err != nil {
			return nil, errors.Errorf("downloading and caching manifest: %w", err)
		}
	}

	// Load metadata from cache
	metadataData, err := os.ReadFile(manifestInfo.MetadataPath)
	if err != nil {
		return nil, errors.Errorf("reading cached metadata: %w", err)
	}

	var metadata v1.Image
	if err := json.Unmarshal(metadataData, &metadata); err != nil {
		return nil, errors.Errorf("unmarshaling cached metadata: %w", err)
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

// refreshImageMetadata refreshes image metadata by pulling manifest information for all platforms
func refreshImageMetadata(ctx context.Context, imageRef string, requestedPlatform units.Platform) (*ImageCacheMetadata, error) {
	// TODO: Implement multi-platform manifest fetching
	// For now, we'll implement a simplified version that fetches the requested platform
	// In a full implementation, this would:
	// 1. Fetch the manifest list/index
	// 2. Extract all available platforms
	// 3. Store metadata for each platform

	slog.InfoContext(ctx, "refreshing image metadata (simplified implementation)",
		"image", imageRef,
		"requested_platform", string(requestedPlatform))

	// Create new metadata structure
	now := time.Now()
	metadata := &ImageCacheMetadata{
		ImageRef:      imageRef,
		CachedAt:      now,
		ExpiresAt:     now.Add(CacheExpirationV2),
		Platforms:     make(map[string]*PlatformManifest),
		ManifestCache: make(map[string]*ManifestCacheInfo),
	}

	// For now, just add the requested platform
	// TODO: Implement proper multi-platform discovery

	// Generate a mock manifest digest for now
	// TODO: Get real manifest digest from registry
	hasher := sha256.New()
	hasher.Write([]byte(imageRef + string(requestedPlatform) + now.Format(time.RFC3339)))
	manifestDigest := hex.EncodeToString(hasher.Sum(nil))

	metadata.Platforms[string(requestedPlatform)] = &PlatformManifest{
		Platform:       requestedPlatform,
		ManifestDigest: manifestDigest,
		Size:           0, // Will be updated when manifest is cached
		LastAccessed:   now,
	}

	// Save metadata
	if err := saveImageCacheMetadata(metadata); err != nil {
		return nil, errors.Errorf("saving image metadata: %w", err)
	}

	return metadata, nil
}

// downloadAndCacheManifest downloads and caches manifest content for a specific platform
func downloadAndCacheManifest(ctx context.Context, imageRef string, platform units.Platform, manifestDigest string) (*ManifestCacheInfo, error) {
	manifestCacheDir, err := getManifestCacheDir(manifestDigest)
	if err != nil {
		return nil, err
	}

	// Ensure cache directory exists
	if err := os.MkdirAll(manifestCacheDir, 0755); err != nil {
		return nil, errors.Errorf("creating manifest cache directory: %w", err)
	}

	// Create a temporary directory for the download
	tempDir, err := os.MkdirTemp(manifestCacheDir, "download-*")
	if err != nil {
		return nil, errors.Errorf("creating temp directory: %w", err)
	}

	// Clean up temp directory on error
	defer func() {
		if err != nil {
			os.RemoveAll(tempDir)
		}
	}()

	// Convert platform to system context
	sysCtx := platformToSystemContext(platform)

	// Pull and extract the image
	metadata, err := PullAndExtractImageSkopeo(ctx, imageRef, tempDir, sysCtx)
	if err != nil {
		return nil, errors.Errorf("pulling and extracting image: %w", err)
	}

	// Move the extracted content to the cache
	cachedRootfsPath := filepath.Join(manifestCacheDir, "fs")
	cachedOciLayoutPath := filepath.Join(manifestCacheDir, "oci-layout")
	cachedMetadataPath := filepath.Join(manifestCacheDir, "metadata.json")

	// Remove old cache if it exists
	os.RemoveAll(cachedRootfsPath)
	os.Remove(cachedMetadataPath)

	// Move rootfs to cache
	if err := os.Rename(filepath.Join(tempDir, "rootfs"), cachedRootfsPath); err != nil {
		return nil, errors.Errorf("moving rootfs to cache: %w", err)
	}

	if err := os.Rename(filepath.Join(tempDir, "oci-layout"), cachedOciLayoutPath); err != nil {
		return nil, errors.Errorf("moving oci-layout to cache: %w", err)
	}

	// Remove the temp dir
	os.RemoveAll(tempDir)

	// Save metadata to cache
	metadataData, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return nil, errors.Errorf("marshaling metadata: %w", err)
	}

	if err := os.WriteFile(cachedMetadataPath, metadataData, 0644); err != nil {
		return nil, errors.Errorf("writing metadata to cache: %w", err)
	}

	// Calculate cache size
	cacheSize, err := getCachedRootfsSize(cachedRootfsPath)
	if err != nil {
		slog.WarnContext(ctx, "failed to calculate cache size", "error", err)
		cacheSize = 0
	}

	// Bundle to ext4 disk image
	rootfsDiskPath, err := bundleToExt4(ctx, cachedRootfsPath, manifestCacheDir)
	if err != nil {
		return nil, errors.Errorf("bundling rootfs to ext4: %w", err)
	}

	// Create manifest cache info
	info := &ManifestCacheInfo{
		ManifestDigest: manifestDigest,
		RootfsPath:     cachedRootfsPath,
		OciLayoutPath:  cachedOciLayoutPath,
		MetadataPath:   cachedMetadataPath,
		RootfsDiskPath: rootfsDiskPath,
		Size:           cacheSize,
		CachedAt:       time.Now(),
		PlatformString: string(platform),
	}

	// Save manifest cache info
	if err := saveManifestCacheInfo(info); err != nil {
		return nil, errors.Errorf("saving manifest cache info: %w", err)
	}

	slog.InfoContext(ctx, "successfully cached manifest content",
		"manifest_digest", manifestDigest,
		"platform", string(platform),
		"cache_dir", manifestCacheDir,
		"rootfs_disk_path", rootfsDiskPath,
		"size_mb", cacheSize/1024/1024)

	return info, nil
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

// ListCachedImagesV2 returns information about all cached images in v2 cache
func ListCachedImagesV2(ctx context.Context) ([]*ImageCacheMetadata, error) {
	cacheDir, err := getCacheDirV2()
	if err != nil {
		return nil, errors.Errorf("getting cache directory: %w", err)
	}

	imagesDir := filepath.Join(cacheDir, "images")
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

		imageCacheDir := filepath.Join(imagesDir, entry.Name())
		metadataPath := filepath.Join(imageCacheDir, ImageMetadataFile)

		data, err := os.ReadFile(metadataPath)
		if err != nil {
			slog.WarnContext(ctx, "failed to read image metadata", "dir", entry.Name(), "error", err)
			continue
		}

		var metadata ImageCacheMetadata
		if err := json.Unmarshal(data, &metadata); err != nil {
			slog.WarnContext(ctx, "failed to unmarshal image metadata", "dir", entry.Name(), "error", err)
			continue
		}

		cachedImages = append(cachedImages, &metadata)
	}

	return cachedImages, nil
}

// CleanExpiredCacheV2 removes expired cache entries from v2 cache
func CleanExpiredCacheV2(ctx context.Context) error {
	cachedImages, err := ListCachedImagesV2(ctx)
	if err != nil {
		return errors.Errorf("listing cached images: %w", err)
	}

	var removedImageCount int
	var removedManifestCount int
	var reclaimedSize int64

	// Clean expired image metadata
	for _, metadata := range cachedImages {
		if time.Now().After(metadata.ExpiresAt) {
			slog.InfoContext(ctx, "removing expired image metadata",
				"image", metadata.ImageRef,
				"expired_since", time.Since(metadata.ExpiresAt))

			imageCacheDir, err := getImageCacheDir(metadata.ImageRef)
			if err != nil {
				slog.WarnContext(ctx, "failed to get image cache dir", "image", metadata.ImageRef, "error", err)
				continue
			}

			if err := os.RemoveAll(imageCacheDir); err != nil {
				slog.WarnContext(ctx, "failed to remove expired image cache", "dir", imageCacheDir, "error", err)
			} else {
				removedImageCount++
			}
		}
	}

	// Clean orphaned manifest cache entries
	// TODO: Implement orphaned manifest cleanup
	// This would involve checking which manifests are no longer referenced by any image metadata

	slog.InfoContext(ctx, "cleaned expired cache entries",
		"removed_images", removedImageCount,
		"removed_manifests", removedManifestCount,
		"reclaimed_mb", reclaimedSize/1024/1024)

	return nil
}

// ClearCacheV2 removes all cached data from v2 cache
func ClearCacheV2(ctx context.Context) error {
	cacheDir, err := getCacheDirV2()
	if err != nil {
		return errors.Errorf("getting cache directory: %w", err)
	}

	slog.InfoContext(ctx, "clearing v2 OCI image cache", "cache_dir", cacheDir)

	if err := os.RemoveAll(cacheDir); err != nil {
		return errors.Errorf("removing cache directory: %w", err)
	}

	slog.InfoContext(ctx, "successfully cleared v2 OCI image cache")
	return nil
}
