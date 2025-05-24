//go:build darwin && libkrun && cgo && !vmnet_helper

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
)

// SetVMNetNetwork configures networking using vmnet (requires entitlement)
// This provides better performance than the default TSI backend when vmnet entitlement is available
func (c *Context) SetVMNetNetwork(ctx context.Context, config VMNetConfig) error {
	log := slog.With(slog.String("component", "libkrun"), slog.Uint64("ctx_id", uint64(c.id)))
	log.WarnContext(ctx, "SetVMNetNetwork is currently only supported with vmnet-helper, use build tag vmnet_helper to enable it")
	return ErrNotImplemented
}
