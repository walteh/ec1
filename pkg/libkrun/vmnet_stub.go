//go:build !darwin || !libkrun || !cgo

package libkrun

import (
	"context"
	"log/slog"
)

// SetVMNetNetwork is not available without vmnet entitlement
func (c *Context) SetVMNetNetwork(ctx context.Context, config VMNetConfig) error {
	log := slog.With(slog.String("component", "libkrun"), slog.Uint64("ctx_id", uint64(c.id)))
	log.WarnContext(ctx, "vmnet networking not available (requires darwin)")
	return ErrVMNetNotAvailable
}
