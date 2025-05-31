package oci_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	oci_image_cache "github.com/walteh/ec1/gen/oci-image-cache"
	"github.com/walteh/ec1/pkg/oci"
	"github.com/walteh/ec1/pkg/units"
)

func TestCachedFetcher(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	// Create a memory map fetcher with test data
	memFetcher := oci.NewMemoryMapFetcher(tempDir, oci_image_cache.Registry)
	
	// Wrap it with a cached fetcher
	cachedFetcher := oci.NewCachedFetcher(tempDir, memFetcher)

	// Test fetching an image
	imageRef := string(oci_image_cache.ALPINE_LATEST)
	ociLayoutPath, err := cachedFetcher.FetchImageToOCILayout(ctx, imageRef)
	require.NoError(t, err)
	assert.NotEmpty(t, ociLayoutPath)
	assert.FileExists(t, filepath.Join(ociLayoutPath, "index.json"))

	// Test fetching the same image again (should use cache)
	ociLayoutPath2, err := cachedFetcher.FetchImageToOCILayout(ctx, imageRef)
	require.NoError(t, err)
	assert.Equal(t, ociLayoutPath, ociLayoutPath2)

	// Test fetching a non-existent image
	_, err = cachedFetcher.FetchImageToOCILayout(ctx, "non-existent-image")
	assert.Error(t, err)
}

func TestMemoryMapFetcher(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	// Create a memory map fetcher with test data
	memFetcher := oci.NewMemoryMapFetcher(tempDir, oci_image_cache.Registry)

	// Test fetching an image
	imageRef := string(oci_image_cache.ALPINE_LATEST)
	ociLayoutPath, err := memFetcher.FetchImageToOCILayout(ctx, imageRef)
	require.NoError(t, err)
	assert.NotEmpty(t, ociLayoutPath)
	assert.FileExists(t, filepath.Join(ociLayoutPath, "index.json"))

	// Test fetching a non-existent image
	_, err = memFetcher.FetchImageToOCILayout(ctx, "non-existent-image")
	assert.Error(t, err)
}

func TestOCIFilesystemConverter(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	// Create a memory map fetcher with test data
	memFetcher := oci.NewMemoryMapFetcher(tempDir, oci_image_cache.Registry)

	// Fetch an image
	imageRef := string(oci_image_cache.ALPINE_LATEST)
	ociLayoutPath, err := memFetcher.FetchImageToOCILayout(ctx, imageRef)
	require.NoError(t, err)

	// Create converter
	converter := oci.NewOCIFilesystemConverter()

	// Convert the image
	platform := units.PlatformLinuxAMD64
	image, err := converter.ConvertOCILayoutToRootfsAndExt4(ctx, ociLayoutPath, platform)
	require.NoError(t, err)
	assert.NotNil(t, image)
	assert.NotEmpty(t, image.RootfsPath)
	assert.NotEmpty(t, image.Ext4Path)
	assert.FileExists(t, image.Ext4Path)
	assert.DirExists(t, image.RootfsPath)
	assert.Equal(t, platform, image.Platform)
	assert.NotNil(t, image.Metadata)
	assert.WithinDuration(t, time.Now(), image.CachedAt, 5*time.Second)
}

func TestCachedConverter(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	// Create a memory map fetcher with test data
	memFetcher := oci.NewMemoryMapFetcher(tempDir, oci_image_cache.Registry)

	// Fetch an image
	imageRef := string(oci_image_cache.ALPINE_LATEST)
	ociLayoutPath, err := memFetcher.FetchImageToOCILayout(ctx, imageRef)
	require.NoError(t, err)

	// Create converter and cached wrapper
	realConverter := oci.NewOCIFilesystemConverter()
	cachedConverter := oci.NewCachedConverter(realConverter)

	// Convert the image
	platform := units.PlatformLinuxAMD64
	image1, err := cachedConverter.ConvertOCILayoutToRootfsAndExt4(ctx, ociLayoutPath, platform)
	require.NoError(t, err)
	assert.NotNil(t, image1)

	// Convert again (should use cache)
	image2, err := cachedConverter.ConvertOCILayoutToRootfsAndExt4(ctx, ociLayoutPath, platform)
	require.NoError(t, err)
	assert.Equal(t, image1, image2)
}

func TestImageCache(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	// Create components
	memFetcher := oci.NewMemoryMapFetcher(tempDir, oci_image_cache.Registry)
	converter := oci.NewOCIFilesystemConverter()
	
	// Create cache
	cache := oci.NewImageCache(tempDir, memFetcher, converter)

	// Test it implements both interfaces
	imageRef := string(oci_image_cache.ALPINE_LATEST)
	platform := units.PlatformLinuxAMD64

	// Test fetcher interface
	ociLayoutPath, err := cache.FetchImageToOCILayout(ctx, imageRef)
	require.NoError(t, err)
	assert.NotEmpty(t, ociLayoutPath)
	assert.FileExists(t, filepath.Join(ociLayoutPath, "index.json"))

	// Test converter interface
	image, err := cache.ConvertOCILayoutToRootfsAndExt4(ctx, ociLayoutPath, platform)
	require.NoError(t, err)
	assert.NotNil(t, image)
	assert.NotEmpty(t, image.RootfsPath)
	assert.NotEmpty(t, image.Ext4Path)
	assert.FileExists(t, image.Ext4Path)
	assert.DirExists(t, image.RootfsPath)
}

func TestFetchAndConvertImage(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	// Create cache
	memFetcher := oci.NewMemoryMapFetcher(tempDir, oci_image_cache.Registry)
	converter := oci.NewOCIFilesystemConverter()
	cache := oci.NewImageCache(tempDir, memFetcher, converter)

	// Test the helper function
	imageRef := string(oci_image_cache.ALPINE_LATEST)
	platform := units.PlatformLinuxAMD64

	image, err := oci.FetchAndConvertImage(ctx, cache, imageRef, platform)
	require.NoError(t, err)
	assert.NotNil(t, image)
	assert.NotEmpty(t, image.RootfsPath)
	assert.NotEmpty(t, image.Ext4Path)
	assert.FileExists(t, image.Ext4Path)
	assert.DirExists(t, image.RootfsPath)
	assert.Equal(t, platform, image.Platform)
}

func TestImageSaveLoad(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	testFilePath := filepath.Join(tempDir, "test-image.json")

	// Create a test image
	testImage := &oci.Image{
		Platform:   units.PlatformLinuxAMD64,
		RootfsPath: "/path/to/rootfs",
		Ext4Path:   "/path/to/ext4",
		Metadata:   nil,
		CachedAt:   time.Now(),
	}

	// Save to file
	err := oci.SaveImageToCache(ctx, testFilePath, testImage)
	require.NoError(t, err)
	assert.FileExists(t, testFilePath)

	// Load from file
	loadedImage, err := oci.LoadImageFromCache(ctx, testFilePath)
	require.NoError(t, err)
	assert.Equal(t, testImage.Platform, loadedImage.Platform)
	assert.Equal(t, testImage.RootfsPath, loadedImage.RootfsPath)
	assert.Equal(t, testImage.Ext4Path, loadedImage.Ext4Path)
	assert.WithinDuration(t, testImage.CachedAt, loadedImage.CachedAt, time.Millisecond)

	// Test loading from non-existent file
	_, err = oci.LoadImageFromCache(ctx, filepath.Join(tempDir, "nonexistent.json"))
	assert.Error(t, err)
}

// Mock for testing the remote fetcher without actual network calls
type mockRemoteFetcher struct {
	oci.RemoteImageFetcher
	mockOCILayoutPath string
	callCount         int
}

func (m *mockRemoteFetcher) FetchImageToOCILayout(ctx context.Context, imageRef string) (string, error) {
	m.callCount++
	// Create a minimal OCI layout structure
	if err := os.MkdirAll(filepath.Join(m.mockOCILayoutPath, "blobs", "sha256"), 0755); err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(m.mockOCILayoutPath, "index.json"), []byte(`{"schemaVersion":2}`), 0644); err != nil {
		return "", err
	}
	return m.mockOCILayoutPath, nil
}

func TestRemoteImageFetcher(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	// Create mock remote fetcher
	mockFetcher := &mockRemoteFetcher{
		RemoteImageFetcher: oci.RemoteImageFetcher{CacheDir: tempDir},
		mockOCILayoutPath:  filepath.Join(tempDir, "mock-oci-layout"),
	}

	// Test with cached wrapper
	cachedFetcher := oci.NewCachedFetcher(tempDir, mockFetcher)

	// First call should use the remote fetcher
	ociLayoutPath1, err := cachedFetcher.FetchImageToOCILayout(ctx, "test:image")
	require.NoError(t, err)
	assert.Equal(t, 1, mockFetcher.callCount)
	assert.Equal(t, mockFetcher.mockOCILayoutPath, ociLayoutPath1)

	// Second call should use cache
	ociLayoutPath2, err := cachedFetcher.FetchImageToOCILayout(ctx, "test:image")
	require.NoError(t, err)
	assert.Equal(t, 1, mockFetcher.callCount) // Should not increment
	assert.Equal(t, ociLayoutPath1, ociLayoutPath2)
}

// Add a test for error cases in FetchAndConvertImage
func TestFetchAndConvertImageErrors(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	// Create components
	memFetcher := oci.NewMemoryMapFetcher(tempDir, oci_image_cache.Registry)
	converter := oci.NewOCIFilesystemConverter()
	
	// Create cache
	cache := oci.NewImageCache(tempDir, memFetcher, converter)

	// Test with non-existent image
	nonExistentImage := "non-existent-image:latest"
	_, err := oci.FetchAndConvertImage(ctx, cache, nonExistentImage, units.PlatformLinuxAMD64)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found in cache")

	// Create a mock fetcher that returns an error on fetch
	mockCache := &mockErrorCache{err: assert.AnError}
	_, err = oci.FetchAndConvertImage(ctx, mockCache, "any-image:latest", units.PlatformLinuxAMD64)
	assert.Error(t, err)
	assert.Equal(t, assert.AnError, errors.Unwrap(err))
}

// Mock cache implementation that returns errors
type mockErrorCache struct {
	err error
}

func (m *mockErrorCache) FetchImageToOCILayout(ctx context.Context, imageRef string) (string, error) {
	return "", m.err
}

func (m *mockErrorCache) ConvertOCILayoutToRootfsAndExt4(ctx context.Context, ociLayoutPath string, platform units.Platform) (*oci.Image, error) {
	return nil, m.err
}
