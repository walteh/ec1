package tcontainerd

import (
	"path/filepath"
	"time"
)

var (
	globalWorkDir = "/tmp/tcontainerd"
	namespace     = "harpoon"
	shimRuntimeID = "io.containerd.harpoon.v2"
	shimName      = "containerd-shim-harpoon-v2"
	timeout       = 10 * time.Second
)

func WorkDir() string                  { return globalWorkDir }
func ContainerdConfigTomlPath() string { return filepath.Join(WorkDir(), "containerd.toml") }
func NerdctlConfigTomlPath() string    { return filepath.Join(WorkDir(), "nerdctl.toml") }
func Namespace() string                { return namespace }
func Address() string                  { return filepath.Join(WorkDir(), "containerd.sock") }
func LockFile() string                 { return filepath.Join(WorkDir(), "lock.pid") }
func ShimSimlinkPath() string          { return filepath.Join(WorkDir(), "reexec", shimName) }
func CtrSimlinkPath() string           { return filepath.Join(WorkDir(), "reexec", "ctr") }
func ShimLogProxySockPath() string     { return filepath.Join(WorkDir(), "reexec-log-proxy.sock") }
func ShimRuntimeID() string            { return shimRuntimeID }
func ShimName() string                 { return shimName }
func Timeout() time.Duration           { return timeout }
