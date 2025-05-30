package oci

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mholt/archives"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/tozd/go/errors"

	"github.com/walteh/ec1/pkg/units"

	oci_image_cache "github.com/walteh/ec1/gen/oci-image-cache"
)

// testDataImageDownloader implements ImageDownloader using local test data
type testDataImageDownloader struct {
	t testing.TB
}

func (d *testDataImageDownloader) DownloadImage(ctx context.Context, imageRef string) (string, error) {

	image, ok := oci_image_cache.Registry[oci_image_cache.OCICachedImage(imageRef)]
	if !ok {
		return "", errors.Errorf("image not found: %s", imageRef)
	}

	imageNameHash := sha256.Sum256([]byte(imageRef))

	destDir := filepath.Join(d.t.TempDir(), fmt.Sprintf("%x", imageNameHash[:8]))
	if _, err := os.Stat(destDir); !os.IsNotExist(err) {
		return destDir, nil
	}

	comp, err := (&archives.Xz{}).OpenReader(bytes.NewReader(image))
	if err != nil {
		return "", err
	}
	defer comp.Close()

	err = (&archives.Tar{}).Extract(ctx, comp, func(ctx context.Context, info archives.FileInfo) error {
		if info.IsDir() {
			err := os.MkdirAll(filepath.Join(destDir, info.NameInArchive), 0755)
			if err != nil {
				return err
			}
		} else {
			err := os.MkdirAll(filepath.Dir(filepath.Join(destDir, info.NameInArchive)), 0755)
			if err != nil {
				return err
			}

			fl, err := os.Create(filepath.Join(destDir, info.NameInArchive))
			if err != nil {
				return err
			}
			defer fl.Close()

			data, err := info.Open()
			if err != nil {
				return err
			}
			defer data.Close()

			_, err = io.Copy(fl, data)
			if err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return "", err
	}

	return destDir, nil
}

// func (d *testDataImageDownloader) extractTarToDir(tarPath, destDir string) error {
// 	file, err := os.Open(tarPath)
// 	if err != nil {
// 		return err
// 	}
// 	defer file.Close()

// 	if err := os.MkdirAll(destDir, 0755); err != nil {
// 		return err
// 	}

// 	tr := tar.NewReader(file)
// 	for {
// 		header, err := tr.Next()
// 		if err == io.EOF {
// 			break
// 		}
// 		if err != nil {
// 			return err
// 		}

// 		target := filepath.Join(destDir, header.Name)

// 		switch header.Typeflag {
// 		case tar.TypeDir:
// 			if err := os.MkdirAll(target, os.FileMode(header.Mode)); err != nil {
// 				return err
// 			}
// 		case tar.TypeReg:
// 			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
// 				return err
// 			}

// 			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
// 			if err != nil {
// 				return err
// 			}

// 			if _, err := io.Copy(f, tr); err != nil {
// 				f.Close()
// 				return err
// 			}
// 			f.Close()
// 		}
// 	}

// 	return nil
// }

func (d *testDataImageDownloader) createMinimalOCILayout(destDir string, platform units.Platform) error {
	// Create minimal OCI layout structure for unknown images (download simulation only)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return err
	}

	// Create oci-layout file
	ociLayout := map[string]string{"imageLayoutVersion": "1.0.0"}
	ociLayoutBytes, _ := json.Marshal(ociLayout)
	if err := os.WriteFile(filepath.Join(destDir, "oci-layout"), ociLayoutBytes, 0644); err != nil {
		return err
	}

	// Create blobs directory structure
	blobsDir := filepath.Join(destDir, "blobs", "sha256")
	if err := os.MkdirAll(blobsDir, 0755); err != nil {
		return err
	}

	// Create minimal config blob
	config := map[string]interface{}{
		"architecture": platform.Arch(),
		"os":           platform.OS(),
		"config": map[string]interface{}{
			"Env": []string{"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"},
			"Cmd": []string{"/bin/sh"},
		},
	}
	configBytes, _ := json.Marshal(config)
	configDigest := "config123"
	if err := os.WriteFile(filepath.Join(blobsDir, configDigest), configBytes, 0644); err != nil {
		return err
	}

	// Create minimal manifest
	manifest := map[string]interface{}{
		"schemaVersion": 2,
		"mediaType":     "application/vnd.oci.image.manifest.v1+json",
		"config": map[string]interface{}{
			"mediaType": "application/vnd.oci.image.config.v1+json",
			"size":      len(configBytes),
			"digest":    "sha256:" + configDigest,
		},
		"layers": []interface{}{}, // Empty layers for minimal image
	}
	manifestBytes, _ := json.Marshal(manifest)
	manifestDigest := "manifest123"
	if err := os.WriteFile(filepath.Join(blobsDir, manifestDigest), manifestBytes, 0644); err != nil {
		return err
	}

	// Create index
	index := map[string]interface{}{
		"schemaVersion": 2,
		"mediaType":     "application/vnd.oci.image.index.v1+json",
		"manifests": []map[string]interface{}{
			{
				"mediaType": "application/vnd.oci.image.manifest.v1+json",
				"size":      len(manifestBytes),
				"digest":    "sha256:" + manifestDigest,
				"platform": map[string]string{
					"architecture": platform.Arch(),
					"os":           platform.OS(),
				},
			},
		},
	}
	indexBytes, _ := json.Marshal(index)
	if err := os.WriteFile(filepath.Join(destDir, "index.json"), indexBytes, 0644); err != nil {
		return err
	}

	return nil
}

// mockPlatformFetcher implements PlatformFetcher for testing
type mockPlatformFetcher struct {
	platforms map[string][]*PlatformInfo
}

func (m *mockPlatformFetcher) FetchAvailablePlatforms(ctx context.Context, imageRef string) ([]*PlatformInfo, error) {
	if platforms, exists := m.platforms[imageRef]; exists {
		return platforms, nil
	}

	// Default platforms for unknown images
	return []*PlatformInfo{
		{
			Platform:       units.PlatformLinuxAMD64,
			ManifestDigest: "sha256:abc123",
			Size:           1024,
		},
		{
			Platform:       units.PlatformLinuxARM64,
			ManifestDigest: "sha256:def456",
			Size:           1024,
		},
	}, nil
}

// createTestCache creates a cache instance with mock implementations for testing
func createTestCache(t *testing.T) (*CacheV2, string) {
	// Create temporary directory for cache
	tempDir, err := os.MkdirTemp("", "ec1-cache-test-*")
	require.NoError(t, err)

	// Get the testdata directory

	// Setup mock implementations
	mockDownloader := &testDataImageDownloader{
		t: t,
	}

	mockFetcher := &mockPlatformFetcher{
		platforms: map[string][]*PlatformInfo{
			"alpine:latest": {
				{
					Platform:       units.PlatformLinuxAMD64,
					ManifestDigest: "sha256:alpine_amd64_digest",
					Size:           5000000, // 5MB
				},
				{
					Platform:       units.PlatformLinuxARM64,
					ManifestDigest: "sha256:alpine_arm64_digest",
					Size:           5200000, // 5.2MB
				},
			},
			"busybox:latest": {
				{
					Platform:       units.PlatformLinuxAMD64,
					ManifestDigest: "sha256:busybox_amd64_digest",
					Size:           2000000, // 2MB
				},
			},
		},
	}

	config := CacheConfig{
		CacheDir:   tempDir,
		Expiration: time.Hour, // 1 hour for tests
	}

	cache := NewCacheV2(config, mockDownloader, mockFetcher)

	// Cleanup function
	t.Cleanup(func() {
		os.RemoveAll(tempDir)
	})

	return cache, tempDir
}

func TestCacheV2Testable_BasicFlow(t *testing.T) {
	cache, _ := createTestCache(t)
	ctx := context.Background()

	// Test loading a container that doesn't exist in cache
	container, err := cache.LoadCachedContainer(ctx, "alpine:latest", units.PlatformLinuxAMD64)
	require.NoError(t, err)
	require.NotNil(t, container)

	assert.Equal(t, "alpine:latest", container.ImageRef)
	assert.Equal(t, units.PlatformLinuxAMD64, container.Platform)
	assert.NotEmpty(t, container.ManifestDigest)
	assert.NotEmpty(t, container.ReadonlyFSPath)
	assert.NotEmpty(t, container.ReadonlyExt4Path)
	assert.NotNil(t, container.Metadata)
}

func TestCacheV2Testable_MultiPlatformSupport(t *testing.T) {
	cache, _ := createTestCache(t)
	ctx := context.Background()

	// Load the same image for different platforms
	containerAMD64, err := cache.LoadCachedContainer(ctx, "alpine:latest", units.PlatformLinuxAMD64)
	require.NoError(t, err)

	containerARM64, err := cache.LoadCachedContainer(ctx, "alpine:latest", units.PlatformLinuxARM64)
	require.NoError(t, err)

	// Should have different manifest digests
	assert.NotEqual(t, containerAMD64.ManifestDigest, containerARM64.ManifestDigest)
	assert.Equal(t, "alpine:latest", containerAMD64.ImageRef)
	assert.Equal(t, "alpine:latest", containerARM64.ImageRef)
	assert.Equal(t, units.PlatformLinuxAMD64, containerAMD64.Platform)
	assert.Equal(t, units.PlatformLinuxARM64, containerARM64.Platform)
}

func TestCacheV2Testable_CacheHit(t *testing.T) {
	cache, _ := createTestCache(t)
	ctx := context.Background()

	// First load - cache miss
	container1, err := cache.LoadCachedContainer(ctx, "alpine:latest", units.PlatformLinuxAMD64)
	require.NoError(t, err)

	// Second load - should be cache hit
	container2, err := cache.LoadCachedContainer(ctx, "alpine:latest", units.PlatformLinuxAMD64)
	require.NoError(t, err)

	// Should return the same cached data
	assert.Equal(t, container1.ManifestDigest, container2.ManifestDigest)
	assert.Equal(t, container1.ReadonlyExt4Path, container2.ReadonlyExt4Path)
}

func TestCacheV2Testable_CacheExpiration(t *testing.T) {
	cache, _ := createTestCache(t)
	ctx := context.Background()

	// Load container
	_, err := cache.LoadCachedContainer(ctx, "alpine:latest", units.PlatformLinuxAMD64)
	require.NoError(t, err)

	// Manually expire the cache by modifying the metadata
	metadata, err := cache.loadImageCacheMetadata("alpine:latest")
	require.NoError(t, err)

	// Set expiration to past
	metadata.ExpiresAt = time.Now().Add(-time.Hour)
	err = cache.saveImageCacheMetadata(metadata)
	require.NoError(t, err)

	// Next load should refresh the cache
	container, err := cache.LoadCachedContainer(ctx, "alpine:latest", units.PlatformLinuxAMD64)
	require.NoError(t, err)
	assert.NotNil(t, container)
}

func TestCacheV2Testable_ListCachedImages(t *testing.T) {
	cache, _ := createTestCache(t)
	ctx := context.Background()

	// Initially should be empty
	images, err := cache.ListCachedImages(ctx)
	require.NoError(t, err)
	assert.Empty(t, images)

	// Load some containers
	_, err = cache.LoadCachedContainer(ctx, "alpine:latest", units.PlatformLinuxAMD64)
	require.NoError(t, err)

	_, err = cache.LoadCachedContainer(ctx, "busybox:latest", units.PlatformLinuxAMD64)
	require.NoError(t, err)

	// Should now have cached images
	images, err = cache.ListCachedImages(ctx)
	require.NoError(t, err)
	assert.Len(t, images, 2)

	// Check image references
	imageRefs := make([]string, len(images))
	for i, img := range images {
		imageRefs[i] = img.ImageRef
	}
	assert.Contains(t, imageRefs, "alpine:latest")
	assert.Contains(t, imageRefs, "busybox:latest")
}

func TestCacheV2Testable_CleanExpiredCache(t *testing.T) {
	cache, _ := createTestCache(t)
	ctx := context.Background()

	// Load some containers
	_, err := cache.LoadCachedContainer(ctx, "alpine:latest", units.PlatformLinuxAMD64)
	require.NoError(t, err)

	_, err = cache.LoadCachedContainer(ctx, "busybox:latest", units.PlatformLinuxAMD64)
	require.NoError(t, err)

	// Verify they're cached
	images, err := cache.ListCachedImages(ctx)
	require.NoError(t, err)
	assert.Len(t, images, 2)

	// Expire one of them
	metadata, err := cache.loadImageCacheMetadata("alpine:latest")
	require.NoError(t, err)
	metadata.ExpiresAt = time.Now().Add(-time.Hour)
	err = cache.saveImageCacheMetadata(metadata)
	require.NoError(t, err)

	// Clean expired cache
	err = cache.CleanExpiredCache(ctx)
	require.NoError(t, err)

	// Should only have one image left
	images, err = cache.ListCachedImages(ctx)
	require.NoError(t, err)
	assert.Len(t, images, 1)
	assert.Equal(t, "busybox:latest", images[0].ImageRef)
}

func TestCacheV2Testable_ClearCache(t *testing.T) {
	cache, _ := createTestCache(t)
	ctx := context.Background()

	// Load some containers
	_, err := cache.LoadCachedContainer(ctx, "alpine:latest", units.PlatformLinuxAMD64)
	require.NoError(t, err)

	// Verify cache exists
	images, err := cache.ListCachedImages(ctx)
	require.NoError(t, err)
	assert.Len(t, images, 1)

	// Clear cache
	err = cache.ClearCache(ctx)
	require.NoError(t, err)

	// Cache should be empty
	images, err = cache.ListCachedImages(ctx)
	require.NoError(t, err)
	assert.Empty(t, images)
}

func TestCacheV2Testable_UnsupportedPlatform(t *testing.T) {
	cache, _ := createTestCache(t)
	ctx := context.Background()

	// Try to load an unsupported platform
	_, err := cache.LoadCachedContainer(ctx, "alpine:latest", units.Platform("windows/amd64"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not supported")
}

func TestCacheV2Testable_OrphanedManifestCleanup(t *testing.T) {
	cache, _ := createTestCache(t)
	ctx := context.Background()

	// Load a container to create manifest cache
	_, err := cache.LoadCachedContainer(ctx, "alpine:latest", units.PlatformLinuxAMD64)
	require.NoError(t, err)

	// Manually remove the image metadata but leave manifest cache
	imageDir, err := cache.getImageCacheDir("alpine:latest")
	require.NoError(t, err)
	err = os.RemoveAll(imageDir)
	require.NoError(t, err)

	// Clean expired cache should remove orphaned manifests
	err = cache.CleanExpiredCache(ctx)
	require.NoError(t, err)

	// Verify manifests directory is cleaned up
	manifestsDir := filepath.Join(cache.config.CacheDir, "manifests")
	entries, err := os.ReadDir(manifestsDir)
	if err == nil {
		// If directory exists, it should be empty or contain no valid manifest dirs
		for _, entry := range entries {
			if entry.IsDir() {
				infoPath := filepath.Join(manifestsDir, entry.Name(), "info.json")
				_, err := os.Stat(infoPath)
				assert.True(t, os.IsNotExist(err), "Found orphaned manifest that wasn't cleaned up")
			}
		}
	}
}

// createTestCacheForBenchmark creates a cache instance for benchmarking
func createTestCacheForBenchmark(b *testing.B) (*CacheV2, string) {
	// Create temporary directory for cache
	tempDir, err := os.MkdirTemp("", "ec1-cache-test-*")
	if err != nil {
		b.Fatal(err)
	}

	// Get the testdata directory

	// Setup mock implementations
	mockDownloader := &testDataImageDownloader{
		t: b,
	}

	mockFetcher := &mockPlatformFetcher{
		platforms: map[string][]*PlatformInfo{
			"alpine:latest": {
				{
					Platform:       units.PlatformLinuxAMD64,
					ManifestDigest: "sha256:alpine_amd64_digest",
					Size:           5000000, // 5MB
				},
			},
		},
	}

	config := CacheConfig{
		CacheDir:   tempDir,
		Expiration: time.Hour, // 1 hour for tests
	}

	cache := NewCacheV2(config, mockDownloader, mockFetcher)

	// Cleanup function
	b.Cleanup(func() {
		os.RemoveAll(tempDir)
	})

	return cache, tempDir
}

// Benchmark tests
func BenchmarkCacheV2Testable_LoadCachedContainer(b *testing.B) {
	cache, _ := createTestCacheForBenchmark(b)
	ctx := context.Background()

	// Pre-load to ensure cache hit
	_, err := cache.LoadCachedContainer(ctx, "alpine:latest", units.PlatformLinuxAMD64)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := cache.LoadCachedContainer(ctx, "alpine:latest", units.PlatformLinuxAMD64)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCacheV2Testable_MetadataOperations(b *testing.B) {
	cache, _ := createTestCacheForBenchmark(b)

	metadata := &ImageCacheMetadata{
		ImageRef:      "test:latest",
		CachedAt:      time.Now(),
		ExpiresAt:     time.Now().Add(time.Hour),
		Platforms:     make(map[string]*PlatformManifest),
		ManifestCache: make(map[string]*ManifestCacheInfo),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := cache.saveImageCacheMetadata(metadata)
		if err != nil {
			b.Fatal(err)
		}

		_, err = cache.loadImageCacheMetadata("test:latest")
		if err != nil {
			b.Fatal(err)
		}
	}
}
