package unzbootgo

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractKernel(t *testing.T) {
	// Skip this test unless we have a test EFI file
	testEFIFile := os.Getenv("TEST_EFI_FILE")
	if testEFIFile == "" {
		t.Skip("Skipping test: TEST_EFI_FILE environment variable not set")
	}

	// Create a temporary directory for the output
	tempDir, err := os.MkdirTemp("", "unzbootgo-test")
	require.NoError(t, err, "Failed to create temp dir")
	defer os.RemoveAll(tempDir)

	// Create output file path
	outputFile := filepath.Join(tempDir, "kernel.out")

	// Test ExtractKernel
	err = ExtractKernel(testEFIFile, outputFile)
	require.NoError(t, err, "ExtractKernel failed")

	// Verify output file exists and is not empty
	stat, err := os.Stat(outputFile)
	require.NoError(t, err, "Failed to stat output file")
	assert.Greater(t, stat.Size(), int64(0), "Output file is empty")

	t.Logf("Successfully extracted kernel of size %d bytes", stat.Size())
}

// TestNonEFIFile tests that non-EFI files are returned unchanged
func TestNonEFIFile(t *testing.T) {
	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "unzbootgo-test")
	require.NoError(t, err, "Failed to create temp dir")
	defer os.RemoveAll(tempDir)

	// Create a test file with non-EFI content
	testContent := []byte("This is not an EFI file")
	inputFile := filepath.Join(tempDir, "non-efi.txt")
	err = os.WriteFile(inputFile, testContent, 0644)
	require.NoError(t, err, "Failed to create test file")

	// Create output file path
	outputFile := filepath.Join(tempDir, "output.txt")

	// Test ExtractKernel
	err = ExtractKernel(inputFile, outputFile)
	require.NoError(t, err, "ExtractKernel failed")

	// Read the output file
	outputContent, err := os.ReadFile(outputFile)
	require.NoError(t, err, "Failed to read output file")

	// Verify the content is unchanged
	assert.Equal(t, testContent, outputContent, "Output content differs from input")
}
