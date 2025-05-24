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
	"unsafe"

	"github.com/rs/zerolog"
	"gitlab.com/tozd/go/errors"
)

// Context represents a libkrun configuration context
type Context struct {
	id C.uint32_t
}

// SetLogLevel sets the log level for libkrun
func SetLogLevel(ctx context.Context, level uint32) error {
	log := zerolog.Ctx(ctx)
	log.Debug().Uint32("level", level).Msg("setting libkrun log level")

	result := C.krun_set_log_level(C.uint32_t(level))
	if result < 0 {
		return errors.Errorf("setting libkrun log level: %d", result)
	}
	return nil
}

// CreateContext creates a new libkrun configuration context
func CreateContext(ctx context.Context) (*Context, error) {
	log := zerolog.Ctx(ctx)
	log.Debug().Msg("creating libkrun context")

	result := C.krun_create_ctx()
	if result < 0 {
		return nil, errors.Errorf("creating libkrun context: %d", result)
	}

	kctx := &Context{id: C.uint32_t(result)}
	log.Debug().Uint32("ctx_id", uint32(kctx.id)).Msg("created libkrun context")
	return kctx, nil
}

// Free frees the libkrun configuration context
func (c *Context) Free(ctx context.Context) error {
	log := zerolog.Ctx(ctx)
	log.Debug().Uint32("ctx_id", uint32(c.id)).Msg("freeing libkrun context")

	result := C.krun_free_ctx(c.id)
	if result < 0 {
		return errors.Errorf("freeing libkrun context: %d", result)
	}
	return nil
}

// SetVMConfig sets the basic configuration parameters for the microVM
func (c *Context) SetVMConfig(ctx context.Context, numVCPUs uint8, ramMiB uint32) error {
	log := zerolog.Ctx(ctx)
	log.Debug().
		Uint32("ctx_id", uint32(c.id)).
		Uint8("num_vcpus", numVCPUs).
		Uint32("ram_mib", ramMiB).
		Msg("setting VM config")

	result := C.krun_set_vm_config(c.id, C.uint8_t(numVCPUs), C.uint32_t(ramMiB))
	if result < 0 {
		return errors.Errorf("setting VM config: %d", result)
	}
	return nil
}

// SetRoot sets the path to be used as root for the microVM
func (c *Context) SetRoot(ctx context.Context, rootPath string) error {
	log := zerolog.Ctx(ctx)
	log.Debug().
		Uint32("ctx_id", uint32(c.id)).
		Str("root_path", rootPath).
		Msg("setting VM root path")

	cRootPath := C.CString(rootPath)
	defer C.free(unsafe.Pointer(cRootPath))

	result := C.krun_set_root(c.id, cRootPath)
	if result < 0 {
		return errors.Errorf("setting VM root path: %d", result)
	}
	return nil
}

// SetExec sets the executable to be run inside the microVM
func (c *Context) SetExec(ctx context.Context, execPath string, argv []string, envp []string) error {
	log := zerolog.Ctx(ctx)
	log.Debug().
		Uint32("ctx_id", uint32(c.id)).
		Str("exec_path", execPath).
		Strs("argv", argv).
		Int("envp_count", len(envp)).
		Msg("setting VM executable")

	cExecPath := C.CString(execPath)
	defer C.free(unsafe.Pointer(cExecPath))

	// Convert argv slice to C array
	cArgv := make([]*C.char, len(argv)+1)
	for i, arg := range argv {
		cArgv[i] = C.CString(arg)
		defer C.free(unsafe.Pointer(cArgv[i]))
	}
	cArgv[len(argv)] = nil

	// Convert envp slice to C array
	var cEnvp **C.char
	if len(envp) > 0 {
		cEnvpSlice := make([]*C.char, len(envp)+1)
		for i, env := range envp {
			cEnvpSlice[i] = C.CString(env)
			defer C.free(unsafe.Pointer(cEnvpSlice[i]))
		}
		cEnvpSlice[len(envp)] = nil
		cEnvp = &cEnvpSlice[0]
	}

	result := C.krun_set_exec(c.id, cExecPath, &cArgv[0], cEnvp)
	if result < 0 {
		return errors.Errorf("setting VM executable: %d", result)
	}
	return nil
}

// StartEnter starts and enters the microVM with the configured parameters
func (c *Context) StartEnter(ctx context.Context) error {
	log := zerolog.Ctx(ctx)
	log.Debug().Uint32("ctx_id", uint32(c.id)).Msg("starting VM")

	result := C.krun_start_enter(c.id)
	if result < 0 {
		return errors.Errorf("starting VM: %d", result)
	}
	return nil
}
