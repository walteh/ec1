//go:build !libkrun && !libkrun_sev && !libkrun_efi

package libkrun

import (
	"context"
	"log/slog"
)

var (
	nextContextID uint32 = 1
)

// SetLogLevel sets the log level for libkrun (stub version)
func SetLogLevel(ctx context.Context, level LogLevel) error {
	log := slog.With(slog.String("component", "libkrun"))
	log.DebugContext(ctx, "setting libkrun log level (stub)", slog.Any("level", level))
	return ErrLibkrunNotAvailable
}

// CreateContext creates a new libkrun configuration context (stub version)
func CreateContext(ctx context.Context) (*Context, error) {
	log := slog.With(slog.String("component", "libkrun"))
	log.DebugContext(ctx, "creating libkrun context (stub)")
	return nil, ErrLibkrunNotAvailable
}

// Free frees the libkrun configuration context (stub version)
func (c *Context) Free(ctx context.Context) error {
	log := slog.With(slog.String("component", "libkrun"), slog.Uint64("ctx_id", uint64(c.id)))
	log.DebugContext(ctx, "freeing libkrun context (stub)")
	return ErrLibkrunNotAvailable
}

// SetVMConfig sets the basic configuration parameters for the microVM (stub version)
func (c *Context) SetVMConfig(ctx context.Context, config VMConfig) error {
	log := slog.With(slog.String("component", "libkrun"), slog.Uint64("ctx_id", uint64(c.id)))
	log.DebugContext(ctx, "setting VM config (stub)",
		slog.Int("num_vcpus", int(config.NumVCPUs)),
		slog.Uint64("ram_mib", uint64(config.RAMMiB)))
	return ErrLibkrunNotAvailable
}

// SetRoot sets the path to be used as root for the microVM (stub version)
func (c *Context) SetRoot(ctx context.Context, rootPath string) error {
	log := slog.With(slog.String("component", "libkrun"), slog.Uint64("ctx_id", uint64(c.id)))
	log.DebugContext(ctx, "setting VM root path (stub)", slog.String("root_path", rootPath))
	return ErrLibkrunNotAvailable
}

// SetRootDisk sets the root disk (stub version)
func (c *Context) SetRootDisk(ctx context.Context, diskPath string) error {
	log := slog.With(slog.String("component", "libkrun"), slog.Uint64("ctx_id", uint64(c.id)))
	log.DebugContext(ctx, "setting VM root disk (stub)", slog.String("disk_path", diskPath))
	return ErrLibkrunNotAvailable
}

// SetDataDisk sets the data disk (stub version)
func (c *Context) SetDataDisk(ctx context.Context, diskPath string) error {
	log := slog.With(slog.String("component", "libkrun"), slog.Uint64("ctx_id", uint64(c.id)))
	log.DebugContext(ctx, "setting VM data disk (stub)", slog.String("disk_path", diskPath))
	return ErrLibkrunNotAvailable
}

// AddDisk adds a disk (stub version)
func (c *Context) AddDisk(ctx context.Context, config DiskConfig) error {
	log := slog.With(slog.String("component", "libkrun"), slog.Uint64("ctx_id", uint64(c.id)))
	log.DebugContext(ctx, "adding VM disk (stub)",
		slog.String("block_id", config.BlockID),
		slog.String("path", config.Path),
		slog.Bool("read_only", config.ReadOnly))
	return ErrLibkrunNotAvailable
}

// AddDisk2 adds a disk with explicit format (stub version)
func (c *Context) AddDisk2(ctx context.Context, config DiskConfig) error {
	log := slog.With(slog.String("component", "libkrun"), slog.Uint64("ctx_id", uint64(c.id)))
	log.DebugContext(ctx, "adding VM disk with format (stub)",
		slog.String("block_id", config.BlockID),
		slog.String("path", config.Path),
		slog.Any("format", config.Format),
		slog.Bool("read_only", config.ReadOnly))
	return ErrLibkrunNotAvailable
}

// AddVirtioFS adds a virtio-fs device (stub version)
func (c *Context) AddVirtioFS(ctx context.Context, config VirtioFSConfig) error {
	log := slog.With(slog.String("component", "libkrun"), slog.Uint64("ctx_id", uint64(c.id)))
	log.DebugContext(ctx, "adding virtio-fs device (stub)",
		slog.String("tag", config.Tag),
		slog.String("path", config.Path))
	return ErrLibkrunNotAvailable
}

// SetNetwork configures networking (stub version)
func (c *Context) SetNetwork(ctx context.Context, config NetworkConfig) error {
	log := slog.With(slog.String("component", "libkrun"), slog.Uint64("ctx_id", uint64(c.id)))
	log.DebugContext(ctx, "setting network config (stub)", slog.Any("port_map", config.PortMap))
	return ErrLibkrunNotAvailable
}

// SetGPU configures GPU options (stub version)
func (c *Context) SetGPU(ctx context.Context, config GPUConfig) error {
	log := slog.With(slog.String("component", "libkrun"), slog.Uint64("ctx_id", uint64(c.id)))
	log.DebugContext(ctx, "setting GPU options (stub)", slog.Any("virgl_flags", config.VirglFlags))
	return ErrLibkrunNotAvailable
}

// SetProcess configures the process to run (stub version)
func (c *Context) SetProcess(ctx context.Context, config ProcessConfig) error {
	log := slog.With(slog.String("component", "libkrun"), slog.Uint64("ctx_id", uint64(c.id)))
	log.DebugContext(ctx, "setting process config (stub)",
		slog.String("exec_path", config.ExecPath),
		slog.Any("args", config.Args),
		slog.Int("env_count", len(config.Env)))
	return ErrLibkrunNotAvailable
}

// SetKernel configures kernel settings (stub version)
func (c *Context) SetKernel(ctx context.Context, config KernelConfig) error {
	log := slog.With(slog.String("component", "libkrun"), slog.Uint64("ctx_id", uint64(c.id)))
	log.DebugContext(ctx, "setting kernel config (stub)",
		slog.String("path", config.Path),
		slog.Any("format", config.Format),
		slog.String("cmdline", config.Cmdline))
	return ErrLibkrunNotAvailable
}

// AddVsockPorts adds vsock port mappings (stub version)
func (c *Context) AddVsockPorts(ctx context.Context, ports []VsockPort) error {
	log := slog.With(slog.String("component", "libkrun"), slog.Uint64("ctx_id", uint64(c.id)))
	log.DebugContext(ctx, "adding vsock ports (stub)", slog.Int("count", len(ports)))
	return ErrLibkrunNotAvailable
}

// SetSecurity configures security settings (stub version)
func (c *Context) SetSecurity(ctx context.Context, config SecurityConfig) error {
	log := slog.With(slog.String("component", "libkrun"), slog.Uint64("ctx_id", uint64(c.id)))
	log.DebugContext(ctx, "setting security config (stub)", slog.Any("rlimits", config.Rlimits))
	return ErrLibkrunNotAvailable
}

// SetAdvanced configures advanced settings (stub version)
func (c *Context) SetAdvanced(ctx context.Context, config AdvancedConfig) error {
	log := slog.With(slog.String("component", "libkrun"), slog.Uint64("ctx_id", uint64(c.id)))
	log.DebugContext(ctx, "setting advanced config (stub)")
	return ErrLibkrunNotAvailable
}

// SetMappedVolumes configures mapped volumes (stub version)
func (c *Context) SetMappedVolumes(ctx context.Context, mappedVolumes []string) error {
	log := slog.With(slog.String("component", "libkrun"), slog.Uint64("ctx_id", uint64(c.id)))
	log.DebugContext(ctx, "setting mapped volumes (stub)", slog.Any("mapped_volumes", mappedVolumes))
	return ErrLibkrunNotAvailable
}

// SetSEVConfig configures SEV-specific settings (stub version)
func (c *Context) SetSEVConfig(ctx context.Context, config SEVConfig) error {
	log := slog.With(slog.String("component", "libkrun"), slog.Uint64("ctx_id", uint64(c.id)))
	log.DebugContext(ctx, "setting SEV config (stub)")
	return ErrLibkrunNotAvailable
}

// GetShutdownEventFD returns the eventfd for orderly shutdown (stub version)
func (c *Context) GetShutdownEventFD(ctx context.Context) (int, error) {
	log := slog.With(slog.String("component", "libkrun"), slog.Uint64("ctx_id", uint64(c.id)))
	log.DebugContext(ctx, "getting shutdown eventfd (stub)")
	return 0, ErrLibkrunNotAvailable
}

// StartEnter starts and enters the microVM with the configured parameters (stub version)
func (c *Context) StartEnter(ctx context.Context) error {
	log := slog.With(slog.String("component", "libkrun"), slog.Uint64("ctx_id", uint64(c.id)))
	log.DebugContext(ctx, "starting VM (stub)")
	return ErrLibkrunNotAvailable
}
