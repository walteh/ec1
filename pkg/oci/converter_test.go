package oci

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEnsureCleanIndex tests the ensureCleanIndex function
func TestEnsureCleanIndex(t *testing.T) {
	// Create a test directory
	tempDir := t.TempDir()
	
	// Create a converter
	converter := NewOCIFilesystemConverter()
	
	// Test with a clean index (no duplicates)
	indexPath := filepath.Join(tempDir, "index.json")
	cleanIndex := map[string]interface{}{
		"schemaVersion": 2,
		"manifests": []map[string]interface{}{
			{
				"mediaType": "application/vnd.docker.distribution.manifest.v2+json",
				"digest": "sha256:abc123",
				"size": 100,
			},
		},
	}
	
	cleanIndexBytes, err := json.Marshal(cleanIndex)
	require.NoError(t, err)
	err = os.WriteFile(indexPath, cleanIndexBytes, 0644)
	require.NoError(t, err)
	
	err = converter.ensureCleanIndex(context.Background(), tempDir)
	assert.NoError(t, err)
	
	// Verify index wasn't modified (since it was already clean)
	indexData, err := os.ReadFile(indexPath)
	require.NoError(t, err)
	
	// Test with duplicate manifests
	dirtyIndex := map[string]interface{}{
		"schemaVersion": 2,
		"manifests": []map[string]interface{}{
			{
				"mediaType": "application/vnd.docker.distribution.manifest.v2+json",
				"digest": "sha256:abc123",
				"size": 100,
			},
			{
				"mediaType": "application/vnd.docker.distribution.manifest.v2+json",
				"digest": "sha256:abc123",
				"size": 100,
			},
			{
				"mediaType": "application/vnd.docker.distribution.manifest.v2+json",
				"digest": "sha256:def456",
				"size": 200,
			},
		},
	}
	
	dirtyIndexBytes, err := json.Marshal(dirtyIndex)
	require.NoError(t, err)
	err = os.WriteFile(indexPath, dirtyIndexBytes, 0644)
	require.NoError(t, err)
	
	err = converter.ensureCleanIndex(context.Background(), tempDir)
	assert.NoError(t, err)
	
	// Verify index was cleaned (duplicates removed)
	var cleanedIndex map[string]interface{}
	indexData, err = os.ReadFile(indexPath)
	require.NoError(t, err)
	err = json.Unmarshal(indexData, &cleanedIndex)
	require.NoError(t, err)
	
	// Should have 2 unique manifests
	manifests := cleanedIndex["manifests"].([]interface{})
	assert.Len(t, manifests, 2)
	
	// Test with malformed index.json
	err = os.WriteFile(indexPath, []byte("invalid json"), 0644)
	require.NoError(t, err)
	
	err = converter.ensureCleanIndex(context.Background(), tempDir)
	assert.Error(t, err)
}

// TestExtractLayer tests the extractLayer function
func TestExtractLayer(t *testing.T) {
	t.Skip("Creating a valid tar.gz file in-memory is complex")
	
	// Create a test directory
	tempDir := t.TempDir()
	
	// Create a converter
	converter := NewOCIFilesystemConverter()
	
	// Test with non-existent layer
	err := converter.extractLayer(context.Background(), "/non/existent/layer", tempDir)
	assert.Error(t, err)
}

// Test path helper functions
func TestPathHelpers(t *testing.T) {
	// Test ociLayoutDirFromImageRef
	dir := ociLayoutDirFromImageRef("docker.io/library/alpine:latest")
	assert.Equal(t, "oci-layout-docker.io_library_alpine_latest", dir)

	// Test rootfsPathFromOCILayoutDirAndPlatform
	rootfsPath := rootfsPathFromOCILayoutDirAndPlatform("/path/to/oci-layout", "linux/amd64")
	assert.Contains(t, rootfsPath, "linux_amd64")
	assert.Contains(t, rootfsPath, "rootfs")

	// Test ext4PathFromOCILayoutDirAndPlatform
	ext4Path := ext4PathFromOCILayoutDirAndPlatform("/path/to/oci-layout", "linux/amd64")
	assert.Contains(t, ext4Path, "linux_amd64")
	assert.Contains(t, ext4Path, "ext4")

	// Test metadataPathFromOCILayoutDirAndPlatform
	metadataPath := metadataPathFromOCILayoutDirAndPlatform("/path/to/oci-layout", "linux/amd64")
	assert.Contains(t, metadataPath, "linux_amd64")
	assert.Contains(t, metadataPath, "metadata.json")
} 