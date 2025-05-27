package oci

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/containers/image/v5/types"
)

func TestContainerToVirtioDevice(t *testing.T) {
	ctx := context.Background()

	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "oci-test-*")
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

	device, err := ContainerToVirtioDevice(ctx, opts)
	if err != nil {
		t.Fatalf("Failed to convert container to virtio device: %v", err)
	}

	if device == nil {
		t.Fatal("Expected non-nil virtio device")
	}

	// Give FUSE a moment to mount
	time.Sleep(100 * time.Millisecond)

	// Verify the mount point exists and has content
	mountInfo, err := os.Stat(opts.MountPoint)
	if err != nil {
		t.Fatalf("Mount point does not exist: %v", err)
	}

	if !mountInfo.IsDir() {
		t.Fatal("Mount point is not a directory")
	}

	// Check for typical Alpine Linux directories
	expectedDirs := []string{"bin", "etc", "lib", "usr"}
	for _, dir := range expectedDirs {
		dirPath := filepath.Join(opts.MountPoint, dir)
		if _, err := os.Stat(dirPath); err != nil {
			t.Logf("Warning: Expected directory %s not found: %v", dir, err)
		} else {
			t.Logf("Found expected directory: %s", dir)
		}
	}

	// Test reading a file
	etcPath := filepath.Join(opts.MountPoint, "etc")
	if entries, err := os.ReadDir(etcPath); err == nil {
		t.Logf("Found %d entries in /etc", len(entries))
		for i, entry := range entries {
			if i < 5 { // Log first 5 entries
				t.Logf("  /etc/%s (dir: %v)", entry.Name(), entry.IsDir())
			}
		}
	}
}

func TestGetImageInfo(t *testing.T) {
	ctx := context.Background()

	sysCtx := &types.SystemContext{
		OSChoice:           "linux",
		ArchitectureChoice: "amd64",
	}

	info, err := GetImageInfo(ctx, "docker.io/library/alpine:latest", sysCtx)
	if err != nil {
		t.Fatalf("Failed to get image info: %v", err)
	}

	if info == nil {
		t.Fatal("Expected non-nil image info")
	}

	t.Logf("Image info: %+v", info)
}

func TestPullAndExtractImage(t *testing.T) {
	ctx := context.Background()

	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "oci-extract-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sysCtx := &types.SystemContext{
		OSChoice:           "linux",
		ArchitectureChoice: "amd64",
	}

	err = pullAndExtractImage(ctx, "docker.io/library/alpine:latest", tempDir, sysCtx)
	if err != nil {
		t.Fatalf("Failed to pull and extract image: %v", err)
	}

	// Verify extraction worked
	entries, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("Failed to read extracted directory: %v", err)
	}

	if len(entries) == 0 {
		t.Fatal("No files extracted from container image")
	}

	t.Logf("Extracted %d entries from container image", len(entries))
	for i, entry := range entries {
		if i < 10 { // Log first 10 entries
			t.Logf("  %s (dir: %v)", entry.Name(), entry.IsDir())
		}
	}
}
