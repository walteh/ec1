package vmm

import (
	"context"
	"fmt"
	"log/slog"

	"golang.org/x/sync/errgroup"

	"gitlab.com/tozd/go/errors"

	"github.com/walteh/ec1/pkg/gvnet"
	"github.com/walteh/ec1/pkg/port"
	"github.com/walteh/ec1/pkg/virtio"
)

func PrepareVirtualNetwork(ctx context.Context, groupErrs *errgroup.Group) (*virtio.VirtioNet, uint16, error) {
	port, err := port.ReservePort(ctx)
	if err != nil {
		return nil, 0, errors.Errorf("reserving port: %w", err)
	}
	cfg := &gvnet.GvproxyConfig{
		VMHostPort:         fmt.Sprintf("tcp://127.0.0.1:%d", port),
		EnableDebug:        false,
		EnableStdioSocket:  false,
		EnableNoConnectAPI: true,
	}

	dev, waiter, err := gvnet.NewProxy(ctx, cfg)
	if err != nil {
		return nil, 0, errors.Errorf("creating gvproxy: %w", err)
	}

	groupErrs.Go(func() error {
		slog.InfoContext(ctx, "waiting on error from gvproxy")
		return waiter(ctx)
	})

	return dev, port, nil

}
