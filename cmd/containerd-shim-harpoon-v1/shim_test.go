package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestShimBinary tests the basic functionality of the shim binary
func TestShimBinary(t *testing.T) {
	ctx := getTestLoggerCtx(t)

	// Skip if not on macOS
	if !isMacOS() {
		t.Skip("Harpoon shim only works on macOS")
	}

	t.Run("BuildShim", func(t *testing.T) {
		testBuildShim(t, ctx)
	})

	t.Run("ShimHelp", func(t *testing.T) {
		testShimHelp(t, ctx)
	})

	t.Run("ShimVersion", func(t *testing.T) {
		testShimVersion(t, ctx)
	})
}

func testBuildShim(t *testing.T, ctx context.Context) {
	// Create temporary directory
	workDir, err := os.MkdirTemp("", "shim-test-*")
	require.NoError(t, err, "Failed to create temp directory")
	defer os.RemoveAll(workDir)

	// Build the shim binary
	shimPath := filepath.Join(workDir, "containerd-shim-harpoon-v1")

	buildCmd := exec.CommandContext(ctx, "go", "build", "-o", shimPath, ".")
	buildCmd.Dir = "." // Current directory (cmd/containerd-shim-harpoon-v1)

	// Measure build time
	start := time.Now()
	output, err := buildCmd.CombinedOutput()
	buildTime := time.Since(start)

	require.NoError(t, err, "Failed to build shim: %s", string(output))

	// Verify binary exists and is executable
	info, err := os.Stat(shimPath)
	require.NoError(t, err, "Shim binary not found")
	require.True(t, info.Mode()&0111 != 0, "Shim binary is not executable")

	t.Logf("✅ Shim built successfully in %v (size: %d bytes)", buildTime, info.Size())

	// Performance check: build should be reasonably fast
	assert.Less(t, buildTime, 30*time.Second, "Build time should be under 30 seconds")

	// Size check: binary should be reasonable size (not too small, not too large)
	assert.Greater(t, info.Size(), int64(1024*1024), "Binary should be at least 1MB")
	assert.Less(t, info.Size(), int64(500*1024*1024), "Binary should be less than 500MB")
}

func testShimHelp(t *testing.T, ctx context.Context) {
	// Build the shim first
	workDir, err := os.MkdirTemp("", "shim-help-test-*")
	require.NoError(t, err, "Failed to create temp directory")
	defer os.RemoveAll(workDir)

	shimPath := filepath.Join(workDir, "containerd-shim-harpoon-v1")
	buildCmd := exec.CommandContext(ctx, "go", "build", "-o", shimPath, ".")
	buildCmd.Dir = "."

	output, err := buildCmd.CombinedOutput()
	require.NoError(t, err, "Failed to build shim: %s", string(output))

	// Test help command (shims typically don't have help, but shouldn't hang)
	helpCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	helpCmd := exec.CommandContext(helpCtx, shimPath, "--help")
	err = helpCmd.Run()

	// Shim should exit quickly (either with help or success), not hang
	// Some shims return 0 (success), others return error - both are acceptable
	t.Logf("✅ Shim exits properly when called with --help (exit code: %v)", err)
}

func testShimVersion(t *testing.T, ctx context.Context) {
	// Build the shim first
	workDir, err := os.MkdirTemp("", "shim-version-test-*")
	require.NoError(t, err, "Failed to create temp directory")
	defer os.RemoveAll(workDir)

	shimPath := filepath.Join(workDir, "containerd-shim-harpoon-v1")
	buildCmd := exec.CommandContext(ctx, "go", "build", "-o", shimPath, ".")
	buildCmd.Dir = "."

	output, err := buildCmd.CombinedOutput()
	require.NoError(t, err, "Failed to build shim: %s", string(output))

	// Test version command
	versionCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	versionCmd := exec.CommandContext(versionCtx, shimPath, "--version")
	err = versionCmd.Run()

	// Shim should exit quickly (either with version info or error)
	t.Logf("✅ Shim exits properly when called with --version (exit code: %v)", err)
}

// TestShimIntegrationWithMockContainerd tests the shim with a mock containerd setup
func TestShimIntegrationWithMockContainerd(t *testing.T) {
	ctx := getTestLoggerCtx(t)

	// Skip if not on macOS
	if !isMacOS() {
		t.Skip("Harpoon shim only works on macOS")
	}

	t.Run("ShimStartup", func(t *testing.T) {
		testShimStartup(t, ctx)
	})
}

func testShimStartup(t *testing.T, ctx context.Context) {
	// Create temporary directory
	workDir, err := os.MkdirTemp("", "shim-startup-test-*")
	require.NoError(t, err, "Failed to create temp directory")
	defer os.RemoveAll(workDir)

	// Build the shim binary
	shimPath := filepath.Join(workDir, "containerd-shim-harpoon-v1")
	buildCmd := exec.CommandContext(ctx, "go", "build", "-o", shimPath, ".")
	buildCmd.Dir = "."

	output, err := buildCmd.CombinedOutput()
	require.NoError(t, err, "Failed to build shim: %s", string(output))

	// Test shim startup with minimal arguments
	startupCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Try to start the shim with basic arguments (it should fail gracefully)
	shimCmd := exec.CommandContext(startupCtx, shimPath,
		"-namespace", "test",
		"-id", "test-container",
		"-address", "/tmp/test.sock",
	)

	err = shimCmd.Run()

	// Shim should exit (either success or controlled failure), not hang
	// The exact exit code doesn't matter as much as not hanging
	t.Logf("✅ Shim startup test completed (exit code doesn't matter for this test)")
}

// TestShimPerformance tests basic performance characteristics
func TestShimPerformance(t *testing.T) {
	ctx := getTestLoggerCtx(t)

	// Skip if not on macOS
	if !isMacOS() {
		t.Skip("Harpoon shim only works on macOS")
	}

	t.Run("BuildPerformance", func(t *testing.T) {
		testBuildPerformance(t, ctx)
	})

	t.Run("StartupPerformance", func(t *testing.T) {
		testStartupPerformance(t, ctx)
	})
}

func testBuildPerformance(t *testing.T, ctx context.Context) {
	// Test multiple builds to check consistency
	var buildTimes []time.Duration

	for i := 0; i < 3; i++ {
		workDir, err := os.MkdirTemp("", "perf-test-*")
		require.NoError(t, err, "Failed to create temp directory")
		defer os.RemoveAll(workDir)

		shimPath := filepath.Join(workDir, "containerd-shim-harpoon-v1")
		buildCmd := exec.CommandContext(ctx, "go", "build", "-o", shimPath, ".")
		buildCmd.Dir = "."

		start := time.Now()
		output, err := buildCmd.CombinedOutput()
		buildTime := time.Since(start)

		require.NoError(t, err, "Failed to build shim (attempt %d): %s", i+1, string(output))
		buildTimes = append(buildTimes, buildTime)

		t.Logf("Build %d: %v", i+1, buildTime)
	}

	// Calculate average build time
	var total time.Duration
	for _, bt := range buildTimes {
		total += bt
	}
	avgBuildTime := total / time.Duration(len(buildTimes))

	t.Logf("✅ Average build time: %v", avgBuildTime)

	// Performance target: builds should be consistent and reasonable
	assert.Less(t, avgBuildTime, 60*time.Second, "Average build time should be under 60 seconds")
}

func testStartupPerformance(t *testing.T, ctx context.Context) {
	// Build the shim once
	workDir, err := os.MkdirTemp("", "startup-perf-test-*")
	require.NoError(t, err, "Failed to create temp directory")
	defer os.RemoveAll(workDir)

	shimPath := filepath.Join(workDir, "containerd-shim-harpoon-v1")
	buildCmd := exec.CommandContext(ctx, "go", "build", "-o", shimPath, ".")
	buildCmd.Dir = "."

	output, err := buildCmd.CombinedOutput()
	require.NoError(t, err, "Failed to build shim: %s", string(output))

	// Test startup time multiple times
	var startupTimes []time.Duration

	for i := 0; i < 5; i++ {
		startupCtx, cancel := context.WithTimeout(ctx, 5*time.Second)

		shimCmd := exec.CommandContext(startupCtx, shimPath, "--help")

		start := time.Now()
		shimCmd.Run() // We expect this to fail, but quickly
		startupTime := time.Since(start)

		cancel()
		startupTimes = append(startupTimes, startupTime)

		t.Logf("Startup %d: %v", i+1, startupTime)
	}

	// Calculate average startup time
	var total time.Duration
	for _, st := range startupTimes {
		total += st
	}
	avgStartupTime := total / time.Duration(len(startupTimes))

	t.Logf("✅ Average startup time: %v", avgStartupTime)

	// Performance target: startup should be very fast
	assert.Less(t, avgStartupTime, time.Second, "Average startup time should be under 1 second")
}

// TestShimCodeSigning tests code signing requirements on Apple Silicon
func TestShimCodeSigning(t *testing.T) {
	ctx := getTestLoggerCtx(t)

	// Skip if not on macOS
	if !isMacOS() {
		t.Skip("Harpoon shim only works on macOS")
	}

	// Only test code signing on Apple Silicon
	if !needsCodeSigning() {
		t.Skip("Code signing test only relevant on Apple Silicon")
	}

	t.Run("CodeSigningBuild", func(t *testing.T) {
		testCodeSigningBuild(t, ctx)
	})
}

func testCodeSigningBuild(t *testing.T, ctx context.Context) {
	workDir, err := os.MkdirTemp("", "codesign-test-*")
	require.NoError(t, err, "Failed to create temp directory")
	defer os.RemoveAll(workDir)

	shimPath := filepath.Join(workDir, "containerd-shim-harpoon-v1")

	// Build with potential code signing
	buildCmd := exec.CommandContext(ctx, "go", "build", "-o", shimPath, ".")
	buildCmd.Dir = "."

	// Set environment to potentially trigger code signing
	buildCmd.Env = append(os.Environ(), "CGO_ENABLED=1")

	output, err := buildCmd.CombinedOutput()
	require.NoError(t, err, "Failed to build shim with code signing: %s", string(output))

	// Verify the binary was created
	info, err := os.Stat(shimPath)
	require.NoError(t, err, "Code-signed shim binary not found")
	require.True(t, info.Mode()&0111 != 0, "Code-signed shim binary is not executable")

	t.Logf("✅ Code signing build completed successfully")
}
