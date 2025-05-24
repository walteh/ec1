//go:build libkrun_sev

package libkrun

/*
#cgo pkg-config: libkrun-sev
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

// SEVConfig represents SEV-specific configuration
type SEVConfig struct {
	TEEConfigFile *string // Path to TEE configuration file
}

// SetSEVConfig configures SEV-specific settings
// Only available in libkrun-sev variant
func (c *Context) SetSEVConfig(ctx context.Context, config SEVConfig) error {
	log := slog.With(slog.String("component", "libkrun-sev"), slog.Uint64("ctx_id", uint64(c.id)))

	// Set TEE config file
	if config.TEEConfigFile != nil {
		log.DebugContext(ctx, "setting TEE config file", slog.String("filepath", *config.TEEConfigFile))
		cFilepath := C.CString(*config.TEEConfigFile)
		defer C.free(unsafe.Pointer(cFilepath))
		result := C.krun_set_tee_config_file(c.id, cFilepath)
		if result < 0 {
			return errors.Errorf("setting TEE config file: %d", result)
		}
	}

	return nil
}

// SetRoot is not available in libkrun-SEV variant
func (c *Context) SetRoot(ctx context.Context, rootPath string) error {
	log := slog.With(slog.String("component", "libkrun-sev"), slog.Uint64("ctx_id", uint64(c.id)))
	log.WarnContext(ctx, "SetRoot is not available in libkrun-SEV variant")
	return errors.Errorf("SetRoot is not available in libkrun-SEV variant")
}

// SetMappedVolumes is not available in libkrun-SEV variant
func (c *Context) SetMappedVolumes(ctx context.Context, mappedVolumes []string) error {
	log := slog.With(slog.String("component", "libkrun-sev"), slog.Uint64("ctx_id", uint64(c.id)))
	log.WarnContext(ctx, "SetMappedVolumes is not available in libkrun-SEV variant")
	return errors.Errorf("SetMappedVolumes is not available in libkrun-SEV variant")
}

// GetShutdownEventFD is not available in libkrun-SEV variant
func (c *Context) GetShutdownEventFD(ctx context.Context) (int, error) {
	log := slog.With(slog.String("component", "libkrun-sev"), slog.Uint64("ctx_id", uint64(c.id)))
	log.WarnContext(ctx, "GetShutdownEventFD is not available in libkrun-SEV variant")
	return 0, errors.Errorf("GetShutdownEventFD is only available in libkrun-efi variant")
}
