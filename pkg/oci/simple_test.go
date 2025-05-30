package oci_test

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/walteh/ec1/pkg/oci"
	"github.com/walteh/ec1/pkg/testing/toci"
	"github.com/walteh/ec1/pkg/units"

	oci_image_cache "github.com/walteh/ec1/gen/oci-image-cache"
)

func TestSimpleDownloadAndExtract(t *testing.T) {
	ctx := context.Background()
	cache := toci.TestSimpleCache(t)
	defer cache.ClearCache(ctx)

	// Test downloading and extracting an image
	imageRef := oci_image_cache.ALPINE_LATEST.String()
	platform := units.PlatformLinuxAMD64

	// First call should download and cache
	cached1, err := cache.LoadImage(ctx, imageRef, platform)
	require.NoError(t, err)
	assert.Equal(t, imageRef, cached1.ImageRef)
	assert.Equal(t, platform, cached1.Platform)
	assert.DirExists(t, cached1.RootfsPath)
	assert.FileExists(t, cached1.Ext4Path)
	assert.NotNil(t, cached1.Metadata)

	// Second call should use cache
	cached2, err := cache.LoadImage(ctx, imageRef, platform)
	require.NoError(t, err)
	assert.Equal(t, cached1.RootfsPath, cached2.RootfsPath)
	assert.Equal(t, cached1.Ext4Path, cached2.Ext4Path)
}

func TestContainerdShimInterface(t *testing.T) {
	ctx := context.Background()
	cache := toci.TestSimpleCache(t)
	defer cache.ClearCache(ctx)

	// Simulate containerd providing an OCI layout path
	downloader := toci.TestImageDownloaderInitialized(t)
	imageRef := oci_image_cache.ALPINE_LATEST.String()

	// Download to get OCI layout path (simulating containerd's work)
	ociLayoutPath, err := downloader.DownloadImage(ctx, imageRef)
	require.NoError(t, err)
	defer os.RemoveAll(ociLayoutPath)

	platform := units.PlatformLinuxAMD64

	// Use the containerd integration method
	cached, err := cache.LoadImageFromOCILayout(ctx, ociLayoutPath, imageRef, platform)
	require.NoError(t, err)
	assert.Equal(t, imageRef, cached.ImageRef)
	assert.Equal(t, platform, cached.Platform)
	assert.DirExists(t, cached.RootfsPath)
	assert.FileExists(t, cached.Ext4Path)
	assert.NotNil(t, cached.Metadata)

	// Verify cache hit on second call
	cached2, err := cache.LoadImageFromOCILayout(ctx, ociLayoutPath, imageRef, platform)
	require.NoError(t, err)
	assert.Equal(t, cached.RootfsPath, cached2.RootfsPath)
}

func TestSimpleCacheFlow(t *testing.T) {
	ctx := context.Background()
	cache := toci.TestSimpleCache(t)
	defer cache.ClearCache(ctx)

	imageRef := oci_image_cache.ALPINE_LATEST.String()
	platform := units.PlatformLinuxAMD64

	// Load image (should download and cache)
	cached, err := cache.LoadImage(ctx, imageRef, platform)
	require.NoError(t, err)
	assert.Equal(t, imageRef, cached.ImageRef)
	assert.Equal(t, platform, cached.Platform)

	// Verify cache entry exists
	images, err := cache.ListCachedImages(ctx)
	require.NoError(t, err)
	assert.Len(t, images, 1)
	assert.Equal(t, imageRef, images[0].ImageRef)

	// Load same image again (should use cache)
	cached2, err := cache.LoadImage(ctx, imageRef, platform)
	require.NoError(t, err)
	assert.Equal(t, cached.RootfsPath, cached2.RootfsPath)
	assert.Equal(t, cached.Ext4Path, cached2.Ext4Path)
}

func TestCacheManagement(t *testing.T) {
	ctx := context.Background()
	cache := toci.TestSimpleCache(t)
	defer cache.ClearCache(ctx)

	imageRef := oci_image_cache.ALPINE_LATEST.String()
	platform := units.PlatformLinuxAMD64

	// Load an image
	_, err := cache.LoadImage(ctx, imageRef, platform)
	require.NoError(t, err)

	// Check cache size
	size, err := cache.GetCacheSize(ctx)
	require.NoError(t, err)
	assert.Greater(t, size, int64(0))

	// List cached images
	images, err := cache.ListCachedImages(ctx)
	require.NoError(t, err)
	assert.Len(t, images, 1)

	// Clear cache
	err = cache.ClearCache(ctx)
	require.NoError(t, err)

	// Verify cache is empty
	images, err = cache.ListCachedImages(ctx)
	require.NoError(t, err)
	assert.Len(t, images, 0)

	size, err = cache.GetCacheSize(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(0), size)
}

func TestCacheExpiration(t *testing.T) {
	ctx := context.Background()
	cache := toci.TestSimpleCache(t)
	defer cache.ClearCache(ctx)

	imageRef := oci_image_cache.ALPINE_LATEST.String()
	platform := units.PlatformLinuxAMD64

	// Load an image
	_, err := cache.LoadImage(ctx, imageRef, platform)
	require.NoError(t, err)

	// Verify image is cached
	images, err := cache.ListCachedImages(ctx)
	require.NoError(t, err)
	assert.Len(t, images, 1)

	// Clean with very short expiration (should remove the image)
	err = cache.CleanExpiredCache(ctx, 1*time.Nanosecond)
	require.NoError(t, err)

	// Verify image was removed
	images, err = cache.ListCachedImages(ctx)
	require.NoError(t, err)
	assert.Len(t, images, 0)

	// Load image again
	_, err = cache.LoadImage(ctx, imageRef, platform)
	require.NoError(t, err)

	// Clean with long expiration (should keep the image)
	err = cache.CleanExpiredCache(ctx, 24*time.Hour)
	require.NoError(t, err)

	// Verify image is still there
	images, err = cache.ListCachedImages(ctx)
	require.NoError(t, err)
	assert.Len(t, images, 1)
}

func TestContainerdIntegrationScenarios(t *testing.T) {
	ctx := context.Background()
	cache := toci.TestSimpleCache(t)
	defer cache.ClearCache(ctx)

	downloader := toci.TestImageDownloaderInitialized(t)
	imageRef := oci_image_cache.ALPINE_LATEST.String()

	// Scenario 1: Containerd provides OCI layout, we extract and cache
	ociLayoutPath, err := downloader.DownloadImage(ctx, imageRef)
	require.NoError(t, err)
	defer os.RemoveAll(ociLayoutPath)

	platform := units.PlatformLinuxAMD64
	cached, err := cache.LoadImageFromOCILayout(ctx, ociLayoutPath, imageRef, platform)
	require.NoError(t, err)
	assert.DirExists(t, cached.RootfsPath)
	assert.FileExists(t, cached.Ext4Path)

	// Scenario 2: Same image, different platform
	platform2 := units.PlatformLinuxARM64
	cached2, err := cache.LoadImageFromOCILayout(ctx, ociLayoutPath, imageRef, platform2)
	require.NoError(t, err)
	assert.NotEqual(t, cached.RootfsPath, cached2.RootfsPath) // Different platforms = different cache entries

	// Scenario 3: Cache hit for existing platform
	cached3, err := cache.LoadImageFromOCILayout(ctx, ociLayoutPath, imageRef, platform)
	require.NoError(t, err)
	assert.Equal(t, cached.RootfsPath, cached3.RootfsPath) // Same platform = cache hit

	// Verify we have 2 cache entries (2 platforms)
	images, err := cache.ListCachedImages(ctx)
	require.NoError(t, err)
	assert.Len(t, images, 2)

	// Scenario 4: Standalone mode without downloader should fail
	cacheNoDownloader := oci.NewSimpleCache("/tmp/test-no-downloader", nil, &oci.OCIImageExtractor{})
	defer cacheNoDownloader.ClearCache(ctx)

	_, err = cacheNoDownloader.LoadImage(ctx, imageRef, platform)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no downloader configured")
}

func TestErrorHandling(t *testing.T) {
	ctx := context.Background()
	cache := toci.TestSimpleCache(t)
	defer cache.ClearCache(ctx)

	// Test with invalid image reference
	_, err := cache.LoadImage(ctx, "invalid-image-ref", units.PlatformLinuxAMD64)
	assert.Error(t, err)

	// Test with non-existent OCI layout path
	_, err = cache.LoadImageFromOCILayout(ctx, "/non/existent/path", "alpine:3.21", units.PlatformLinuxAMD64)
	assert.Error(t, err)

	// Test cache operations on non-existent cache
	tempDownloader := toci.TestImageDownloaderInitialized(t)
	tempCache := oci.NewSimpleCache("/tmp/non-existent-cache-dir", tempDownloader, &oci.OCIImageExtractor{})

	images, err := tempCache.ListCachedImages(ctx)
	require.NoError(t, err)
	assert.Len(t, images, 0) // Should return empty list, not error

	_, err = tempCache.GetCacheSize(ctx)
	assert.Error(t, err) // Should error on non-existent directory

	err = tempCache.CleanExpiredCache(ctx, time.Hour)
	require.NoError(t, err) // Should not error on non-existent directory
}

func TestConcurrentAccess(t *testing.T) {
	ctx := context.Background()

	// Create separate cache instances to avoid shared state
	downloader1 := toci.TestImageDownloaderInitialized(t)
	downloader2 := toci.TestImageDownloaderInitialized(t)
	cache1 := oci.NewSimpleCache("/tmp/ec1-concurrent-test-1", downloader1, &oci.OCIImageExtractor{})
	cache2 := oci.NewSimpleCache("/tmp/ec1-concurrent-test-2", downloader2, &oci.OCIImageExtractor{})
	defer cache1.ClearCache(ctx)
	defer cache2.ClearCache(ctx)

	imageRef := oci_image_cache.ALPINE_LATEST.String()
	platform := units.PlatformLinuxAMD64

	var wg sync.WaitGroup
	var errors []error
	var mu sync.Mutex

	// Launch multiple goroutines to access cache concurrently
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(cacheInstance *oci.SimpleCache) {
			defer wg.Done()
			_, err := cacheInstance.LoadImage(ctx, imageRef, platform)
			if err != nil {
				mu.Lock()
				errors = append(errors, err)
				mu.Unlock()
			}
		}(cache1)
	}

	// Also test concurrent access to different cache instances
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(cacheInstance *oci.SimpleCache) {
			defer wg.Done()
			_, err := cacheInstance.LoadImage(ctx, imageRef, platform)
			if err != nil {
				mu.Lock()
				errors = append(errors, err)
				mu.Unlock()
			}
		}(cache2)
	}

	wg.Wait()

	// Check that no errors occurred
	assert.Empty(t, errors, "Concurrent access should not cause errors")

	// Verify both caches have the image
	images1, err := cache1.ListCachedImages(ctx)
	require.NoError(t, err)
	assert.Len(t, images1, 1)

	images2, err := cache2.ListCachedImages(ctx)
	require.NoError(t, err)
	assert.Len(t, images2, 1)
}

func TestGlobalCacheFunctions(t *testing.T) {
	ctx := context.Background()

	// Clear all caches
	err := oci.ClearAllCaches(ctx)
	require.NoError(t, err)

	// List all cached images (should be empty)
	images, err := oci.ListAllCachedImages(ctx)
	require.NoError(t, err)
	assert.Len(t, images, 0)
}
