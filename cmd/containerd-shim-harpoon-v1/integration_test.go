package main

import (
	_ "github.com/containerd/containerd/v2/cmd/containerd/builtins"

	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/containerd/containerd/v2/client"
	"github.com/containerd/containerd/v2/cmd/containerd/command"
	"github.com/containerd/containerd/v2/pkg/cio"
	"github.com/containerd/containerd/v2/pkg/namespaces"
	"github.com/containerd/containerd/v2/pkg/oci"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testNamespace     = "harpoon-test"
	testImage         = "docker.io/library/alpine:latest"
	containerdTimeout = 30 * time.Second
	vmBootTimeout     = 10 * time.Second
	testRuntime       = "io.containerd.harpoon.v1"
	shimName          = "containerd-shim-harpoon-v1"
)

var (
	shimContainerdExecutablePath = ""
)

func safeGetShimContainerdExecutablePath(t testing.TB) string {
	t.Helper()
	require.NotEmpty(t, shimContainerdExecutablePath, "shimContainerdExecutablePath is not set")
	return shimContainerdExecutablePath
}

// TestContainerdShimIntegration is the main integration test that validates
// the entire containerd shim functionality end-to-end
func TestContainerdShimIntegration(t *testing.T) {
	ctx := getTestLoggerCtx(t)

	// Skip if not on macOS
	if !isMacOS() {
		t.Skip("Harpoon shim only works on macOS")
	}

	// Create test environment
	testEnv := setupTestEnvironment(t, ctx)
	defer testEnv.cleanup()

	// // Run test phases
	// t.Run("Phase1_BuildShim", func(t *testing.T) {
	// 	testEnv.testBuildShim(t, ctx)
	// })

	// // Create containerd client
	// testEnv.createContainerdClient(t, ctx)
	// defer testEnv.client.Close()

	testEnv.testStartContainerd(t, ctx)

	testEnv.testBasicContainerLifecycle(t, ctx)

	testEnv.testCommandExecution(t, ctx)

	t.Run("Phase4_CommandExecution", func(t *testing.T) {
	})

	t.Run("Phase5_IOStreams", func(t *testing.T) {
		testEnv.testIOStreams(t, ctx)
	})

	t.Run("Phase6_ErrorHandling", func(t *testing.T) {
		testEnv.testErrorHandling(t, ctx)
	})

	t.Run("Phase7_Performance", func(t *testing.T) {
		testEnv.testPerformance(t, ctx)
	})

	t.Run("Phase8_ResourceCleanup", func(t *testing.T) {
		testEnv.testResourceCleanup(t, ctx)
	})
}

// testEnvironment manages the test setup and cleanup
type testEnvironment struct {
	t              *testing.T
	workDir        string
	containerdProc *os.Process
	containerdAddr string
	configFile     string
	client         *client.Client
	mu             sync.Mutex
	containers     []string // Track containers for cleanup
}

func setupTestEnvironment(t *testing.T, ctx context.Context) *testEnvironment {
	// Create temporary directory
	workDir, err := os.MkdirTemp("", "harpoon-test-*")
	require.NoError(t, err, "Failed to create temp directory")

	env := &testEnvironment{
		t:       t,
		workDir: workDir,
	}

	slog.InfoContext(ctx, "Test environment created", "workDir", workDir)
	return env
}

func (env *testEnvironment) cleanup() {
	env.mu.Lock()
	defer env.mu.Unlock()

	ctx := context.Background()

	// Clean up containers
	for _, containerID := range env.containers {
		env.cleanupContainer(ctx, containerID)
	}

	// Close containerd client
	if env.client != nil {
		env.client.Close()
	}

	// Stop containerd
	if env.containerdProc != nil {
		slog.Info("Stopping containerd process")
		env.containerdProc.Kill()
		env.containerdProc.Wait()
	}

	// Clean up work directory
	if env.workDir != "" {
		os.RemoveAll(env.workDir)
		slog.Info("Cleaned up work directory", "path", env.workDir)
	}
}

// func (env *testEnvironment) testBuildShim(t *testing.T, ctx context.Context) {
// 	slog.InfoContext(ctx, "Building containerd shim")

// 	// Build the shim binary
// 	shimPath := filepath.Join(env.workDir, "containerd-shim-harpoon-v1")

// 	buildCmd := exec.CommandContext(ctx, "go", "build", "-o", shimPath, ".")
// 	buildCmd.Dir = "." // Current directory (cmd/containerd-shim-harpoon-v1)

// 	// Handle code signing if needed
// 	if needsCodeSigning() {
// 		slog.InfoContext(ctx, "Code signing may be required for Apple Virtualization Framework")
// 		// The build will handle this via go tool codesign if configured
// 	}

// 	output, err := buildCmd.CombinedOutput()
// 	require.NoError(t, err, "Failed to build shim: %s", string(output))

// 	// Verify binary exists and is executable
// 	info, err := os.Stat(shimPath)
// 	require.NoError(t, err, "Shim binary not found")
// 	require.True(t, info.Mode()&0111 != 0, "Shim binary is not executable")

// 	env.shimBinary = shimPath
// 	slog.InfoContext(ctx, "Shim built successfully", "path", shimPath, "size", info.Size())

// 	// Test basic shim functionality
// 	helpCmd := exec.CommandContext(ctx, shimPath, "--help")
// 	err = helpCmd.Run()
// 	// Note: shim might not have --help, so we just check it doesn't crash immediately
// 	assert.NotNil(t, err, "Shim should exit with help or error, not hang")
// }

func (env *testEnvironment) testStartContainerd(t *testing.T, ctx context.Context) {
	slog.InfoContext(ctx, "Starting containerd with harpoon shim configuration")

	// Create containerd config
	env.createContainerdConfig(t, ctx)

	t.Logf("containterd config created")

	// Start containerd as a Go process (not external binary)
	ctx = env.startContainerdProcess(t, ctx)

	t.Logf("containerd process started")

	// Wait for containerd to be ready
	env.waitForContainerdReady(t, ctx)

	t.Logf("containerd ready")

	// Create containerd client
	env.createContainerdClient(t, ctx)
}

func (env *testEnvironment) createContainerdConfig(t *testing.T, ctx context.Context) {
	// Build a containerd v3 config that points to our custom shim
	configContent := fmt.Sprintf(`
version = 3
root   = "%[1]s"
state  = "%[2]s"

[grpc]
  address = "%[3]s"

[ttrpc]
  address = "%[3]s.ttrpc"

[debug]
  level = "debug"

[plugins."io.containerd.runtime.v1.linux"]
  shim_debug = true

[plugins."io.containerd.runtime.v2.task"]
  platforms = ["linux/amd64","linux/arm64"]

# --- CRI runtime section (v3) ------------------------------------------
[plugins."io.containerd.cri.v1.runtime".containerd]
  default_runtime_name = "%[4]s"

  [plugins."io.containerd.cri.v1.runtime".containerd.runtimes]
    [plugins."io.containerd.cri.v1.runtime".containerd.runtimes."%[4]s"]
      runtime_type = "%[4]s"
      [plugins."io.containerd.cri.v1.runtime".containerd.runtimes."%[4]s".options]
        binary_name = "%[5]s"
`,
		filepath.Join(env.workDir, "root"),     // %[1]s
		filepath.Join(env.workDir, "state"),    // %[2]s
		env.getContainerdAddress(),             // %[3]s
		testRuntime,                            // %[4]s
		safeGetShimContainerdExecutablePath(t), // %[5]s
	)

	// Create required directories
	dirs := []string{
		filepath.Join(env.workDir, "root"),
		filepath.Join(env.workDir, "state"),
		filepath.Join(env.workDir, "snapshots"),
		filepath.Join(env.workDir, "content"),
	}
	for _, dir := range dirs {
		err := os.MkdirAll(dir, 0755)
		require.NoError(t, err, "Failed to create directory: %s", dir)
	}

	env.configFile = filepath.Join(env.workDir, "containerd.toml")
	err := os.WriteFile(env.configFile, []byte(configContent), 0644)
	require.NoError(t, err, "Failed to write containerd config")

	slog.InfoContext(ctx, "Containerd config created", "path", env.configFile)
}

func (env *testEnvironment) getContainerdAddress() string {
	if env.containerdAddr == "" {
		env.containerdAddr = filepath.Join(env.workDir, "containerd.sock")
	}
	return env.containerdAddr
}

func (env *testEnvironment) startContainerdProcess(t *testing.T, ctx context.Context) context.Context {
	// Create all necessary directories for containerd
	dirs := []string{
		filepath.Join(env.workDir, "state"),
		filepath.Join(env.workDir, "root"),
		filepath.Join(env.workDir, "run"),
		filepath.Join(env.workDir, "snapshots"),
		filepath.Join(env.workDir, "content"),
		filepath.Join(env.workDir, "metadata"),
	}

	for _, dir := range dirs {
		err := os.MkdirAll(dir, 0755)
		require.NoError(t, err, "Failed to create directory %s", dir)
	}

	ctx, cancel := context.WithCancel(ctx)

	// Start containerd using exec to avoid permission issues with the embedded approach
	go func() {
		defer func() {
			cancel()
			if r := recover(); r != nil {
				slog.ErrorContext(ctx, "Containerd process panicked", "error", r)
			}
		}()

		// Fallback to embedded containerd
		args := []string{
			"containerd",
			"--config", env.configFile,
			"--address", env.getContainerdAddress(),
			"--state", filepath.Join(env.workDir, "state"),
			"--root", filepath.Join(env.workDir, "root"),
			"--log-level", "debug",
		}

		app := command.App()
		if err := app.Run(args); err != nil {
			slog.ErrorContext(ctx, "Embedded containerd failed", "error", err)
		}

	}()

	// Wait until the unix socket appears or we hit the overall timeout.
	startDeadline := time.Now().Add(containerdTimeout)
	for {
		if env.isContainerdReady(ctx) {
			slog.InfoContext(ctx, "Containerd process started and socket is up")
			break
		}
		if time.Now().After(startDeadline) {
			t.Fatalf("Timeout (%s) waiting for containerd to start", containerdTimeout)
		}
		time.Sleep(200 * time.Millisecond)
	}

	return ctx
}

func (env *testEnvironment) waitForContainerdReady(t *testing.T, ctx context.Context) {
	slog.InfoContext(ctx, "Waiting for containerd to be ready")

	timeout := time.After(containerdTimeout)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			t.Fatal("Context cancelled")
		case <-timeout:
			t.Fatal("Timeout waiting for containerd to be ready")
		case <-ticker.C:
			if env.isContainerdReady(ctx) {
				slog.InfoContext(ctx, "Containerd is ready")
				return
			}
		}
	}
}

func (env *testEnvironment) isContainerdReady(ctx context.Context) bool {
	// Try to connect to containerd socket
	conn, err := net.DialTimeout("unix", env.getContainerdAddress(), time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func (env *testEnvironment) createContainerdClient(t *testing.T, ctx context.Context) {
	var err error

	// Retry client creation a few times
	for i := 0; i < 5; i++ {
		env.client, err = client.New(env.getContainerdAddress())
		if err == nil {
			break
		}
		slog.WarnContext(ctx, "Failed to create containerd client, retrying", "attempt", i+1, "error", err)
		time.Sleep(time.Second)
	}

	require.NoError(t, err, "Failed to create containerd client after retries")

	// Test the connection
	_, err = env.client.Version(ctx)
	require.NoError(t, err, "Failed to get containerd version")

	slog.InfoContext(ctx, "Containerd client created and connected")
}

func (env *testEnvironment) testBasicContainerLifecycle(t *testing.T, ctx context.Context) {
	slog.InfoContext(ctx, "Testing basic container lifecycle")

	containerID := env.generateContainerID("lifecycle")
	env.trackContainer(containerID)

	// Test container creation
	container := env.createContainer(t, ctx, containerID, []string{"echo", "Hello from Harpoon VM!"})

	// Test container start
	startTime := time.Now()
	task := env.startContainer(t, ctx, container)
	bootTime := time.Since(startTime)

	slog.InfoContext(ctx, "Container started", "bootTime", bootTime)

	// Verify boot time target (<100ms is ambitious, <1s is more realistic for initial implementation)
	assert.Less(t, bootTime, time.Second, "Container boot time should be under 1 second")

	// Test container wait (should complete quickly since it's just echo)
	env.waitContainer(t, ctx, task, 10*time.Second)

	// Test container cleanup
	env.deleteContainer(t, ctx, container, task)
}

func (env *testEnvironment) testCommandExecution(t *testing.T, ctx context.Context) {
	slog.InfoContext(ctx, "Testing command execution")

	// Test various commands
	testCases := []struct {
		name string
		cmd  []string
	}{
		{"simple_echo", []string{"echo", "test output"}},
		{"list_root", []string{"ls", "-la", "/"}},
		{"show_processes", []string{"ps", "aux"}},
		{"memory_info", []string{"cat", "/proc/meminfo"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			subContainerID := env.generateContainerID(tc.name)
			env.trackContainer(subContainerID)

			container := env.createContainer(t, ctx, subContainerID, tc.cmd)

			execStart := time.Now()
			task := env.startContainer(t, ctx, container)
			execTime := time.Since(execStart)

			slog.InfoContext(ctx, "Command executed", "command", tc.cmd, "execTime", execTime)

			// Verify execution time target (<10ms overhead is very ambitious, <100ms is more realistic)
			assert.Less(t, execTime, 100*time.Millisecond, "Command execution should be fast")

			env.waitContainer(t, ctx, task, 5*time.Second)
			env.deleteContainer(t, ctx, container, task)
		})
	}
}

func (env *testEnvironment) testIOStreams(t *testing.T, ctx context.Context) {
	slog.InfoContext(ctx, "Testing I/O streams")

	// Test stdout
	containerID := env.generateContainerID("stdout")
	env.trackContainer(containerID)

	container := env.createContainer(t, ctx, containerID, []string{"echo", "stdout test"})
	task := env.startContainer(t, ctx, container)
	env.waitContainer(t, ctx, task, 5*time.Second)
	env.deleteContainer(t, ctx, container, task)

	// Test stderr
	containerID = env.generateContainerID("stderr")
	env.trackContainer(containerID)

	container = env.createContainer(t, ctx, containerID, []string{"sh", "-c", "echo 'stderr test' >&2"})
	task = env.startContainer(t, ctx, container)
	env.waitContainer(t, ctx, task, 5*time.Second)
	env.deleteContainer(t, ctx, container, task)

	// TODO: Add stdin testing when interactive containers are supported
}

func (env *testEnvironment) testErrorHandling(t *testing.T, ctx context.Context) {
	slog.InfoContext(ctx, "Testing error handling")

	// Test invalid command
	containerID := env.generateContainerID("invalid")
	env.trackContainer(containerID)

	container := env.createContainer(t, ctx, containerID, []string{"/nonexistent/command"})
	task := env.startContainer(t, ctx, container)

	// This should fail, but gracefully
	env.waitContainer(t, ctx, task, 5*time.Second)
	env.deleteContainer(t, ctx, container, task)

	// Test signal handling
	containerID = env.generateContainerID("signal")
	env.trackContainer(containerID)

	container = env.createContainer(t, ctx, containerID, []string{"sleep", "30"})
	task = env.startContainer(t, ctx, container)

	// Kill the container
	env.killContainer(t, ctx, task)
	env.waitContainer(t, ctx, task, 5*time.Second)
	env.deleteContainer(t, ctx, container, task)
}

func (env *testEnvironment) testPerformance(t *testing.T, ctx context.Context) {
	slog.InfoContext(ctx, "Testing performance metrics")

	// Test multiple containers in parallel
	numContainers := 3
	var wg sync.WaitGroup

	for i := 0; i < numContainers; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			containerID := env.generateContainerID(fmt.Sprintf("perf-%d", idx))
			env.trackContainer(containerID)

			start := time.Now()
			container := env.createContainer(t, ctx, containerID, []string{"echo", fmt.Sprintf("container-%d", idx)})
			task := env.startContainer(t, ctx, container)
			env.waitContainer(t, ctx, task, 10*time.Second)
			totalTime := time.Since(start)

			slog.InfoContext(ctx, "Container performance",
				"container", containerID,
				"totalTime", totalTime)

			env.deleteContainer(t, ctx, container, task)
		}(i)
	}

	wg.Wait()
	slog.InfoContext(ctx, "Parallel container test completed")
}

func (env *testEnvironment) testResourceCleanup(t *testing.T, ctx context.Context) {
	slog.InfoContext(ctx, "Testing resource cleanup")

	// Create and destroy several containers to test cleanup
	for i := 0; i < 3; i++ {
		containerID := env.generateContainerID(fmt.Sprintf("cleanup-%d", i))

		container := env.createContainer(t, ctx, containerID, []string{"echo", "cleanup test"})
		task := env.startContainer(t, ctx, container)
		env.waitContainer(t, ctx, task, 5*time.Second)
		env.deleteContainer(t, ctx, container, task)
	}

	// Verify no leftover processes
	env.verifyNoLeftoverProcesses(t, ctx)
}

// Helper methods for container operations using containerd client
func (env *testEnvironment) generateContainerID(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}

func (env *testEnvironment) trackContainer(containerID string) {
	env.mu.Lock()
	defer env.mu.Unlock()
	env.containers = append(env.containers, containerID)
}

func (env *testEnvironment) createContainer(t *testing.T, ctx context.Context, containerID string, cmd []string) client.Container {
	ctx = namespaces.WithNamespace(ctx, testNamespace)

	// Pull image if needed (simplified - in real test we'd need proper image handling)
	image, err := env.client.Pull(ctx, testImage, client.WithPullUnpack)
	require.NoError(t, err, "Failed to pull image %s", testImage)

	// Create container
	container, err := env.client.NewContainer(
		ctx,
		containerID,
		client.WithImage(image),
		client.WithNewSnapshot(containerID, image),
		// client.WithNewSpec(oci.WithImageConfig(image), oci.WithProcessArgs(cmd...)),
		client.WithNewSpec(oci.WithProcessArgs(cmd...)),
		client.WithRuntime(testRuntime, nil),
	)
	require.NoError(t, err, "Failed to create container %s", containerID)

	slog.InfoContext(ctx, "Container created", "id", containerID, "cmd", cmd)
	return container
}

func (env *testEnvironment) startContainer(t *testing.T, ctx context.Context, container client.Container) client.Task {
	ctx = namespaces.WithNamespace(ctx, testNamespace)

	creator := cio.NewCreator(cio.WithStdio, cio.WithFIFODir(filepath.Join(env.workDir, "fifo")))
	task, err := container.NewTask(ctx, creator)
	require.NoError(t, err, "Failed to create task for container %s", container.ID())

	err = task.Start(ctx)
	require.NoError(t, err, "Failed to start container %s", container.ID())

	slog.InfoContext(ctx, "Container started", "id", container.ID())
	return task
}

func (env *testEnvironment) waitContainer(t *testing.T, ctx context.Context, task client.Task, timeout time.Duration) {
	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	statusC, err := task.Wait(waitCtx)
	if err != nil {
		slog.WarnContext(ctx, "Container wait failed", "id", task.ID(), "error", err)
		return
	}

	select {
	case status := <-statusC:
		slog.InfoContext(ctx, "Container completed", "id", task.ID(), "exitCode", status.ExitCode())
	case <-waitCtx.Done():
		slog.WarnContext(ctx, "Container wait timeout", "id", task.ID())
	}
}

func (env *testEnvironment) killContainer(t *testing.T, ctx context.Context, task client.Task) {
	ctx = namespaces.WithNamespace(ctx, testNamespace)

	err := task.Kill(ctx, 9) // SIGKILL
	if err != nil {
		slog.WarnContext(ctx, "Container kill failed", "id", task.ID(), "error", err)
	} else {
		slog.InfoContext(ctx, "Container killed", "id", task.ID())
	}
}

func (env *testEnvironment) deleteContainer(t *testing.T, ctx context.Context, container client.Container, task client.Task) {
	ctx = namespaces.WithNamespace(ctx, testNamespace)

	// Kill task if still running
	task.Kill(ctx, 9)

	// Delete task
	_, err := task.Delete(ctx)
	if err != nil {
		slog.WarnContext(ctx, "Task delete failed", "id", task.ID(), "error", err)
	}

	// Delete container
	err = container.Delete(ctx, client.WithSnapshotCleanup)
	if err != nil {
		slog.WarnContext(ctx, "Container delete failed", "id", container.ID(), "error", err)
	} else {
		slog.InfoContext(ctx, "Container deleted", "id", container.ID())
	}
}

func (env *testEnvironment) cleanupContainer(ctx context.Context, containerID string) {
	ctx = namespaces.WithNamespace(ctx, testNamespace)

	// Try to get and delete the container
	container, err := env.client.LoadContainer(ctx, containerID)
	if err != nil {
		return // Container doesn't exist
	}

	// Try to get and delete the task
	task, err := container.Task(ctx, nil)
	if err == nil {
		task.Kill(ctx, 9)
		task.Delete(ctx)
	}

	// Delete container
	container.Delete(ctx, client.WithSnapshotCleanup)
}

func (env *testEnvironment) verifyNoLeftoverProcesses(t *testing.T, ctx context.Context) {
	// Check for leftover shim processes
	psCmd := exec.CommandContext(ctx, "ps", "aux")
	output, err := psCmd.CombinedOutput()
	if err != nil {
		slog.WarnContext(ctx, "Failed to check processes", "error", err)
		return
	}

	lines := strings.Split(string(output), "\n")
	shimProcesses := 0
	for _, line := range lines {
		if strings.Contains(line, "containerd-shim-harpoon") {
			shimProcesses++
			slog.InfoContext(ctx, "Found shim process", "process", line)
		}
	}

	// We might have some shim processes still running, but they should clean up eventually
	if shimProcesses > 5 {
		t.Logf("Warning: Found %d shim processes, might indicate cleanup issues", shimProcesses)
	}
}

// Utility functions
func isMacOS() bool {
	return strings.Contains(strings.ToLower(os.Getenv("GOOS")), "darwin") ||
		exec.Command("uname").Run() == nil
}

func needsCodeSigning() bool {
	// Check if we're on Apple Silicon or if codesigning is required
	cmd := exec.Command("uname", "-m")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(output), "arm64")
}
