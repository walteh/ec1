package tapsock

import (
	"context"
	"io"
	"log/slog"
	"net"
	"reflect"

	"github.com/containers/gvisor-tap-vsock/pkg/tap"
	"github.com/containers/gvisor-tap-vsock/pkg/types"
	"github.com/containers/gvisor-tap-vsock/pkg/virtualnetwork"
	"github.com/walteh/run"
	"gitlab.com/tozd/go/errors"

	"github.com/walteh/ec1/pkg/hack"
)

var _ run.Runnable = (*VirtualNetworkRunner)(nil)

type VirtualNetworkRunner struct {
	name    string
	running bool
	swich   *tap.Switch
	netConn net.Conn

	toClose map[string]io.Closer
}

func (me *VirtualNetworkRunner) ApplyVirtualNetwork(vn *virtualnetwork.VirtualNetwork) error {

	val := hack.GetUnexportedField(reflect.ValueOf(vn).Elem().FieldByName("networkSwitch"))
	if val == nil {
		return errors.New("invalid virtual network")
	}

	if swtch, ok := val.(*tap.Switch); ok {
		me.swich = swtch
	} else {
		return errors.Errorf("invalid virtual network: expected *tap.Switch, got %T", val)
	}

	return nil
}

func (me *VirtualNetworkRunner) Run(ctx context.Context) error {

	if me.swich == nil {
		return errors.New("virtual network is not set")
	}

	me.running = true
	defer func() {
		me.running = false
	}()

	slog.InfoContext(ctx, "accepting vfkit connection")

	err := me.swich.Accept(ctx, me.netConn, types.VfkitProtocol)
	if err != nil {
		slog.ErrorContext(ctx, "error accepting vfkit connection", "error", err)
		return errors.Errorf("accepting vfkit connection: %w", err)
	}

	slog.InfoContext(ctx, "accepted vfkit connection")

	return nil
}

func (me *VirtualNetworkRunner) Close(ctx context.Context) error {
	<-ctx.Done()
	if me.netConn != nil {
		if err := me.netConn.Close(); err != nil {
			slog.WarnContext(ctx, "error closing dgramVirtioNet netConn", "error", err)
		}
	}
	for name, closer := range me.toClose {
		if err := closer.Close(); err != nil {
			slog.WarnContext(ctx, "error closing dgramVirtioNet closer", "name", name, "error", err)
		}
	}
	return nil
}

func (me *VirtualNetworkRunner) Alive() bool {
	return me.running
}

func (me *VirtualNetworkRunner) Fields() []slog.Attr {
	return []slog.Attr{
		slog.String("name", me.name),
	}
}

func (me *VirtualNetworkRunner) Name() string {
	return me.name
}
