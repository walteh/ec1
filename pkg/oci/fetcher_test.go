package oci

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock command function for ExecImageFetcher that doesn't execute external commands
func mockSkopeoCommandFunc(ctx context.Context, imageRef string, ociLayoutPath string) *exec.Cmd {
	// Create a mock command that just creates the expected directory structure
	// rather than actually running skopeo
	mockCmd := exec.CommandContext(ctx, "true")
	
	// Create the necessary OCI layout structure directly instead of using PreRunE
	os.MkdirAll(filepath.Join(ociLayoutPath, "blobs", "sha256"), 0755)
	os.WriteFile(filepath.Join(ociLayoutPath, "index.json"), []byte(`{"schemaVersion":2}`), 0644)
	
	return mockCmd
}

func TestExecImageFetcher(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	
	// Create a fetcher that uses our mock command function
	fetcher := NewExecImageFetcherWithTempDir(tempDir, mockSkopeoCommandFunc)
	
	// Test fetching an image
	imageRef := "alpine:latest"
	ociLayoutPath, err := fetcher.FetchImageToOCILayout(ctx, imageRef)
	require.NoError(t, err)
	assert.NotEmpty(t, ociLayoutPath)
	assert.FileExists(t, filepath.Join(ociLayoutPath, "index.json"))
	assert.DirExists(t, filepath.Join(ociLayoutPath, "blobs", "sha256"))
}

func TestSkopeoCommandFunc(t *testing.T) {
	// Test that the real SkopeoCommandFunc produces the expected command
	ctx := context.Background()
	imageRef := "alpine:latest"
	ociLayoutPath := "/tmp/test-oci-layout"
	
	cmd := SkopeoCommandFunc(ctx, imageRef, ociLayoutPath)
	assert.Equal(t, "skopeo", filepath.Base(cmd.Path))
	assert.Contains(t, cmd.Args, "copy")
	assert.Contains(t, cmd.Args, "docker://alpine:latest")
	assert.Contains(t, cmd.Args, "oci:/tmp/test-oci-layout")
}

func TestHelperFunctions(t *testing.T) {
	// Test ociLayoutDirFromImageRef
	dir := ociLayoutDirFromImageRef("docker.io/library/alpine:latest")
	assert.Equal(t, "oci-layout-docker.io_library_alpine_latest", dir)
} 