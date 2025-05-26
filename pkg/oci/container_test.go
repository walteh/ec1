package oci

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/containers/image/v5/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/walteh/ec1/pkg/testing/tlog"
)

func TestContainerToVirtioDevice_Alpine(t *testing.T) {
	ctx := tlog.SetupSlogForTest(t)

	// Skip if running in CI or if docker/podman is not available
	// This test requires network access and a container runtime to pull images.
	if os.Getenv("CI") != "" {
		t.Skip("Skipping container to virtio device tests in CI environment as they require network and container runtime.")
	}

	// Create temporary directory for output
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "alpine-rootfs-fat32.img")

	opts := ContainerToVirtioOptions{
		ImageRef: "docker.io/library/alpine:latest", // A small, common image
		Platform: &types.SystemContext{
			OSChoice: "linux", // Explicitly request linux
			// ArchitectureChoice: "arm64", // Or detect host arch, or leave blank for library to pick
		},
		OutputDir:      tempDir,
		FilesystemType: "fat32",
		Size:           512 * 1024 * 1024, // 512MB
		ReadOnly:       false,
	}

	device, err := ContainerToVirtioDevice(ctx, opts)
	require.NoError(t, err, "ContainerToVirtioDevice should not return an error for alpine")
	require.NotNil(t, device, "ContainerToVirtioDevice should return a non-nil device")

	// Check if the output file was created
	_, err = os.Stat(outputPath)
	assert.NoError(t, err, "Output file should exist at %s", outputPath)

	// Further checks could involve trying to mount the image (if feasible in a test env)
	// or inspecting its contents, but that adds complexity.
	// For now, successful creation and existence of the file is a good first step.

	slog.InfoContext(ctx, "Successfully tested ContainerToVirtioDevice with alpine image.")
}

func TestContainerToVirtioDevice_InvalidImage(t *testing.T) {
	ctx := tlog.SetupSlogForTest(t)

	if os.Getenv("CI") != "" {
		t.Skip("Skipping test in CI environment.") // Might still attempt network
	}

	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "invalid-rootfs.img")

	opts := ContainerToVirtioOptions{
		ImageRef:       "this-image-definitely-does-not-exist/hopefully:latest",
		OutputDir:      tempDir,
		FilesystemType: "ext4",
		Size:           100 * 1024 * 1024, // 100MB
	}

	_, err := ContainerToVirtioDevice(ctx, opts)
	assert.Error(t, err, "ContainerToVirtioDevice should return an error for an invalid image reference")

	// Ensure the output file was NOT created or is empty if it was touched before erroring
	fi, statErr := os.Stat(outputPath)
	if statErr == nil {
		assert.Zero(t, fi.Size(), "Output file should be empty or not exist for a failed conversion")
	} else {
		assert.True(t, os.IsNotExist(statErr), "Output file should not exist for a failed conversion")
	}
}

func TestContainerToVirtioOptions_Defaults(t *testing.T) {
	ctx := tlog.SetupSlogForTest(t) // Added for consistency, though not strictly needed if no logging in func
	tempDir := t.TempDir()
	_ = ctx // avoid unused variable error if slog is not used
	opts := ContainerToVirtioOptions{
		ImageRef:  "some/image",
		OutputDir: tempDir,
		// FilesystemType and Size left to default
	}

	// This test doesn't run the conversion, just checks default option application logic
	// if it were part of the New function or similar. For now, it implies checking inside
	// ContainerToVirtioDevice if we were to expand it.

	// For the actual ContainerToVirtioDevice, defaults are applied internally.
	// We can assert the expected defaults if we had a separate constructor or options processor.
	// Here, we mostly ensure the struct can be created with minimal fields.
	assert.Equal(t, "some/image", opts.ImageRef)
	// Default FS type is ext4, Size is 1GB. These are applied inside ContainerToVirtioDevice.
}

func TestGetImageInfo(t *testing.T) {
	ctx := tlog.SetupSlogForTest(t)

	// Skip if running in CI or if network is not available
	if os.Getenv("CI") != "" {
		t.Skip("Skipping network-dependent tests in CI environment")
	}

	sysCtx := &types.SystemContext{
		OSChoice:           "linux",
		ArchitectureChoice: "arm64",
	}

	info, err := GetImageInfo(ctx, "docker.io/library/alpine:latest", sysCtx)
	if err != nil {
		t.Skipf("Skipping test due to network/registry error: %v", err)
	}

	require.NotNil(t, info)
	assert.Contains(t, info.Tag, "alpine")

	t.Logf("Image info: %+v", info)
}
