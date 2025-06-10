package tcontainerd

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/containerd/containerd/v2/client"
	"gitlab.com/tozd/go/errors"
)

func NewContainerdClient(ctx context.Context) (*client.Client, error) {
	return client.New(Address(), client.WithDefaultNamespace(Namespace()), client.WithTimeout(Timeout()))
}

func LoadCurrentServerConfig(ctx context.Context) ([]byte, error) {

	// read the lock file
	running, err := isServerRunning(ctx)
	if err != nil {
		return nil, errors.Errorf("checking if server is running: %w", err)
	}
	if !running {
		return nil, errors.Errorf("server is already running, please stop it first")
	}

	configFile := filepath.Join(WorkDir(), "containerd.toml")

	config, err := os.ReadFile(configFile)
	if err != nil {
		return nil, errors.Errorf("reading config file: %w", err)
	}

	return config, nil
}

func (s *DevContainerdServer) createNerdctlConfig(ctx context.Context) error {
	logLevel := "info"
	if s.debug {
		logLevel = "debug"
	}

	configContent := fmt.Sprintf(`
debug          = false
debug_full     = false
address        = "unix://%[2]s"
namespace      = "%[3]s"
snapshotter    = "%[4]s"
# pull_policy    = "%[5]s"
# cgroup_manager = "cgroupfs"
# hosts_dir      = ["/etc/containerd/certs.d", "/etc/docker/certs.d"]
experimental   = true
# userns_remap   = ""
	`, logLevel == "debug", Address(), Namespace(), Snapshotter(), PullPolicy())

	if err := os.WriteFile(NerdctlConfigTomlPath(), []byte(configContent), 0644); err != nil {
		return errors.Errorf("writing nerdctl config: %w", err)
	}

	slog.InfoContext(ctx, "Created nerdctl config", "path", NerdctlConfigTomlPath())
	return nil
}

func (s *DevContainerdServer) createConfig(ctx context.Context) error {
	logLevel := "info"
	if s.debug {
		logLevel = "debug"
	}

	configContent := fmt.Sprintf(`
version = 3
root   = "%[1]s"
state  = "%[2]s"

[grpc]
  address = "%[3]s"

[ttrpc]
  address = "%[3]s.ttrpc"

[debug]
  level = "%[4]s"

[plugins."io.containerd.runtime.v1.linux"]
  shim_debug = true

# Register harpoon runtime for CRI
[plugins."io.containerd.cri.v1.runtime".containerd]
  default_runtime_name = "%[5]s"

  [plugins."io.containerd.cri.v1.runtime".containerd.runtimes]
    [plugins."io.containerd.cri.v1.runtime".containerd.runtimes."%[5]s"]
      runtime_type = "%[5]s"
      [plugins."io.containerd.cri.v1.runtime".containerd.runtimes."%[5]s".options]
        binary_name = "%[6]s"

# Snapshotter config
[plugins."io.containerd.snapshotter.v1.overlayfs"]
  root_path = "%[7]s/overlayfs"

[plugins."io.containerd.snapshotter.v1.native"]
  root_path = "%[7]s/native"

# Content store config
[plugins."io.containerd.content.v1.content"]
  path = "%[8]s"

# Garbage collection settings - delay GC to prevent race conditions during testing
[plugins."io.containerd.gc.v1.scheduler"]
  pause_threshold = 0.02
  deletion_threshold = 0
  mutation_threshold = 100
  schedule_delay = "0s"
  startup_delay = "10s"

# Metadata settings for content sharing
[plugins."io.containerd.metadata.v1.bolt"]
  content_sharing_policy = "shared"
`,
		ContainerdRootDir(),      // %[1]s
		ContainerdStateDir(),     // %[2]s
		Address(),                // %[3]s
		logLevel,                 // %[4]s
		shimRuntimeID,            // %[5]s
		ShimSimlinkPath(),        // %[6]s
		ContainerdSnapshotsDir(), // %[7]s
		ContainerdContentDir(),   // %[8]s
	)

	if err := os.WriteFile(ContainerdConfigTomlPath(), []byte(configContent), 0644); err != nil {
		return errors.Errorf("writing containerd config: %w", err)
	}

	slog.InfoContext(ctx, "Created containerd config", "path", ContainerdConfigTomlPath())
	return nil
}
