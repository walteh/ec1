//go:build darwin

package ignition

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	types_exp "github.com/coreos/ignition/v2/config/v3_6_experimental/types"
	"github.com/walteh/ec1/pkg/hypervisors/vf/config"
	"github.com/walteh/ec1/pkg/machines/boot"
	"gitlab.com/tozd/go/errors"
)

const (
	appleVFIgnitionPort = 1024
)

var _ boot.BootConfigProvider = &DarwinIgnitionBootConfigProvider{}

type IgnitionBootConfigProvider = DarwinIgnitionBootConfigProvider

func NewIgnitionBootConfigProvider(cfg *types_exp.Config, tmpDir string) *IgnitionBootConfigProvider {
	return &DarwinIgnitionBootConfigProvider{cfg: cfg, tmpDir: tmpDir}
}

type DarwinIgnitionBootConfigProvider struct {
	cfg    *types_exp.Config
	tmpDir string
}

func (me *DarwinIgnitionBootConfigProvider) ignitionSocketPath() string {
	return filepath.Join(me.tmpDir, "ignition.sock")
}

func (me *DarwinIgnitionBootConfigProvider) Run(ctx context.Context) error {
	// we could prob just straight encode this, might as well do the validation and marshall before
	// 	waiting on the server though to avoid any issues with error propagation

	marshaledIgnition, err := json.Marshal(me.cfg)
	if err != nil {
		return errors.Errorf("marshaling ignition: %w", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		slog.DebugContext(ctx, "ignition request (TODO: see if we can use this to expire the server)", "method", req.Method, "url", req.URL, "headers", req.Header)

		_, err := io.Copy(w, bytes.NewReader(marshaledIgnition))
		if err != nil {
			slog.ErrorContext(ctx, "failed to serve ignition file", "error", err)
		}
	})

	listener, err := net.Listen("unix", me.ignitionSocketPath())
	if err != nil {
		return errors.Errorf("listening on ignition socket: %w", err)
	}
	defer func() {
		if err := listener.Close(); err != nil {
			slog.ErrorContext(ctx, "failed to close ignition socket (defer)", "error", err)
		}
		if err := os.Remove(me.ignitionSocketPath()); err != nil {
			slog.ErrorContext(ctx, "failed to remove ignition socket (defer)", "error", err)
		}
	}()

	srv := &http.Server{
		Handler:           mux,
		Addr:              me.ignitionSocketPath(),
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		<-ctx.Done()
		if err := srv.Shutdown(ctx); err != nil {
			slog.WarnContext(ctx, "failed to shutdown ignition server", "error", err)
		}
	}()

	slog.DebugContext(ctx, "ignition socket", "socket", me.ignitionSocketPath)

	return srv.Serve(listener)
}

func (me *DarwinIgnitionBootConfigProvider) Device(ctx context.Context) boot.BootDevice {
	return boot.BootDevice(config.VirtioVsock{
		Port:      appleVFIgnitionPort,
		SocketURL: me.ignitionSocketPath(),
		Listen:    true,
	})
}
