package oci

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/containers/image/v5/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContainerToVirtioDeviceCached(t *testing.T) {
	ctx := context.Background()

	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "oci-cache-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Test with a small Alpine Linux image
	opts := ContainerToVirtioOptions{
		ImageRef:   "docker.io/library/alpine:latest",
		OutputDir:  tempDir,
		MountPoint: filepath.Join(tempDir, "mount"),
		Platform: &types.SystemContext{
			OSChoice:           "linux",
			ArchitectureChoice: "arm64",
		},
		ReadOnly: true,
	}

	// First call should download and cache
	t.Logf("First call - should download and cache")
	device1, metadata1, err := ContainerToVirtioDeviceCached(ctx, opts)
	require.NoError(t, err, "Failed to convert container to virtio device (cached)")
	require.NotNil(t, device1, "Expected non-nil virtio device")
	require.NotNil(t, metadata1, "Expected non-nil container metadata")

	// Log the extracted metadata
	t.Logf("Container metadata:")
	t.Logf("  Entrypoint: %v", metadata1.Config.Entrypoint)
	t.Logf("  Cmd: %v", metadata1.Config.Cmd)
	t.Logf("  WorkingDir: %s", metadata1.Config.WorkingDir)
	t.Logf("  User: %s", metadata1.Config.User)
	t.Logf("  Env count: %d", len(metadata1.Config.Env))

	// Second call should use cache
	t.Logf("Second call - should use cache")
	device2, metadata2, err := ContainerToVirtioDeviceCached(ctx, opts)
	require.NoError(t, err, "Failed to convert container to virtio device (from cache)")
	require.NotNil(t, device2, "Expected non-nil virtio device from cache")
	require.NotNil(t, metadata2, "Expected non-nil container metadata from cache")

	// Verify metadata is the same
	assert.Equal(t, metadata1.Config.Entrypoint, metadata2.Config.Entrypoint, "Entrypoint should be the same")
	assert.Equal(t, metadata1.Config.Cmd, metadata2.Config.Cmd, "Cmd should be the same")
	assert.Equal(t, metadata1.Config.WorkingDir, metadata2.Config.WorkingDir, "WorkingDir should be the same")
}

func TestCacheExpiration(t *testing.T) {
	// Create a cache entry that's already expired
	cacheEntry := &CacheEntry{
		ImageRef:     "docker.io/library/alpine:latest",
		Platform:     "linux-arm64",
		CachedAt:     time.Now().Add(-25 * time.Hour), // 25 hours ago
		ExpiresAt:    time.Now().Add(-1 * time.Hour),  // 1 hour ago (expired)
		RootfsPath:   "/tmp/fake/rootfs",
		MetadataPath: "/tmp/fake/metadata.json",
		Size:         1024 * 1024, // 1MB
	}

	// Should be invalid due to expiration
	assert.False(t, isCacheValid(cacheEntry), "Expired cache should be invalid")

	// Create a cache entry that's still valid
	validCacheEntry := &CacheEntry{
		ImageRef:     "docker.io/library/alpine:latest",
		Platform:     "linux-arm64",
		CachedAt:     time.Now().Add(-1 * time.Hour), // 1 hour ago
		ExpiresAt:    time.Now().Add(23 * time.Hour), // 23 hours from now
		RootfsPath:   "/tmp/fake/rootfs",
		MetadataPath: "/tmp/fake/metadata.json",
		Size:         1024 * 1024, // 1MB
	}

	// Should be invalid due to missing files (even though not expired)
	assert.False(t, isCacheValid(validCacheEntry), "Cache with missing files should be invalid")
}

func TestGetCacheDirForImage(t *testing.T) {
	platform := &types.SystemContext{
		OSChoice:           "linux",
		ArchitectureChoice: "arm64",
	}

	cacheDir1, err := getCacheDirForImage("docker.io/library/alpine:latest", platform)
	require.NoError(t, err, "Failed to get cache dir")

	cacheDir2, err := getCacheDirForImage("docker.io/library/alpine:latest", platform)
	require.NoError(t, err, "Failed to get cache dir")

	// Same image and platform should produce same cache directory
	assert.Equal(t, cacheDir1, cacheDir2, "Same image and platform should produce same cache directory")

	// Different platform should produce different cache directory
	differentPlatform := &types.SystemContext{
		OSChoice:           "linux",
		ArchitectureChoice: "amd64",
	}

	cacheDir3, err := getCacheDirForImage("docker.io/library/alpine:latest", differentPlatform)
	require.NoError(t, err, "Failed to get cache dir")

	assert.NotEqual(t, cacheDir1, cacheDir3, "Different platforms should produce different cache directories")

	t.Logf("Cache directories:")
	t.Logf("  arm64: %s", cacheDir1)
	t.Logf("  amd64: %s", cacheDir3)
}

func TestGetPlatformString(t *testing.T) {
	tests := []struct {
		name     string
		platform *types.SystemContext
		expected string
	}{
		{
			name:     "nil platform",
			platform: nil,
			expected: "linux-amd64",
		},
		{
			name: "linux arm64",
			platform: &types.SystemContext{
				OSChoice:           "linux",
				ArchitectureChoice: "arm64",
			},
			expected: "linux-arm64",
		},
		{
			name: "empty fields",
			platform: &types.SystemContext{
				OSChoice:           "",
				ArchitectureChoice: "",
			},
			expected: "linux-amd64",
		},
		{
			name: "only os specified",
			platform: &types.SystemContext{
				OSChoice:           "linux",
				ArchitectureChoice: "",
			},
			expected: "linux-amd64",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getPlatformString(tt.platform)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestListCachedImages(t *testing.T) {
	ctx := context.Background()

	// This test will work even if no images are cached
	cachedImages, err := ListCachedImages(ctx)
	require.NoError(t, err, "Failed to list cached images")

	t.Logf("Found %d cached images", len(cachedImages))
	for i, entry := range cachedImages {
		t.Logf("  %d: %s (%s) - %d MB", i+1, entry.ImageRef, entry.Platform, entry.Size/1024/1024)
	}
}

func TestCleanExpiredCache(t *testing.T) {
	ctx := context.Background()

	// This test will work even if no expired cache exists
	err := CleanExpiredCache(ctx)
	require.NoError(t, err, "Failed to clean expired cache")

	t.Logf("Successfully cleaned expired cache entries")
}
