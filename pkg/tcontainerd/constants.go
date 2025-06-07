package tcontainerd

import (
	"path/filepath"
	"time"
)

var (
	globalWorkDir           = "/tmp/tcontainerd"
	globalPersistentWorkDir = "/tmp/tcontainerd-persistent"
	namespace               = "harpoon"
	shimRuntimeID           = "io.containerd.harpoon.v2"
	shimName                = "containerd-shim-harpoon-v2"
	timeout                 = 10 * time.Second
	pullPolicy              = "missing"
	snapshotter             = "native"
)

func WorkDir() string                  { return globalWorkDir }
func PersistentWorkDir() string        { return globalPersistentWorkDir }
func ContainerdConfigTomlPath() string { return filepath.Join(WorkDir(), "containerd.toml") }
func NerdctlConfigTomlPath() string    { return filepath.Join(WorkDir(), "nerdctl.toml") }
func Namespace() string                { return namespace }
func Address() string                  { return filepath.Join(WorkDir(), "containerd.sock") }
func LockFile() string                 { return filepath.Join(PersistentWorkDir(), "lock.pid") }
func ShimSimlinkPath() string          { return filepath.Join(WorkDir(), "reexec", shimName) }
func CtrSimlinkPath() string           { return filepath.Join(WorkDir(), "reexec", "ctr") }
func ShimLogProxySockPath() string     { return filepath.Join(WorkDir(), "reexec-log-proxy.sock") }
func ShimRuntimeID() string            { return shimRuntimeID }
func ShimName() string                 { return shimName }
func Timeout() time.Duration           { return timeout }
func ContainerdRootDir() string        { return filepath.Join(PersistentWorkDir(), "root") }
func ContainerdStateDir() string       { return filepath.Join(PersistentWorkDir(), "state") }
func ContainerdContentDir() string     { return filepath.Join(PersistentWorkDir(), "content") }
func ContainerdSnapshotsDir() string   { return filepath.Join(PersistentWorkDir(), "snapshots") }
func PullPolicy() string               { return pullPolicy }
func Snapshotter() string              { return snapshotter }
