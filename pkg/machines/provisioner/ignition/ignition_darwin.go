//go:build darwin

package ignition

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"time"

	types_exp "github.com/coreos/ignition/v2/config/v3_6_experimental/types"
	"github.com/walteh/ec1/pkg/hypervisors"
	"github.com/walteh/ec1/pkg/machines/virtio"
	"gitlab.com/tozd/go/errors"
)

const (
	APPLE_HF_STATIC_IGNITION_PORT = 1024
)

var _ hypervisors.BootProvisioner = &DarwinIgnitionBootConfigProvider{}

type IgnitionBootConfigProvider = DarwinIgnitionBootConfigProvider

type DarwinIgnitionBootConfigProvider struct {
	cfg *types_exp.Config
}

func (me *DarwinIgnitionBootConfigProvider) ignitionSocketPath() string {
	return "ignition.sock"
}

func (me *DarwinIgnitionBootConfigProvider) device() *virtio.VirtioVsock {
	return &virtio.VirtioVsock{
		Port:      APPLE_HF_STATIC_IGNITION_PORT,
		SocketURL: me.ignitionSocketPath(),
		Direction: virtio.VirtioVsockDirectionGuestConnectsAsClient,
	}
}

func (me *DarwinIgnitionBootConfigProvider) RunDuringBoot(ctx context.Context, vm hypervisors.VirtualMachine) error {
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

	listener, err := hypervisors.ListenVsock(ctx, vm, me.device())
	if err != nil {
		return errors.Errorf("listening on ignition socket: %w", err)
	}
	defer func() {
		if err := listener.Close(); err != nil {
			slog.ErrorContext(ctx, "failed to close ignition socket (defer)", "error", err)
		}
		// if err := os.Remove(socketPath); err != nil {
		// 	slog.ErrorContext(ctx, "failed to remove ignition socket (defer)", "error", err)
		// }
	}()

	srv := &http.Server{
		Handler: mux,
		// Addr:              socketPath,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		<-ctx.Done()
		if err := srv.Shutdown(ctx); err != nil {
			slog.WarnContext(ctx, "failed to shutdown ignition server", "error", err)
		}
	}()

	// slog.DebugContext(ctx, "ignition socket", "socket", socketPath)

	return srv.Serve(listener)
}

func (me *DarwinIgnitionBootConfigProvider) VirtioDevices(ctx context.Context) ([]virtio.VirtioDevice, error) {
	return []virtio.VirtioDevice{me.device()}, nil
}
