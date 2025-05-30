package oci

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/walteh/ec1/pkg/testing/tlog"
	"github.com/walteh/ec1/pkg/units"
)

func TestCacheV2BasicFlow(t *testing.T) {
	ctx := tlog.SetupSlogForTest(t)

	// Clear cache before test
	err := ClearCacheV2(ctx)
	require.NoError(t, err)

	imageRef := "docker.io/library/alpine:latest"
	platform := units.PlatformLinuxARM64

	// First call should download and cache
	t.Logf("First call - should download and cache")
	container1, err := LoadCachedContainerV2(ctx, imageRef, platform)
	require.NoError(t, err, "Failed to load container with v2 cache")
	require.NotNil(t, container1, "Expected non-nil container")
	require.Equal(t, imageRef, container1.ImageRef)
	require.Equal(t, platform, container1.Platform)
	require.NotEmpty(t, container1.ManifestDigest)
	require.NotEmpty(t, container1.ReadonlyFSPath)
	require.NotEmpty(t, container1.ReadonlyExt4Path)
	require.NotNil(t, container1.Metadata)

	// Verify cache files exist
	assert.FileExists(t, container1.ReadonlyExt4Path, "Expected ext4 disk image to exist")
	assert.DirExists(t, container1.ReadonlyFSPath, "Expected rootfs directory to exist")

	// Second call should use cache
	t.Logf("Second call - should use cache")
	container2, err := LoadCachedContainerV2(ctx, imageRef, platform)
	require.NoError(t, err, "Failed to load container from cache")
	require.NotNil(t, container2, "Expected non-nil container from cache")

	// Verify same manifest digest (should be cached)
	assert.Equal(t, container1.ManifestDigest, container2.ManifestDigest, "Expected same manifest digest from cache")
	assert.Equal(t, container1.ReadonlyExt4Path, container2.ReadonlyExt4Path, "Expected same ext4 path from cache")
	assert.Equal(t, container1.ReadonlyFSPath, container2.ReadonlyFSPath, "Expected same rootfs path from cache")
}

func TestCacheV2MetadataStructure(t *testing.T) {
	ctx := tlog.SetupSlogForTest(t)

	// Clear cache before test
	err := ClearCacheV2(ctx)
	require.NoError(t, err)

	imageRef := "docker.io/library/alpine:latest"
	platform := units.PlatformLinuxARM64

	// Load container to create cache
	_, err = LoadCachedContainerV2(ctx, imageRef, platform)
	require.NoError(t, err)

	// Verify image metadata structure
	imageMetadata, err := loadImageCacheMetadata(imageRef)
	require.NoError(t, err, "Failed to load image metadata")
	require.NotNil(t, imageMetadata, "Expected image metadata to exist")

	assert.Equal(t, imageRef, imageMetadata.ImageRef)
	assert.True(t, time.Now().Before(imageMetadata.ExpiresAt), "Expected cache to not be expired")
	assert.Contains(t, imageMetadata.Platforms, string(platform), "Expected platform to be cached")

	platformManifest := imageMetadata.Platforms[string(platform)]
	assert.Equal(t, platform, platformManifest.Platform)
	assert.NotEmpty(t, platformManifest.ManifestDigest)
	assert.True(t, time.Now().After(platformManifest.LastAccessed.Add(-time.Minute)), "Expected recent last accessed time")

	// Verify manifest cache structure
	manifestInfo, err := loadManifestCacheInfo(platformManifest.ManifestDigest)
	require.NoError(t, err, "Failed to load manifest cache info")
	require.NotNil(t, manifestInfo, "Expected manifest cache info to exist")

	assert.Equal(t, platformManifest.ManifestDigest, manifestInfo.ManifestDigest)
	assert.Equal(t, string(platform), manifestInfo.PlatformString)
	assert.FileExists(t, manifestInfo.RootfsDiskPath)
	assert.DirExists(t, manifestInfo.RootfsPath)
	assert.FileExists(t, manifestInfo.MetadataPath)
	assert.Greater(t, manifestInfo.Size, int64(0), "Expected non-zero cache size")
}

func TestCacheV2Expiration(t *testing.T) {
	// Create a mock expired cache entry
	imageRef := "docker.io/library/alpine:test-expired"

	// Create image metadata that's already expired
	expiredMetadata := &ImageCacheMetadata{
		ImageRef:      imageRef,
		CachedAt:      time.Now().Add(-25 * time.Hour), // 25 hours ago
		ExpiresAt:     time.Now().Add(-1 * time.Hour),  // 1 hour ago (expired)
		Platforms:     make(map[string]*PlatformManifest),
		ManifestCache: make(map[string]*ManifestCacheInfo),
	}

	// Should be invalid due to expiration
	assert.False(t, isImageCacheValid(expiredMetadata), "Expired cache should be invalid")

	// Create a valid cache entry
	validMetadata := &ImageCacheMetadata{
		ImageRef:      imageRef,
		CachedAt:      time.Now().Add(-1 * time.Hour), // 1 hour ago
		ExpiresAt:     time.Now().Add(23 * time.Hour), // 23 hours from now
		Platforms:     make(map[string]*PlatformManifest),
		ManifestCache: make(map[string]*ManifestCacheInfo),
	}

	// Should be valid
	assert.True(t, isImageCacheValid(validMetadata), "Valid cache should be valid")
}

func TestCacheV2ManifestValidation(t *testing.T) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "oci-cache-v2-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create mock manifest cache info with valid files
	rootfsPath := filepath.Join(tempDir, "rootfs")
	metadataPath := filepath.Join(tempDir, "metadata.json")
	diskPath := filepath.Join(tempDir, "fs.ext4.img")

	// Create the files
	err = os.MkdirAll(rootfsPath, 0755)
	require.NoError(t, err)
	err = os.WriteFile(metadataPath, []byte(`{"test": "data"}`), 0644)
	require.NoError(t, err)
	err = os.WriteFile(diskPath, []byte("fake disk data"), 0644)
	require.NoError(t, err)

	validManifestInfo := &ManifestCacheInfo{
		ManifestDigest: "test-digest",
		RootfsPath:     rootfsPath,
		MetadataPath:   metadataPath,
		RootfsDiskPath: diskPath,
		Size:           1024,
		CachedAt:       time.Now(),
	}

	// Should be valid when all files exist
	assert.True(t, isManifestCacheValid(validManifestInfo), "Manifest cache with existing files should be valid")

	// Should be invalid when files are missing
	invalidManifestInfo := &ManifestCacheInfo{
		ManifestDigest: "test-digest",
		RootfsPath:     "/nonexistent/path",
		MetadataPath:   "/nonexistent/metadata.json",
		RootfsDiskPath: "/nonexistent/disk.img",
		Size:           1024,
		CachedAt:       time.Now(),
	}

	assert.False(t, isManifestCacheValid(invalidManifestInfo), "Manifest cache with missing files should be invalid")
}

func TestCacheV2DirectoryStructure(t *testing.T) {
	imageRef := "docker.io/library/test:v1.0"
	manifestDigest := "sha256:abcdef1234567890"

	// Test image cache directory
	imageCacheDir, err := getImageCacheDir(imageRef)
	require.NoError(t, err)
	assert.Contains(t, imageCacheDir, "oci-v2/images/")
	assert.Contains(t, imageCacheDir, "docker.io_library_test_v1.0")

	// Test manifest cache directory
	manifestCacheDir, err := getManifestCacheDir(manifestDigest)
	require.NoError(t, err)
	assert.Contains(t, manifestCacheDir, "oci-v2/manifests/")
	assert.Contains(t, manifestCacheDir, manifestDigest)

	// Verify different images get different directories
	imageRef2 := "docker.io/library/test:v2.0"
	imageCacheDir2, err := getImageCacheDir(imageRef2)
	require.NoError(t, err)
	assert.NotEqual(t, imageCacheDir, imageCacheDir2, "Different images should have different cache directories")
}

func TestCacheV2PlatformToSystemContext(t *testing.T) {
	testCases := []struct {
		platform     units.Platform
		expectedOS   string
		expectedArch string
	}{
		{units.PlatformLinuxAMD64, "linux", "amd64"},
		{units.PlatformLinuxARM64, "linux", "arm64"},
		{units.PlatformDarwinAMD64, "darwin", "amd64"},
		{units.PlatformDarwinARM64, "darwin", "arm64"},
	}

	for _, tc := range testCases {
		t.Run(string(tc.platform), func(t *testing.T) {
			sysCtx := platformToSystemContext(tc.platform)
			assert.Equal(t, tc.expectedOS, sysCtx.OSChoice)
			assert.Equal(t, tc.expectedArch, sysCtx.ArchitectureChoice)
		})
	}
}

func TestCacheV2ListCachedImages(t *testing.T) {
	ctx := tlog.SetupSlogForTest(t)

	// Clear cache before test
	err := ClearCacheV2(ctx)
	require.NoError(t, err)

	// Initially should be empty
	cachedImages, err := ListCachedImagesV2(ctx)
	require.NoError(t, err)
	assert.Empty(t, cachedImages, "Expected empty cache initially")

	// Create some cache entries by loading containers
	imageRef1 := "docker.io/library/alpine:latest"
	imageRef2 := "docker.io/library/alpine:3.19" // Use a valid tag instead of ubuntu
	platform := units.PlatformLinuxARM64

	// Load first image
	_, err = LoadCachedContainerV2(ctx, imageRef1, platform)
	require.NoError(t, err)

	// Load second image
	_, err = LoadCachedContainerV2(ctx, imageRef2, platform)
	require.NoError(t, err)

	// List cached images
	cachedImages, err = ListCachedImagesV2(ctx)
	require.NoError(t, err)
	assert.Len(t, cachedImages, 2, "Expected 2 cached images")

	// Verify image references are present
	imageRefs := make([]string, len(cachedImages))
	for i, img := range cachedImages {
		imageRefs[i] = img.ImageRef
	}
	assert.Contains(t, imageRefs, imageRef1)
	assert.Contains(t, imageRefs, imageRef2)
}

func TestCacheV2CleanExpiredCache(t *testing.T) {
	ctx := tlog.SetupSlogForTest(t)

	// Clear cache before test
	err := ClearCacheV2(ctx)
	require.NoError(t, err)

	// Create a cache entry that will be expired
	imageRef := "docker.io/library/alpine:3.17" // Use a valid tag
	expiredMetadata := &ImageCacheMetadata{
		ImageRef:      imageRef,
		CachedAt:      time.Now().Add(-25 * time.Hour), // 25 hours ago
		ExpiresAt:     time.Now().Add(-1 * time.Hour),  // 1 hour ago (expired)
		Platforms:     make(map[string]*PlatformManifest),
		ManifestCache: make(map[string]*ManifestCacheInfo),
	}

	// Save the expired metadata
	err = saveImageCacheMetadata(expiredMetadata)
	require.NoError(t, err)

	// Verify it exists
	cachedImages, err := ListCachedImagesV2(ctx)
	require.NoError(t, err)
	assert.Len(t, cachedImages, 1, "Expected 1 cached image before cleanup")

	// Clean expired cache
	err = CleanExpiredCacheV2(ctx)
	require.NoError(t, err)

	// Verify it's been removed
	cachedImages, err = ListCachedImagesV2(ctx)
	require.NoError(t, err)
	assert.Empty(t, cachedImages, "Expected empty cache after cleanup")
}

func TestCacheV2ClearCache(t *testing.T) {
	ctx := tlog.SetupSlogForTest(t)

	// Create some cache entries
	imageRef := "docker.io/library/alpine:3.18" // Use a valid tag
	platform := units.PlatformLinuxARM64

	_, err := LoadCachedContainerV2(ctx, imageRef, platform)
	require.NoError(t, err)

	// Verify cache exists
	cachedImages, err := ListCachedImagesV2(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, cachedImages, "Expected cache to exist before clear")

	// Clear cache
	err = ClearCacheV2(ctx)
	require.NoError(t, err)

	// Verify cache is empty
	cachedImages, err = ListCachedImagesV2(ctx)
	require.NoError(t, err)
	assert.Empty(t, cachedImages, "Expected empty cache after clear")
}

func TestCacheV2LastAccessedTracking(t *testing.T) {
	ctx := tlog.SetupSlogForTest(t)

	// Clear cache before test
	err := ClearCacheV2(ctx)
	require.NoError(t, err)

	imageRef := "docker.io/library/alpine:latest"
	platform := units.PlatformLinuxARM64

	// Load container first time
	_, err = LoadCachedContainerV2(ctx, imageRef, platform)
	require.NoError(t, err)

	// Get initial last accessed time
	imageMetadata1, err := loadImageCacheMetadata(imageRef)
	require.NoError(t, err)
	platformManifest1 := imageMetadata1.Platforms[string(platform)]
	initialLastAccessed := platformManifest1.LastAccessed

	// Wait a bit and load again
	time.Sleep(100 * time.Millisecond)
	_, err = LoadCachedContainerV2(ctx, imageRef, platform)
	require.NoError(t, err)

	// Get updated last accessed time
	imageMetadata2, err := loadImageCacheMetadata(imageRef)
	require.NoError(t, err)
	platformManifest2 := imageMetadata2.Platforms[string(platform)]
	updatedLastAccessed := platformManifest2.LastAccessed

	// Verify last accessed time was updated
	assert.True(t, updatedLastAccessed.After(initialLastAccessed),
		"Expected last accessed time to be updated on second access")
}

// Benchmark tests for cache performance
func BenchmarkCacheV2LoadCachedContainer(b *testing.B) {
	ctx := context.Background()

	// Setup cache with a container
	imageRef := "docker.io/library/alpine:latest"
	platform := units.PlatformLinuxARM64

	// Load once to populate cache
	_, err := LoadCachedContainerV2(ctx, imageRef, platform)
	if err != nil {
		b.Fatalf("Failed to setup cache: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := LoadCachedContainerV2(ctx, imageRef, platform)
		if err != nil {
			b.Fatalf("Failed to load cached container: %v", err)
		}
	}
}

func BenchmarkCacheV2MetadataOperations(b *testing.B) {
	imageRef := "docker.io/library/alpine:benchmark"

	// Create test metadata
	metadata := &ImageCacheMetadata{
		ImageRef:      imageRef,
		CachedAt:      time.Now(),
		ExpiresAt:     time.Now().Add(CacheExpirationV2),
		Platforms:     make(map[string]*PlatformManifest),
		ManifestCache: make(map[string]*ManifestCacheInfo),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Benchmark save and load operations
		err := saveImageCacheMetadata(metadata)
		if err != nil {
			b.Fatalf("Failed to save metadata: %v", err)
		}

		_, err = loadImageCacheMetadata(imageRef)
		if err != nil {
			b.Fatalf("Failed to load metadata: %v", err)
		}
	}
}
