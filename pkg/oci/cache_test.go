package oci_test

// func setupInterfaces(t *testing.T) (oci.ImageFetcher, oci.FilesystemConverter, oci.ImageCache) {
// 	converter := oci.NewOCIFilesystemConverter()
// 	fetcher, err := oci.NewMemoryMapFetcher(t.Context(), oci_image_cache.Registry)
// 	require.NoError(t, err)
// 	cache := oci.NewImageCache(t.TempDir(), fetcher, converter)
// 	t.Cleanup(func() {
// 		cache.ClearCache(t.Context())
// 	})
// 	return fetcher, converter, cache
// }

// func TestImageCacheFetchAndConvert(t *testing.T) {
// 	ctx := context.Background()
// 	_, _, cache := setupInterfaces(t)

// 	// Test fetching and converting an image
// 	imageRef := oci_image_cache.ALPINE_LATEST.String()
// 	platform := units.PlatformLinuxAMD64

// 	// First call should fetch and cache
// 	cached1, err := cache.LoadImage(ctx, imageRef, platform)
// 	require.NoError(t, err)
// 	assert.Equal(t, imageRef, cached1.ImageRef)
// 	assert.Equal(t, platform, cached1.Platform)
// 	assert.DirExists(t, cached1.RootfsPath)
// 	assert.FileExists(t, cached1.Ext4Path)
// 	assert.NotNil(t, cached1.Metadata)

// 	// Second call should use cache
// 	cached2, err := cache.LoadImage(ctx, imageRef, platform)
// 	require.NoError(t, err)
// 	assert.Equal(t, cached1.RootfsPath, cached2.RootfsPath)
// 	assert.Equal(t, cached1.Ext4Path, cached2.Ext4Path)
// }

// func TestContainerdShimInterface(t *testing.T) {
// 	ctx := context.Background()
// 	fetcher, _, cache := setupInterfaces(t)

// 	// Simulate containerd providing an OCI layout path
// 	imageRef := oci_image_cache.ALPINE_LATEST.String()

// 	// Fetch to get OCI layout path (simulating containerd's work)
// 	ociLayoutPath, err := fetcher.FetchImage(ctx, imageRef)
// 	require.NoError(t, err)
// 	defer os.RemoveAll(ociLayoutPath)

// 	platform := units.PlatformLinuxAMD64

// 	// Use the containerd integration method
// 	cached, err := cache.LoadImageFromOCILayout(ctx, ociLayoutPath, imageRef, platform)
// 	require.NoError(t, err)
// 	assert.Equal(t, imageRef, cached.ImageRef)
// 	assert.Equal(t, platform, cached.Platform)
// 	assert.DirExists(t, cached.RootfsPath)
// 	assert.FileExists(t, cached.Ext4Path)
// 	assert.NotNil(t, cached.Metadata)

// 	// Verify cache hit on second call
// 	cached2, err := cache.LoadImageFromOCILayout(ctx, ociLayoutPath, imageRef, platform)
// 	require.NoError(t, err)
// 	assert.Equal(t, cached.RootfsPath, cached2.RootfsPath)
// }

// func TestImageCacheFlow(t *testing.T) {
// 	ctx := context.Background()
// 	_, _, cache := setupInterfaces(t)

// 	imageRef := oci_image_cache.ALPINE_LATEST.String()
// 	platform := units.PlatformLinuxAMD64

// 	// Load image (should fetch and cache)
// 	cached, err := cache.LoadImage(ctx, imageRef, platform)
// 	require.NoError(t, err)
// 	assert.Equal(t, imageRef, cached.ImageRef)
// 	assert.Equal(t, platform, cached.Platform)

// 	// Verify cache entry exists
// 	images, err := cache.ListCachedImages(ctx)
// 	require.NoError(t, err)
// 	assert.Len(t, images, 1)
// 	assert.Equal(t, imageRef, images[0].ImageRef)

// 	// Load same image again (should use cache)
// 	cached2, err := cache.LoadImage(ctx, imageRef, platform)
// 	require.NoError(t, err)
// 	assert.Equal(t, cached.RootfsPath, cached2.RootfsPath)
// 	assert.Equal(t, cached.Ext4Path, cached2.Ext4Path)
// }

// func TestCacheManagement(t *testing.T) {
// 	ctx := context.Background()
// 	_, _, cache := setupInterfaces(t)

// 	imageRef := oci_image_cache.ALPINE_LATEST.String()
// 	platform := units.PlatformLinuxAMD64

// 	// Load an image
// 	_, err := cache.LoadImage(ctx, imageRef, platform)
// 	require.NoError(t, err)

// 	// Check cache size
// 	size, err := cache.GetCacheSize(ctx)
// 	require.NoError(t, err)
// 	assert.Greater(t, size, int64(0))

// 	// List cached images
// 	images, err := cache.ListCachedImages(ctx)
// 	require.NoError(t, err)
// 	assert.Len(t, images, 1)

// 	// Clear cache
// 	err = cache.ClearCache(ctx)
// 	require.NoError(t, err)

// 	// Verify cache is empty
// 	images, err = cache.ListCachedImages(ctx)
// 	require.NoError(t, err)
// 	assert.Len(t, images, 0)

// 	size, err = cache.GetCacheSize(ctx)
// 	require.NoError(t, err)
// 	assert.Equal(t, int64(0), size)
// }

// func TestCacheExpiration(t *testing.T) {
// 	ctx := context.Background()
// 	_, _, cache := setupInterfaces(t)

// 	imageRef := oci_image_cache.ALPINE_LATEST.String()
// 	platform := units.PlatformLinuxAMD64

// 	// Load an image
// 	_, err := cache.LoadImage(ctx, imageRef, platform)
// 	require.NoError(t, err)

// 	// Verify image is cached
// 	images, err := cache.ListCachedImages(ctx)
// 	require.NoError(t, err)
// 	assert.Len(t, images, 1)

// 	// Clean with very short expiration (should remove the image)
// 	err = cache.CleanExpiredCache(ctx, 1*time.Nanosecond)
// 	require.NoError(t, err)

// 	// Verify image was removed
// 	images, err = cache.ListCachedImages(ctx)
// 	require.NoError(t, err)
// 	assert.Len(t, images, 0)

// 	// Load image again
// 	_, err = cache.LoadImage(ctx, imageRef, platform)
// 	require.NoError(t, err)

// 	// Clean with long expiration (should keep the image)
// 	err = cache.CleanExpiredCache(ctx, 24*time.Hour)
// 	require.NoError(t, err)

// 	// Verify image is still there
// 	images, err = cache.ListCachedImages(ctx)
// 	require.NoError(t, err)
// 	assert.Len(t, images, 1)
// }

// func TestContainerdIntegrationScenarios(t *testing.T) {
// 	ctx := context.Background()
// 	fetcher, converter, cache := setupInterfaces(t)

// 	imageRef := oci_image_cache.ALPINE_LATEST.String()

// 	// Scenario 1: Containerd provides OCI layout, we convert and cache
// 	ociLayoutPath, err := fetcher.FetchImage(ctx, imageRef)
// 	require.NoError(t, err)
// 	defer os.RemoveAll(ociLayoutPath)

// 	platform := units.PlatformLinuxAMD64
// 	cached, err := cache.LoadImageFromOCILayout(ctx, ociLayoutPath, imageRef, platform)
// 	require.NoError(t, err)
// 	assert.DirExists(t, cached.RootfsPath)
// 	assert.FileExists(t, cached.Ext4Path)

// 	// Scenario 2: Same image, different platform
// 	platform2 := units.PlatformLinuxARM64
// 	cached2, err := cache.LoadImageFromOCILayout(ctx, ociLayoutPath, imageRef, platform2)
// 	require.NoError(t, err)
// 	assert.NotEqual(t, cached.RootfsPath, cached2.RootfsPath) // Different platforms = different cache entries

// 	// Scenario 3: Cache hit for existing platform
// 	cached3, err := cache.LoadImageFromOCILayout(ctx, ociLayoutPath, imageRef, platform)
// 	require.NoError(t, err)
// 	assert.Equal(t, cached.RootfsPath, cached3.RootfsPath) // Same platform = cache hit

// 	// Verify we have 2 cache entries (2 platforms)
// 	images, err := cache.ListCachedImages(ctx)
// 	require.NoError(t, err)
// 	assert.Len(t, images, 2)

// 	// Scenario 4: Standalone mode without fetcher should fail
// 	cacheNoFetcher := oci.NewImageCache("/tmp/test-no-fetcher", nil, converter)
// 	defer cacheNoFetcher.ClearCache(ctx)

// 	_, err = cacheNoFetcher.LoadImage(ctx, imageRef, platform)
// 	assert.Error(t, err)
// 	assert.Contains(t, err.Error(), "no fetcher configured")
// }

// func TestErrorHandling(t *testing.T) {
// 	ctx := context.Background()
// 	fetcher, converter, cache := setupInterfaces(t)

// 	// Test with invalid image reference
// 	_, err := cache.LoadImage(ctx, "invalid-image-ref", units.PlatformLinuxAMD64)
// 	assert.Error(t, err)

// 	// Test with non-existent OCI layout path
// 	_, err = cache.LoadImageFromOCILayout(ctx, "/non/existent/path", "alpine:3.21", units.PlatformLinuxAMD64)
// 	assert.Error(t, err)

// 	// Test cache operations on non-existent cache
// 	tempCache := oci.NewImageCache("/tmp/non-existent-cache-dir", fetcher, converter)

// 	images, err := tempCache.ListCachedImages(ctx)
// 	require.NoError(t, err)
// 	assert.Len(t, images, 0) // Should return empty list, not error

// 	_, err = tempCache.GetCacheSize(ctx)
// 	assert.Error(t, err) // Should error on non-existent directory

// 	err = tempCache.CleanExpiredCache(ctx, time.Hour)
// 	require.NoError(t, err) // Should not error on non-existent directory
// }

// func TestConcurrentAccess(t *testing.T) {
// 	ctx := context.Background()

// 	fetcher, converter, _ := setupInterfaces(t)
// 	cache1 := oci.NewImageCache("/tmp/ec1-concurrent-test-1", fetcher, converter)
// 	cache2 := oci.NewImageCache("/tmp/ec1-concurrent-test-2", fetcher, converter)
// 	t.Cleanup(func() {
// 		cache1.ClearCache(ctx)
// 		cache2.ClearCache(ctx)
// 	})

// 	imageRef := oci_image_cache.ALPINE_LATEST.String()
// 	platform := units.PlatformLinuxAMD64

// 	var wg sync.WaitGroup
// 	var errors []error
// 	var mu sync.Mutex

// 	// Launch multiple goroutines to access cache concurrently
// 	for i := 0; i < 5; i++ {
// 		wg.Add(1)
// 		go func(cacheInstance oci.ImageCache) {
// 			defer wg.Done()
// 			_, err := cacheInstance.LoadImage(ctx, imageRef, platform)
// 			if err != nil {
// 				mu.Lock()
// 				errors = append(errors, err)
// 				mu.Unlock()
// 			}
// 		}(cache1)
// 	}

// 	// Also test concurrent access to different cache instances
// 	for i := 0; i < 5; i++ {
// 		wg.Add(1)
// 		go func(cacheInstance oci.ImageCache) {
// 			defer wg.Done()
// 			_, err := cacheInstance.LoadImage(ctx, imageRef, platform)
// 			if err != nil {
// 				mu.Lock()
// 				errors = append(errors, err)
// 				mu.Unlock()
// 			}
// 		}(cache2)
// 	}

// 	wg.Wait()

// 	// Check that no errors occurred
// 	assert.Empty(t, errors, "Concurrent access should not cause errors")

// 	// Verify both caches have the image
// 	images1, err := cache1.ListCachedImages(ctx)
// 	require.NoError(t, err)
// 	assert.Len(t, images1, 1)

// 	images2, err := cache2.ListCachedImages(ctx)
// 	require.NoError(t, err)
// 	assert.Len(t, images2, 1)
// }

// func TestContainerShim(t *testing.T) {
// 	ctx := context.Background()
// 	_, _, cache := setupInterfaces(t)

// 	shim := oci.NewContainerShim(cache)

// 	imageRef := oci_image_cache.ALPINE_LATEST.String()
// 	platform := units.PlatformLinuxAMD64
// 	containerID := "test-container-1"

// 	// Create a container
// 	container, err := shim.CreateContainer(ctx, imageRef, platform, containerID)
// 	require.NoError(t, err)
// 	assert.Equal(t, containerID, container.ID)
// 	assert.Equal(t, imageRef, container.ImageRef)
// 	assert.Equal(t, platform, container.Platform)
// 	assert.DirExists(t, container.RootfsPath)
// 	assert.FileExists(t, container.Ext4Path)

// 	// List containers
// 	containers, err := shim.ListContainers(ctx)
// 	require.NoError(t, err)
// 	assert.Len(t, containers, 1)
// 	assert.Equal(t, containerID, containers[0].ID)

// 	// Get specific container
// 	retrieved, err := shim.GetContainer(ctx, containerID)
// 	require.NoError(t, err)
// 	assert.Equal(t, container.ID, retrieved.ID)

// 	// Try to create duplicate container
// 	_, err = shim.CreateContainer(ctx, imageRef, platform, containerID)
// 	assert.Error(t, err)
// 	assert.Contains(t, err.Error(), "already exists")

// 	// Remove container
// 	err = shim.RemoveContainer(ctx, containerID)
// 	require.NoError(t, err)

// 	// Verify container is removed
// 	containers, err = shim.ListContainers(ctx)
// 	require.NoError(t, err)
// 	assert.Len(t, containers, 0)

// 	// Try to remove non-existent container
// 	err = shim.RemoveContainer(ctx, "non-existent")
// 	assert.Error(t, err)
// 	assert.Contains(t, err.Error(), "not found")

// 	// Try to get non-existent container
// 	_, err = shim.GetContainer(ctx, "non-existent")
// 	assert.Error(t, err)
// 	assert.Contains(t, err.Error(), "not found")
// }

// func TestConstants(t *testing.T) {
// 	// Verify constants are defined
// 	assert.Equal(t, "ec1-oci-cache", oci.DefaultCacheDir)
// 	assert.Equal(t, "metadata.json", oci.CacheMetadataFile)
// 	assert.Equal(t, "rootfs", oci.CacheRootfsDir)
// 	assert.Equal(t, "rootfs.ext4", oci.CacheExt4File)
// 	assert.Equal(t, 24*time.Hour, oci.DefaultCacheExpiration)
// 	assert.Equal(t, os.FileMode(0755), oci.CacheDirPerm)
// 	assert.Equal(t, os.FileMode(0644), oci.CacheFilePerm)
// 	assert.Equal(t, int64(1<<30), oci.DefaultExt4MaxSize)
// }
