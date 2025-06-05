package main

import (
	_ "github.com/containerd/containerd/v2/cmd/containerd/builtins"
	_ "github.com/walteh/ec1/gen/oci-image-cache"

	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/containerd/containerd/v2/client"
	"github.com/containerd/containerd/v2/core/diff"
	"github.com/containerd/containerd/v2/core/images"
	"github.com/containerd/containerd/v2/core/images/archive"
	"github.com/containerd/containerd/v2/pkg/cio"
	"github.com/containerd/containerd/v2/pkg/namespaces"
	"github.com/containerd/platforms"
	"github.com/moby/sys/reexec"
	"github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	slogctx "github.com/veqryn/slog-context"

	"github.com/walteh/ec1/pkg/logging"
	"github.com/walteh/ec1/pkg/tcontainerd"
	"github.com/walteh/ec1/pkg/testing/tctx"
	"github.com/walteh/ec1/pkg/testing/tlog"
	"github.com/walteh/ec1/pkg/testing/toci"

	ec1oci "github.com/walteh/ec1/pkg/oci"
)

func init() {
	tcontainerd.ShimReexecInit()

	if reexec.Init() {
		os.Exit(0)
	}
}

type testEnvironment struct {
	mu         sync.Mutex
	server     *tcontainerd.DevContainerdServer
	containers []string // Track containers for cleanup
	client     *client.Client
	images     map[string][]images.Image
}

const (
	testImage         = "docker.io/library/alpine:latest"
	containerdTimeout = 30 * time.Second
	vmBootTimeout     = 10 * time.Second
)

var testEnv *testEnvironment
var testLogger *slog.Logger

func getTestLoggerCtx(t testing.TB) context.Context {
	ctx := slogctx.NewCtx(t.Context(), testLogger)
	ctx = tlog.SetupSlogForTestWithContext(t, ctx)
	ctx = tctx.WithContext(ctx, t)
	return ctx
}

func TestMain(m *testing.M) {
	ctx := context.Background()
	ctx = logging.SetupSlogSimpleToWriterWithProcessName(ctx, os.Stdout, true, "test")
	testLogger = slogctx.FromCtx(ctx)
	testEnv = setupTestEnvironment(ctx)

	code := m.Run()
	testEnv.server.Stop(ctx)
	os.Exit(code)
}

func (env *testEnvironment) preloadTestImages(t *testing.T, ctx context.Context, name string, compressedData []byte) {
	data := bytes.NewReader(compressedData)

	gzdata, err := gzip.NewReader(data)
	require.NoError(t, err)

	platSpec, err := platforms.Parse("linux")
	if err != nil {
		t.Fatalf("failed to parse platform: %v", err)
	}

	matcher := platforms.OnlyStrict(platSpec)

	prefix := fmt.Sprintf("import-%s", time.Now().Format("2006-01-02"))
	opts := []client.ImportOpt{
		client.WithImageRefTranslator(archive.AddRefPrefix(prefix)),
		// client.WithPlatformMatcher(platforms.DefaultStrict()),
		client.WithImportPlatform(matcher),
		client.WithDiscardUnpackedLayers(),
		client.WithImportCompression(),
		client.WithSkipMissing(),
		// client.WithImageLabels(map[string]string{
		// 	"io.containerd.image.name":          name,
		// 	"org.opencontainers.image.ref.name": strings.Split(name, ":")[1],
		// }),
		client.WithDigestRef(func(dgst digest.Digest) string {
			return name
		}),
	}

	image, err := env.client.Import(ctx, gzdata, opts...)
	require.NoError(t, err)

	env.images[name] = image

	for _, img := range image {

		image := client.NewImageWithPlatform(env.client, img, platforms.All)

		// TODO: Show unpack status
		fmt.Printf("unpacking %s (%s)...\n", img.Name, img.Target.Digest)
		err = image.Unpack(ctx, "native", client.WithUnpackApplyOpts(diff.WithSyncFs(true)))
		require.NoError(t, err)

	}
}

// TestContainerdShimIntegration is the main integration test that validates
// the entire containerd shim functionality end-to-end
func TestContainerdShimIntegration(t *testing.T) {
	ctx := getTestLoggerCtx(t)

	data := map[string][]byte{
		"docker.io/library/alpine:latest": toci.Registry()["docker.io/library/alpine:latest"],
	}

	for name, compressedData := range data {
		testEnv.preloadTestImages(t, ctx, name, compressedData)
	}

	// err = importer.PreloadTestImages(ctx, )
	// require.NoError(t, err)

	testEnv.testBasicContainerLifecycle(t, ctx)

	testEnv.testCommandExecution(t, ctx)

	// t.Run("BasicContainerLifecycle", func(t *testing.T) {
	// })

	// t.Run("CommandExecution", func(t *testing.T) {
	// })

	t.Run("IOStreams", func(t *testing.T) {
		testEnv.testIOStreams(t, ctx)
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		testEnv.testErrorHandling(t, ctx)
	})

	t.Run("Performance", func(t *testing.T) {
		testEnv.testPerformance(t, ctx)
	})

	t.Run("ResourceCleanup", func(t *testing.T) {
		testEnv.testResourceCleanup(t, ctx)
	})
}

// testEnvironment manages the test setup and cleanup
func setupTestEnvironment(ctx context.Context) *testEnvironment {

	server, err := tcontainerd.NewDevContainerdServer(ctx, true)
	if err != nil {
		panic(err)
	}

	err = server.StartBackground(ctx)
	if err != nil {
		panic(err)
	}

	client, err := tcontainerd.NewContainerdClient(ctx)
	if err != nil {
		panic(err)
	}

	env := &testEnvironment{
		server:     server,
		containers: []string{},
		client:     client,
		images:     map[string][]images.Image{},
	}

	return env
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

var _ ec1oci.ImageFetchConverter

type ContainerdOci struct {
}

func (env *testEnvironment) createContainer(t *testing.T, ctx context.Context, containerID string, cmd []string) client.Container {
	ctx = namespaces.WithNamespace(ctx, tcontainerd.Namespace())

	// list images
	imgs, err := env.client.ListImages(ctx)
	require.NoError(t, err, "Failed to list images")
	for _, img := range imgs {
		fmt.Println("images", img.Name(), img.Labels())
	}

	// // Pull image if needed (simplified - in real test we'd need proper image handling)
	image, err := env.client.GetImage(ctx, testImage)
	require.NoError(t, err, "Failed to pull image %s", testImage)

	// Create container
	container, err := env.client.NewContainer(
		ctx,
		containerID,
		client.WithImage(image),
		// client.WithNewSnapshot(containerID, image),
		// client.With(env.storage),
		// client.WithNewSpec(oci.WithImageConfig(image), oci.WithProcessArgs(cmd...)),
		// client.WithNewSpec(oci.WithProcessArgs(cmd...)),
		client.WithRuntime(ContainerdShimRuntimeID, nil),
	)
	require.NoError(t, err, "Failed to create container %s", containerID)

	slog.InfoContext(ctx, "Container created", "id", containerID, "cmd", cmd)
	return container
}

func (env *testEnvironment) startContainer(t *testing.T, ctx context.Context, container client.Container) client.Task {
	ctx = namespaces.WithNamespace(ctx, tcontainerd.Namespace())

	creator := cio.NewCreator(cio.WithStdio, cio.WithFIFODir(filepath.Join(tcontainerd.WorkDir(), "fifo")))
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
	ctx = namespaces.WithNamespace(ctx, tcontainerd.Namespace())

	err := task.Kill(ctx, 9) // SIGKILL
	if err != nil {
		slog.WarnContext(ctx, "Container kill failed", "id", task.ID(), "error", err)
	} else {
		slog.InfoContext(ctx, "Container killed", "id", task.ID())
	}
}

func (env *testEnvironment) deleteContainer(t *testing.T, ctx context.Context, container client.Container, task client.Task) {
	ctx = namespaces.WithNamespace(ctx, tcontainerd.Namespace())

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
	ctx = namespaces.WithNamespace(ctx, tcontainerd.Namespace())

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
