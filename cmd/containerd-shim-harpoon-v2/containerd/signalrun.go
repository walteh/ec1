package containerd

import (
	"context"
	"log/slog"
	"syscall"
	"time"

	taskt "github.com/containerd/containerd/api/types/task"
	"gitlab.com/tozd/go/errors"

	harpoonv1 "github.com/walteh/ec1/gen/proto/golang/harpoon/v1"
)

type signalRunner struct {
	error        error
	exitCode     int32
	done         chan struct{}
	signalClient harpoonv1.TTRPCGuestService_RunSpecSignalClient
	status       taskt.Status
	exitedAt     time.Time
	startedAt    time.Time
}

func (rs *signalRunner) Wait() (int32, error) {
	<-rs.done
	return rs.exitCode, rs.error
}

func (rs *signalRunner) SendSignal(signal syscall.Signal) error {
	if rs.exitCode != 0 {
		return errors.Errorf("process already exited with code %d", rs.exitCode)
	}

	req, err := harpoonv1.NewRunSpecSignalRequestE(func(b *harpoonv1.RunSpecSignalRequest_builder) {
		b.Signal = ptr(int32(signal))
	})
	if err != nil {
		return errors.Errorf("creating run signal request: %w", err)
	}
	if err := rs.signalClient.Send(req); err != nil {
		return errors.Errorf("sending run signal request: %w", err)
	}
	return nil
}

func (rs *signalRunner) Serve(ctx context.Context) (int32, error) {
	defer func() {
		close(rs.done)
		rs.exitedAt = time.Now()
		rs.status = taskt.Status_STOPPED
	}()

	req, err := harpoonv1.NewRunSpecSignalRequestE(func(b *harpoonv1.RunSpecSignalRequest_builder) {
	})
	if err != nil {
		return 0, errors.Errorf("creating run signal request: %w", err)
	}
	rs.startedAt = time.Now()

	if err := rs.signalClient.Send(req); err != nil {
		return 0, errors.Errorf("sending run signal request: %w", err)
	}

	slog.InfoContext(ctx, "sent run signal request, waiting for response")

	msg, err := rs.signalClient.Recv()

	if err != nil {
		rs.error = err
		rs.exitCode = 18
	} else {
		rs.exitCode = msg.GetExitCode()
	}

	slog.InfoContext(ctx, "received run signal response", "exitCode", msg.GetExitCode())

	return rs.exitCode, rs.error
}
func ptr[T any](v T) *T {
	return &v
}
