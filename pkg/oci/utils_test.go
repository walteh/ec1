package oci

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/walteh/ec1/pkg/units"
)

// TestMkdirAll tests the MkdirAll utility function
func TestMkdirAll(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Test creating a nested directory structure
	dirPath := filepath.Join(tempDir, "a", "b", "c")

	// Create a mock FS that just passes through to os
	mockFS := &mockFS{}

	err := MkdirAll(mockFS, dirPath, 0755)
	require.NoError(t, err)

	// Verify the directory was created
	info, err := os.Stat(dirPath)
	require.NoError(t, err)
	assert.True(t, info.IsDir())

	// Test creating a directory that already exists (should not error)
	err = MkdirAll(mockFS, dirPath, 0755)
	require.NoError(t, err)

	// Test with invalid permissions
	invalidDir := filepath.Join(tempDir, "invalid")
	// Create a file where we want to create a directory
	err = os.WriteFile(invalidDir, []byte("test"), 0644)
	require.NoError(t, err)

	// Now try to create a directory at the same path
	err = MkdirAll(mockFS, filepath.Join(invalidDir, "dir"), 0755)
	assert.Error(t, err)
}

// Mock filesystem that passes through to os
type mockFS struct {
	fs.FS
}

func (m *mockFS) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

// TestExtractCompressedOCI tests the extractCompressedOCI function
func TestExtractCompressedOCI(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Create minimal valid data - just a few bytes that will fail to extract
	// This at least tests the error paths
	data := []byte{0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00} // Invalid gzip header

	err := ExtractCompressedOCI(context.Background(), data, tempDir)
	assert.Error(t, err)
}

// TestIntegrationHelpers tests the integration helpers
func TestIntegrationHelpers(t *testing.T) {
	// Test platform string conversion
	platform := units.Platform("linux/amd64")
	// Make sure various path helpers work with this platform
	ociPath := "/path/to/oci-layout"

	rootfsPath := rootfsPathFromOCILayoutDirAndPlatform(ociPath, platform)
	assert.Contains(t, rootfsPath, "rootfs")
	assert.Contains(t, rootfsPath, "linux_amd64")

	ext4Path := ext4PathFromOCILayoutDirAndPlatform(ociPath, platform)
	assert.Contains(t, ext4Path, "ext4")
	assert.Contains(t, ext4Path, "linux_amd64")

	metadataPath := metadataPathFromOCILayoutDirAndPlatform(ociPath, platform)
	assert.Contains(t, metadataPath, "metadata.json")
	assert.Contains(t, metadataPath, "linux_amd64")
}
