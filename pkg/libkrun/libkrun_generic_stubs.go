//go:build libkrun && !libkrun_sev && !libkrun_efi

package libkrun

import (
	"context"
	"log/slog"

	"gitlab.com/tozd/go/errors"
)

// SetSEVConfig is not available in generic libkrun variant
func (c *Context) SetSEVConfig(ctx context.Context, config SEVConfig) error {
	log := slog.With(slog.String("component", "libkrun"), slog.Uint64("ctx_id", uint64(c.id)))
	log.WarnContext(ctx, "SetSEVConfig is not available in generic libkrun variant")
	return errors.Errorf("SetSEVConfig is only available in libkrun-sev variant")
}

// GetShutdownEventFD is not available in generic libkrun variant
func (c *Context) GetShutdownEventFD(ctx context.Context) (int, error) {
	log := slog.With(slog.String("component", "libkrun"), slog.Uint64("ctx_id", uint64(c.id)))
	log.WarnContext(ctx, "GetShutdownEventFD is not available in generic libkrun variant")
	return 0, errors.Errorf("GetShutdownEventFD is only available in libkrun-efi variant")
}
