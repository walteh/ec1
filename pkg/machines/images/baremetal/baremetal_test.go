package baremetal

import (
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/walteh/ec1/pkg/machines/guest"
)

func TestBareMetalProvider(t *testing.T) {
	provider := NewBareMetalProvider()

	assert.Equal(t, "baremetal", provider.Name(), "Provider name should be 'baremetal'")
	assert.Equal(t, "v1.0.0", provider.Version(), "Version should be v1.0.0")
	assert.Equal(t, guest.GuestKernelTypeLinux, provider.GuestKernelType(), "Kernel type should be Linux")
	assert.False(t, provider.SupportsEFI(), "Provider should not support EFI")

	// Check that the URL is correctly formatted
	url := provider.DiskImageURL()
	assert.Contains(t, url, "alpine-virt-3.19.1", "URL should contain Alpine version")
	assert.Contains(t, url, "alpine/v3.19/releases", "URL should point to Alpine releases")
}

func TestKernelDecompression(t *testing.T) {
	// Create a temporary directory for the test
	tempDir, err := os.MkdirTemp("", "kernel-test")
	require.NoError(t, err, "Failed to create temp directory")
	defer os.RemoveAll(tempDir)

	// Create a simple gzipped test file
	testData := []byte("This is test kernel data")
	compressedPath := filepath.Join(tempDir, "test-kernel.gz")
	uncompressedPath := filepath.Join(tempDir, "test-kernel-uncompressed")

	// Create a gzipped file
	file, err := os.Create(compressedPath)
	require.NoError(t, err, "Failed to create test file")

	gzipWriter := gzip.NewWriter(file)
	_, err = gzipWriter.Write(testData)
	require.NoError(t, err, "Failed to write test data")
	require.NoError(t, gzipWriter.Close(), "Failed to close gzip writer")
	require.NoError(t, file.Close(), "Failed to close file")

	// Test decompression
	err = decompressKernel(t.Context(), compressedPath, uncompressedPath)
	require.NoError(t, err, "Kernel decompression should succeed")

	// Verify the decompressed content
	result, err := os.ReadFile(uncompressedPath)
	require.NoError(t, err, "Failed to read uncompressed file")
	assert.Equal(t, testData, result, "Decompressed content should match the original")
}
