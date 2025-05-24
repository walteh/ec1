//go:build libkrun && !libkrun_sev

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

// SetRoot sets the path to be used as root for the microVM
// Not available in libkrun-SEV variant
func (c *Context) SetRoot(ctx context.Context, rootPath string) error {
	log := slog.With(slog.String("component", "libkrun"), slog.Uint64("ctx_id", uint64(c.id)))
	log.DebugContext(ctx, "setting VM root path", slog.String("root_path", rootPath))

	cRootPath := C.CString(rootPath)
	defer C.free(unsafe.Pointer(cRootPath))

	result := C.krun_set_root(C.uint32_t(c.id), cRootPath)
	if result < 0 {
		return errors.Errorf("setting VM root path: %d", result)
	}
	return nil
}

// SetMappedVolumes configures mapped volumes for the microVM
// DEPRECATED: NO LONGER SUPPORTED. Not available in libkrun-SEV variant
func (c *Context) SetMappedVolumes(ctx context.Context, mappedVolumes []string) error {
	log := slog.With(slog.String("component", "libkrun"), slog.Uint64("ctx_id", uint64(c.id)))
	log.DebugContext(ctx, "setting mapped volumes (deprecated)", slog.Any("mapped_volumes", mappedVolumes))

	if len(mappedVolumes) == 0 {
		result := C.krun_set_mapped_volumes(C.uint32_t(c.id), nil)
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

		result := C.krun_set_mapped_volumes(C.uint32_t(c.id), &cMappedVolumes[0])
		if result < 0 {
			return errors.Errorf("setting mapped volumes: %d", result)
		}
	}

	return nil
}
