package oci_test

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/containerd/containerd/v2/client"
	"github.com/containerd/containerd/v2/cmd/containerd/command"
	"github.com/containerd/containerd/v2/pkg/namespaces"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	oci_image_cache "github.com/walteh/ec1/gen/oci-image-cache"
)

// func TestContainerdTestdataImporter(t *testing.T) {
// 	ctx := context.Background()

// 	// Set up test containerd instance
// 	env := setupTestContainerd(t, ctx)
// 	defer env.cleanup()

// 	// Create importer
// 	importer, err := oci.NewContainerdTestdataImporter(env.containerdAddr, "test-namespace")
// 	require.NoError(t, err)
// 	defer importer.Close()

// 	// Preload test images
// 	err = importer.PreloadTestImages(ctx, toci.Registry())
// 	require.NoError(t, err)

// 	// Verify images are available
// 	available, err := importer.IsImageAvailable(ctx, string(oci_image_cache.ALPINE_LATEST))
// 	require.NoError(t, err)
// 	assert.True(t, available)

// 	// Test using containerd client directly
// 	testContainerdOperations(t, ctx, env.client)
// }

func testContainerdOperations(t *testing.T, ctx context.Context, containerdClient *client.Client) {
	ctx = namespaces.WithNamespace(ctx, "test-namespace")

	imageRef := string(oci_image_cache.ALPINE_LATEST)

	// Get the image (should be available from our preload)
	image, err := containerdClient.ImageService().Get(ctx, imageRef)
	require.NoError(t, err)
	assert.Equal(t, imageRef, image.Name)

	// You can now use containerd's native functionality:

	// 1. Export to OCI layout if needed
	// exporter := containerdClient.ContentStore()
	// ... use containerd's export functionality

	// 2. Create containers directly
	// container, err := containerdClient.NewContainer(ctx, "test-container",
	//     client.WithImage(image), ...)

	// 3. Use with snapshots/overlayfs
	// snapshotService := containerdClient.SnapshotService("overlayfs")
	// ... create snapshots for VM rootfs

	t.Logf("Successfully accessed image %s through containerd", imageRef)
}

// Test environment for containerd
type testContainerdEnv struct {
	workDir        string
	containerdAddr string
	configFile     string
	client         *client.Client
	cancel         context.CancelFunc
}

func setupTestContainerd(t *testing.T, ctx context.Context) *testContainerdEnv {
	t.Helper()
	sha256sum := sha256.Sum256([]byte(t.TempDir()))
	workDir, err := os.MkdirTemp("", fmt.Sprintf("%x", sha256sum[:8]))
	require.NoError(t, err)
	t.Cleanup(func() {
		os.RemoveAll(workDir)
	})

	env := &testContainerdEnv{
		workDir:        workDir,
		containerdAddr: filepath.Join(workDir, "containerd.sock"),
	}

	// Create containerd config
	env.createConfig(t)

	// Start containerd
	env.startContainerd(t, ctx)

	// Create client
	env.createClient(t, ctx)

	return env
}

func (env *testContainerdEnv) createConfig(t *testing.T) {
	configContent := `
version = 3
root = "` + filepath.Join(env.workDir, "root") + `"
state = "` + filepath.Join(env.workDir, "state") + `"

[grpc]
  address = "` + env.containerdAddr + `"

[debug]
  level = "debug"
`

	// Create required directories
	dirs := []string{
		filepath.Join(env.workDir, "root"),
		filepath.Join(env.workDir, "state"),
		filepath.Join(env.workDir, "content"),
		filepath.Join(env.workDir, "metadata"),
	}
	for _, dir := range dirs {
		err := os.MkdirAll(dir, 0755)
		require.NoError(t, err)
	}

	env.configFile = filepath.Join(env.workDir, "containerd.toml")
	err := os.WriteFile(env.configFile, []byte(configContent), 0644)
	require.NoError(t, err)
}

func (env *testContainerdEnv) startContainerd(t *testing.T, ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	env.cancel = cancel

	go func() {
		defer cancel()

		args := []string{
			"containerd",
			"--config", env.configFile,
			"--address", env.containerdAddr,
			"--log-level", "debug",
		}

		app := command.App()
		if err := app.Run(args); err != nil {
			t.Logf("Containerd stopped: %v", err)
		}
	}()

	// Wait for containerd to be ready
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(env.containerdAddr); err == nil {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}

	t.Fatal("Containerd failed to start within timeout")
}

func (env *testContainerdEnv) createClient(t *testing.T, ctx context.Context) {
	var err error

	// Retry client creation
	for i := 0; i < 10; i++ {
		env.client, err = client.New(env.containerdAddr)
		if err == nil {
			// Test connection
			if _, err = env.client.Version(ctx); err == nil {
				return
			}
		}
		time.Sleep(500 * time.Millisecond)
	}

	require.NoError(t, err, "Failed to create containerd client")
}

func (env *testContainerdEnv) cleanup() {
	if env.client != nil {
		env.client.Close()
	}
	if env.cancel != nil {
		env.cancel()
	}
}

// Example of how you'd use this in practice:
// func SZExampleContainerdDirectUsage() {
// 	ctx := context.Background()

// 	// In your real tests, you'd set up containerd with testdata
// 	containerdAddr := "/run/containerd/containerd.sock" // or test socket

// 	// Create importer and preload testdata
// 	importer, _ := oci.NewContainerdTestdataImporter(containerdAddr, "test")
// 	importer.PreloadTestImages(ctx, toci.Registry())
// 	defer importer.Close()

// 	// Now use containerd directly
// 	containerdClient, _ := client.New(containerdAddr)
// 	defer containerdClient.Close()

// 	ctx = namespaces.WithNamespace(ctx, "test")

// 	// Use containerd's native Pull (will find the image in content store)
// 	image, _ := containerdClient.Pull(ctx, string(oci_image_cache.ALPINE_LATEST))

// 	// Use containerd's native container creation
// 	container, _ := containerdClient.NewContainer(ctx, "my-container",
// 		client.WithImage(image),
// 		client.WithNewSnapshot("my-snapshot", image))

// 	// Use containerd's native task management
// 	task, _ := container.NewTask(ctx, nil)
// 	task.Start(ctx)

// 	// Clean up
// 	task.Delete(ctx)
// 	container.Delete(ctx)
// }
