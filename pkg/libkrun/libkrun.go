//go:build libkrun

package libkrun

/*
#cgo pkg-config: libkrun
#include <stdlib.h>
#include "libkrun.h"
*/
import "C"
import (
	"context"
	"log/slog"
	"unsafe"

	"gitlab.com/tozd/go/errors"
)

// SetLogLevel sets the log level for libkrun
func SetLogLevel(ctx context.Context, level LogLevel) error {
	log := slog.With(slog.String("component", "libkrun"))
	log.DebugContext(ctx, "setting libkrun log level", slog.Any("level", level))

	result := C.krun_set_log_level(C.uint32_t(level))
	if result < 0 {
		return errors.Errorf("setting libkrun log level: %d", result)
	}
	return nil
}

// CreateContext creates a new libkrun configuration context
func CreateContext(ctx context.Context) (*Context, error) {
	log := slog.With(slog.String("component", "libkrun"))
	log.DebugContext(ctx, "creating libkrun context")

	result := C.krun_create_ctx()
	if result < 0 {
		return nil, errors.Errorf("creating libkrun context: %d", result)
	}

	kctx := &Context{id: uint32(result)}
	log.DebugContext(ctx, "created libkrun context", slog.Uint64("ctx_id", uint64(kctx.id)))
	return kctx, nil
}

// Free frees the libkrun configuration context
func (c *Context) Free(ctx context.Context) error {
	log := slog.With(slog.String("component", "libkrun"), slog.Uint64("ctx_id", uint64(c.id)))
	log.DebugContext(ctx, "freeing libkrun context")

	result := C.krun_free_ctx(C.uint32_t(c.id))
	if result < 0 {
		return errors.Errorf("freeing libkrun context: %d", result)
	}
	return nil
}

// SetVMConfig sets the basic configuration parameters for the microVM
func (c *Context) SetVMConfig(ctx context.Context, config VMConfig) error {
	log := slog.With(slog.String("component", "libkrun"), slog.Uint64("ctx_id", uint64(c.id)))
	log.DebugContext(ctx, "setting VM config",
		slog.Int("num_vcpus", int(config.NumVCPUs)),
		slog.Uint64("ram_mib", uint64(config.RAMMiB)))

	result := C.krun_set_vm_config(C.uint32_t(c.id), C.uint8_t(config.NumVCPUs), C.uint32_t(config.RAMMiB))
	if result < 0 {
		return errors.Errorf("setting VM config: %d", result)
	}
	return nil
}

// SetRootDisk sets the root disk (DEPRECATED - use AddDisk instead)
func (c *Context) SetRootDisk(ctx context.Context, diskPath string) error {
	log := slog.With(slog.String("component", "libkrun"), slog.Uint64("ctx_id", uint64(c.id)))
	log.DebugContext(ctx, "setting VM root disk (deprecated)", slog.String("disk_path", diskPath))

	cDiskPath := C.CString(diskPath)
	defer C.free(unsafe.Pointer(cDiskPath))

	result := C.krun_set_root_disk(C.uint32_t(c.id), cDiskPath)
	if result < 0 {
		return errors.Errorf("setting VM root disk: %d", result)
	}
	return nil
}

// SetDataDisk sets the data disk (DEPRECATED - use AddDisk instead)
func (c *Context) SetDataDisk(ctx context.Context, diskPath string) error {
	log := slog.With(slog.String("component", "libkrun"), slog.Uint64("ctx_id", uint64(c.id)))
	log.DebugContext(ctx, "setting VM data disk (deprecated)", slog.String("disk_path", diskPath))

	cDiskPath := C.CString(diskPath)
	defer C.free(unsafe.Pointer(cDiskPath))

	result := C.krun_set_data_disk(C.uint32_t(c.id), cDiskPath)
	if result < 0 {
		return errors.Errorf("setting VM data disk: %d", result)
	}
	return nil
}

// AddDisk adds a disk with automatic format detection (Raw only)
func (c *Context) AddDisk(ctx context.Context, config DiskConfig) error {
	log := slog.With(slog.String("component", "libkrun"), slog.Uint64("ctx_id", uint64(c.id)))
	log.DebugContext(ctx, "adding VM disk",
		slog.String("block_id", config.BlockID),
		slog.String("path", config.Path),
		slog.Bool("read_only", config.ReadOnly))

	cBlockID := C.CString(config.BlockID)
	defer C.free(unsafe.Pointer(cBlockID))
	cPath := C.CString(config.Path)
	defer C.free(unsafe.Pointer(cPath))

	result := C.krun_add_disk(C.uint32_t(c.id), cBlockID, cPath, C.bool(config.ReadOnly))
	if result < 0 {
		return errors.Errorf("adding VM disk: %d", result)
	}
	return nil
}

// AddDisk2 adds a disk with explicit format
func (c *Context) AddDisk2(ctx context.Context, config DiskConfig) error {
	log := slog.With(slog.String("component", "libkrun"), slog.Uint64("ctx_id", uint64(c.id)))
	log.DebugContext(ctx, "adding VM disk with format",
		slog.String("block_id", config.BlockID),
		slog.String("path", config.Path),
		slog.Any("format", config.Format),
		slog.Bool("read_only", config.ReadOnly))

	cBlockID := C.CString(config.BlockID)
	defer C.free(unsafe.Pointer(cBlockID))
	cPath := C.CString(config.Path)
	defer C.free(unsafe.Pointer(cPath))

	result := C.krun_add_disk2(C.uint32_t(c.id), cBlockID, cPath, C.uint32_t(config.Format), C.bool(config.ReadOnly))
	if result < 0 {
		return errors.Errorf("adding VM disk with format: %d", result)
	}
	return nil
}

// AddVirtioFS adds a virtio-fs device
func (c *Context) AddVirtioFS(ctx context.Context, config VirtioFSConfig) error {
	log := slog.With(slog.String("component", "libkrun"), slog.Uint64("ctx_id", uint64(c.id)))
	log.DebugContext(ctx, "adding virtio-fs device",
		slog.String("tag", config.Tag),
		slog.String("path", config.Path))

	cTag := C.CString(config.Tag)
	defer C.free(unsafe.Pointer(cTag))
	cPath := C.CString(config.Path)
	defer C.free(unsafe.Pointer(cPath))

	if config.ShmSize != nil {
		log.DebugContext(ctx, "using custom DAX window size", slog.Uint64("shm_size", *config.ShmSize))
		result := C.krun_add_virtiofs2(C.uint32_t(c.id), cTag, cPath, C.uint64_t(*config.ShmSize))
		if result < 0 {
			return errors.Errorf("adding virtio-fs device with DAX: %d", result)
		}
	} else {
		result := C.krun_add_virtiofs(C.uint32_t(c.id), cTag, cPath)
		if result < 0 {
			return errors.Errorf("adding virtio-fs device: %d", result)
		}
	}
	return nil
}

// SetNetwork configures networking
func (c *Context) SetNetwork(ctx context.Context, config NetworkConfig) error {
	log := slog.With(slog.String("component", "libkrun"), slog.Uint64("ctx_id", uint64(c.id)))

	// Set networking backend
	if config.PasstFD != nil {
		log.DebugContext(ctx, "setting passt network backend", slog.Int("fd", *config.PasstFD))
		result := C.krun_set_passt_fd(C.uint32_t(c.id), C.int(*config.PasstFD))
		if result < 0 {
			return errors.Errorf("setting passt network backend: %d", result)
		}
	} else if config.GvproxyPath != nil {
		log.DebugContext(ctx, "setting gvproxy network backend", slog.String("path", *config.GvproxyPath))
		cPath := C.CString(*config.GvproxyPath)
		defer C.free(unsafe.Pointer(cPath))
		result := C.krun_set_gvproxy_path(C.uint32_t(c.id), cPath)
		if result < 0 {
			return errors.Errorf("setting gvproxy network backend: %d", result)
		}
	}

	// Set MAC address
	if config.MAC != nil {
		log.DebugContext(ctx, "setting network MAC address")
		result := C.krun_set_net_mac(C.uint32_t(c.id), (*C.uint8_t)(unsafe.Pointer(&config.MAC[0])))
		if result < 0 {
			return errors.Errorf("setting network MAC address: %d", result)
		}
	}

	// Set port mapping - be very careful with C memory management
	log.DebugContext(ctx, "setting port mapping", slog.Any("port_map", config.PortMap))

	// Always create a valid C array, even for empty port maps
	cPortMap := make([]*C.char, len(config.PortMap)+1)
	for i, port := range config.PortMap {
		cPortMap[i] = C.CString(port)
		defer C.free(unsafe.Pointer(cPortMap[i]))
	}
	cPortMap[len(config.PortMap)] = nil // NULL terminate

	// Pass a valid pointer to the array, even if it only contains NULL
	var portMapPtr **C.char
	if len(cPortMap) > 0 {
		portMapPtr = &cPortMap[0]
	} else {
		// For empty arrays, create a single NULL pointer
		nullPtr := (*C.char)(nil)
		portMapPtr = &nullPtr
	}

	result := C.krun_set_port_map(C.uint32_t(c.id), portMapPtr)
	if result < 0 {
		return errors.Errorf("setting port map: %d", result)
	}

	return nil
}

// SetGPU configures GPU options
func (c *Context) SetGPU(ctx context.Context, config GPUConfig) error {
	log := slog.With(slog.String("component", "libkrun"), slog.Uint64("ctx_id", uint64(c.id)))
	log.DebugContext(ctx, "setting GPU options", slog.Any("virgl_flags", config.VirglFlags))

	if config.ShmSize != nil {
		log.DebugContext(ctx, "using custom vRAM size", slog.Uint64("shm_size", *config.ShmSize))
		result := C.krun_set_gpu_options2(C.uint32_t(c.id), C.uint32_t(config.VirglFlags), C.uint64_t(*config.ShmSize))
		if result < 0 {
			return errors.Errorf("setting GPU options with vRAM: %d", result)
		}
	} else {
		result := C.krun_set_gpu_options(C.uint32_t(c.id), C.uint32_t(config.VirglFlags))
		if result < 0 {
			return errors.Errorf("setting GPU options: %d", result)
		}
	}
	return nil
}

// SetProcess configures the process to run
func (c *Context) SetProcess(ctx context.Context, config ProcessConfig) error {
	log := slog.With(slog.String("component", "libkrun"), slog.Uint64("ctx_id", uint64(c.id)))
	log.DebugContext(ctx, "setting process config",
		slog.String("exec_path", config.ExecPath),
		slog.Any("args", config.Args),
		slog.Int("env_count", len(config.Env)))

	// Set working directory if specified
	if config.WorkDir != nil {
		log.DebugContext(ctx, "setting working directory", slog.String("workdir", *config.WorkDir))
		cWorkdir := C.CString(*config.WorkDir)
		defer C.free(unsafe.Pointer(cWorkdir))
		result := C.krun_set_workdir(C.uint32_t(c.id), cWorkdir)
		if result < 0 {
			return errors.Errorf("setting working directory: %d", result)
		}
	}

	// Set environment if specified separately
	if len(config.Env) > 0 {
		cEnvp := make([]*C.char, len(config.Env)+1)
		for i, env := range config.Env {
			cEnvp[i] = C.CString(env)
			defer C.free(unsafe.Pointer(cEnvp[i]))
		}
		cEnvp[len(config.Env)] = nil

		result := C.krun_set_env(C.uint32_t(c.id), &cEnvp[0])
		if result < 0 {
			return errors.Errorf("setting environment variables: %d", result)
		}
	}

	// Set executable and arguments
	cExecPath := C.CString(config.ExecPath)
	defer C.free(unsafe.Pointer(cExecPath))

	cArgv := make([]*C.char, len(config.Args)+1)
	for i, arg := range config.Args {
		cArgv[i] = C.CString(arg)
		defer C.free(unsafe.Pointer(cArgv[i]))
	}
	cArgv[len(config.Args)] = nil

	var cEnvp **C.char
	if len(config.Env) > 0 {
		cEnvpSlice := make([]*C.char, len(config.Env)+1)
		for i, env := range config.Env {
			cEnvpSlice[i] = C.CString(env)
			defer C.free(unsafe.Pointer(cEnvpSlice[i]))
		}
		cEnvpSlice[len(config.Env)] = nil
		cEnvp = &cEnvpSlice[0]
	}

	result := C.krun_set_exec(C.uint32_t(c.id), cExecPath, &cArgv[0], cEnvp)
	if result < 0 {
		return errors.Errorf("setting VM executable: %d", result)
	}
	return nil
}

// SetKernel configures kernel settings
func (c *Context) SetKernel(ctx context.Context, config KernelConfig) error {
	log := slog.With(slog.String("component", "libkrun"), slog.Uint64("ctx_id", uint64(c.id)))
	log.DebugContext(ctx, "setting kernel config",
		slog.String("path", config.Path),
		slog.Any("format", config.Format),
		slog.String("cmdline", config.Cmdline))

	cKernelPath := C.CString(config.Path)
	defer C.free(unsafe.Pointer(cKernelPath))

	var cInitramfs *C.char
	if config.Initramfs != nil {
		cInitramfs = C.CString(*config.Initramfs)
		defer C.free(unsafe.Pointer(cInitramfs))
	}

	cCmdline := C.CString(config.Cmdline)
	defer C.free(unsafe.Pointer(cCmdline))

	result := C.krun_set_kernel(C.uint32_t(c.id), cKernelPath, C.uint32_t(config.Format), cInitramfs, cCmdline)
	if result < 0 {
		return errors.Errorf("setting kernel: %d", result)
	}
	return nil
}

// AddVsockPorts adds vsock port mappings
func (c *Context) AddVsockPorts(ctx context.Context, ports []VsockPort) error {
	log := slog.With(slog.String("component", "libkrun"), slog.Uint64("ctx_id", uint64(c.id)))

	for _, port := range ports {
		log.DebugContext(ctx, "adding vsock port",
			slog.Uint64("port", uint64(port.Port)),
			slog.String("filepath", port.FilePath))

		cFilepath := C.CString(port.FilePath)
		defer C.free(unsafe.Pointer(cFilepath))

		if port.Listen != nil {
			result := C.krun_add_vsock_port2(C.uint32_t(c.id), C.uint32_t(port.Port), cFilepath, C.bool(*port.Listen))
			if result < 0 {
				return errors.Errorf("adding vsock port with listen config: %d", result)
			}
		} else {
			result := C.krun_add_vsock_port(C.uint32_t(c.id), C.uint32_t(port.Port), cFilepath)
			if result < 0 {
				return errors.Errorf("adding vsock port: %d", result)
			}
		}
	}
	return nil
}

// SetSecurity configures security settings
func (c *Context) SetSecurity(ctx context.Context, config SecurityConfig) error {
	log := slog.With(slog.String("component", "libkrun"), slog.Uint64("ctx_id", uint64(c.id)))

	// Set UID/GID
	if config.UID != nil {
		log.DebugContext(ctx, "setting user ID", slog.Uint64("uid", uint64(*config.UID)))
		result := C.krun_setuid(C.uint32_t(c.id), C.uid_t(*config.UID))
		if result < 0 {
			return errors.Errorf("setting user ID: %d", result)
		}
	}

	if config.GID != nil {
		log.DebugContext(ctx, "setting group ID", slog.Uint64("gid", uint64(*config.GID)))
		result := C.krun_setgid(C.uint32_t(c.id), C.gid_t(*config.GID))
		if result < 0 {
			return errors.Errorf("setting group ID: %d", result)
		}
	}

	// Set resource limits
	if len(config.Rlimits) > 0 {
		log.DebugContext(ctx, "setting resource limits", slog.Any("rlimits", config.Rlimits))
		cRlimits := make([]*C.char, len(config.Rlimits)+1)
		for i, rlimit := range config.Rlimits {
			cRlimits[i] = C.CString(rlimit)
			defer C.free(unsafe.Pointer(cRlimits[i]))
		}
		cRlimits[len(config.Rlimits)] = nil

		result := C.krun_set_rlimits(C.uint32_t(c.id), &cRlimits[0])
		if result < 0 {
			return errors.Errorf("setting resource limits: %d", result)
		}
	}

	// Set SMBIOS OEM strings
	if len(config.SMBIOSOEMStrings) > 0 {
		log.DebugContext(ctx, "setting SMBIOS OEM strings", slog.Any("oem_strings", config.SMBIOSOEMStrings))
		cOemStrings := make([]*C.char, len(config.SMBIOSOEMStrings)+1)
		for i, str := range config.SMBIOSOEMStrings {
			cOemStrings[i] = C.CString(str)
			defer C.free(unsafe.Pointer(cOemStrings[i]))
		}
		cOemStrings[len(config.SMBIOSOEMStrings)] = nil

		result := C.krun_set_smbios_oem_strings(C.uint32_t(c.id), &cOemStrings[0])
		if result < 0 {
			return errors.Errorf("setting SMBIOS OEM strings: %d", result)
		}
	}

	return nil
}

// SetAdvanced configures advanced settings
func (c *Context) SetAdvanced(ctx context.Context, config AdvancedConfig) error {
	log := slog.With(slog.String("component", "libkrun"), slog.Uint64("ctx_id", uint64(c.id)))

	// Set nested virtualization
	if config.NestedVirt != nil {
		log.DebugContext(ctx, "setting nested virtualization", slog.Bool("enabled", *config.NestedVirt))
		result := C.krun_set_nested_virt(C.uint32_t(c.id), C.bool(*config.NestedVirt))
		if result < 0 {
			return errors.Errorf("setting nested virtualization: %d", result)
		}
	}

	// Set sound device
	if config.SoundDevice != nil {
		log.DebugContext(ctx, "setting sound device", slog.Bool("enabled", *config.SoundDevice))
		result := C.krun_set_snd_device(C.uint32_t(c.id), C.bool(*config.SoundDevice))
		if result < 0 {
			return errors.Errorf("setting sound device: %d", result)
		}
	}

	// Set console output
	if config.ConsoleOutput != nil {
		log.DebugContext(ctx, "setting console output", slog.String("filepath", *config.ConsoleOutput))
		cFilepath := C.CString(*config.ConsoleOutput)
		defer C.free(unsafe.Pointer(cFilepath))
		result := C.krun_set_console_output(C.uint32_t(c.id), cFilepath)
		if result < 0 {
			return errors.Errorf("setting console output: %d", result)
		}
	}

	return nil
}

// StartEnter starts and enters the microVM with the configured parameters
func (c *Context) StartEnter(ctx context.Context) error {
	log := slog.With(slog.String("component", "libkrun"), slog.Uint64("ctx_id", uint64(c.id)))
	log.DebugContext(ctx, "starting VM")

	result := C.krun_start_enter(C.uint32_t(c.id))
	if result < 0 {
		return errors.Errorf("starting VM: %d", result)
	}
	return nil
}
