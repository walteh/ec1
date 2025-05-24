//go:build libkrun_efi

package libkrun

/*
#cgo pkg-config: libkrun-efi
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

// GetShutdownEventFD returns the eventfd for orderly shutdown
// Only available in libkrun-efi variant
func (c *Context) GetShutdownEventFD(ctx context.Context) (int, error) {
	log := slog.With(slog.String("component", "libkrun-efi"), slog.Uint64("ctx_id", uint64(c.id)))
	log.DebugContext(ctx, "getting shutdown eventfd")

	result := C.krun_get_shutdown_eventfd(c.id)
	if result < 0 {
		return 0, errors.Errorf("getting shutdown eventfd: %d", result)
	}
	return int(result), nil
}

// SetRoot sets the path to be used as root for the microVM
// Available in libkrun-efi variant
func (c *Context) SetRoot(ctx context.Context, rootPath string) error {
	log := slog.With(slog.String("component", "libkrun-efi"), slog.Uint64("ctx_id", uint64(c.id)))
	log.DebugContext(ctx, "setting VM root path", slog.String("root_path", rootPath))

	cRootPath := C.CString(rootPath)
	defer C.free(unsafe.Pointer(cRootPath))

	result := C.krun_set_root(c.id, cRootPath)
	if result < 0 {
		return errors.Errorf("setting VM root path: %d", result)
	}
	return nil
}

// SetMappedVolumes configures mapped volumes for the microVM
// Available in libkrun-efi variant
func (c *Context) SetMappedVolumes(ctx context.Context, mappedVolumes []string) error {
	log := slog.With(slog.String("component", "libkrun-efi"), slog.Uint64("ctx_id", uint64(c.id)))
	log.DebugContext(ctx, "setting mapped volumes", slog.Any("mapped_volumes", mappedVolumes))

	if len(mappedVolumes) == 0 {
		result := C.krun_set_mapped_volumes(c.id, nil)
		if result < 0 {
			return errors.Errorf("setting empty mapped volumes: %d", result)
		}
	} else {
		cMappedVolumes := make([]*C.char, len(mappedVolumes)+1)
		for i, volume := range mappedVolumes {
			cMappedVolumes[i] = C.CString(volume)
			defer C.free(unsafe.Pointer(cMappedVolumes[i]))
		}
		cMappedVolumes[len(mappedVolumes)] = nil

		result := C.krun_set_mapped_volumes(c.id, &cMappedVolumes[0])
		if result < 0 {
			return errors.Errorf("setting mapped volumes: %d", result)
		}
	}

	return nil
}

// SetSEVConfig is not available in libkrun-efi variant
func (c *Context) SetSEVConfig(ctx context.Context, config SEVConfig) error {
	log := slog.With(slog.String("component", "libkrun-efi"), slog.Uint64("ctx_id", uint64(c.id)))
	log.WarnContext(ctx, "SetSEVConfig is not available in libkrun-efi variant")
	return errors.Errorf("SetSEVConfig is only available in libkrun-sev variant")
} 