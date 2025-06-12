package vmm

import (
	"context"
	"fmt"

	"gitlab.com/tozd/go/errors"

	"github.com/walteh/ec1/pkg/gvnet"
	"github.com/walteh/ec1/pkg/port"
)

func PrepareVirtualNetwork(ctx context.Context) (gvnet.Proxy, uint16, error) {
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

	dev, err := gvnet.NewProxy(ctx, cfg)
	if err != nil {
		return nil, 0, errors.Errorf("creating gvproxy: %w", err)
	}

	return dev, port, nil

}
