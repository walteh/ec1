//go:build !libkrun

package libkrun

import (
	"context"

	"github.com/rs/zerolog"
	"gitlab.com/tozd/go/errors"
)

// Context represents a libkrun configuration context (stub version)
type Context struct {
	id uint32
}

var (
	ErrLibkrunNotAvailable        = errors.New("libkrun not available (stub implementation)")
	nextContextID          uint32 = 1
)

// SetLogLevel sets the log level for libkrun (stub version)
func SetLogLevel(ctx context.Context, level uint32) error {
	log := zerolog.Ctx(ctx)
	log.Debug().Uint32("level", level).Msg("setting libkrun log level (stub)")
	return ErrLibkrunNotAvailable
}

// CreateContext creates a new libkrun configuration context (stub version)
func CreateContext(ctx context.Context) (*Context, error) {
	log := zerolog.Ctx(ctx)
	log.Debug().Msg("creating libkrun context (stub)")
	return nil, ErrLibkrunNotAvailable
}

// Free frees the libkrun configuration context (stub version)
func (c *Context) Free(ctx context.Context) error {
	log := zerolog.Ctx(ctx)
	log.Debug().Uint32("ctx_id", c.id).Msg("freeing libkrun context (stub)")
	return ErrLibkrunNotAvailable
}

// SetVMConfig sets the basic configuration parameters for the microVM (stub version)
func (c *Context) SetVMConfig(ctx context.Context, numVCPUs uint8, ramMiB uint32) error {
	log := zerolog.Ctx(ctx)
	log.Debug().
		Uint32("ctx_id", c.id).
		Uint8("num_vcpus", numVCPUs).
		Uint32("ram_mib", ramMiB).
		Msg("setting VM config (stub)")
	return ErrLibkrunNotAvailable
}

// SetRoot sets the path to be used as root for the microVM (stub version)
func (c *Context) SetRoot(ctx context.Context, rootPath string) error {
	log := zerolog.Ctx(ctx)
	log.Debug().
		Uint32("ctx_id", c.id).
		Str("root_path", rootPath).
		Msg("setting VM root path (stub)")
	return ErrLibkrunNotAvailable
}

// SetExec sets the executable to be run inside the microVM (stub version)
func (c *Context) SetExec(ctx context.Context, execPath string, argv []string, envp []string) error {
	log := zerolog.Ctx(ctx)
	log.Debug().
		Uint32("ctx_id", c.id).
		Str("exec_path", execPath).
		Strs("argv", argv).
		Int("envp_count", len(envp)).
		Msg("setting VM executable (stub)")
	return ErrLibkrunNotAvailable
}

// StartEnter starts and enters the microVM with the configured parameters (stub version)
func (c *Context) StartEnter(ctx context.Context) error {
	log := zerolog.Ctx(ctx)
	log.Debug().Uint32("ctx_id", c.id).Msg("starting VM (stub)")
	return ErrLibkrunNotAvailable
}
