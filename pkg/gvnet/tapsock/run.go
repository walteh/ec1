package tapsock

import (
	"context"
	"log/slog"
	"net"
	"reflect"

	"github.com/containers/gvisor-tap-vsock/pkg/tap"
	"github.com/containers/gvisor-tap-vsock/pkg/types"
	"github.com/containers/gvisor-tap-vsock/pkg/virtualnetwork"
	"github.com/walteh/run"
	"gitlab.com/tozd/go/errors"
	"gvisor.dev/gvisor/pkg/tcpip/stack"

	"github.com/walteh/ec1/pkg/hack"
)

var _ run.Runnable = (*VirtualNetworkRunner)(nil)

type VirtualNetworkRunner struct {
	name    string
	running bool
	swich   *tap.Switch
	netConn net.Conn

	// Direct references to Unix connections for better debugging
	// hostConnUnix *net.UnixConn
	// vmConnUnix   *net.UnixConn

	cleanup func() error
	// proxyErrChans []<-chan error
}

func IsolateNetworkSwitch(vn *virtualnetwork.VirtualNetwork) (*tap.Switch, error) {
	val := hack.GetUnexportedField(reflect.ValueOf(vn).Elem().FieldByName("networkSwitch"))
	if val == nil {
		return nil, errors.New("invalid virtual network, networkSwitch is nil")
	}

	if swtch, ok := val.(*tap.Switch); ok {
		return swtch, nil
	} else {
		return nil, errors.Errorf("invalid virtual network: expected *tap.Switch, got %T", val)
	}
}

func IsolateNetworkStack(vn *virtualnetwork.VirtualNetwork) (*stack.Stack, error) {
	val := hack.GetUnexportedField(reflect.ValueOf(vn).Elem().FieldByName("stack"))
	if val == nil {
		return nil, errors.New("invalid virtual network, stack is nil")
	}

	if stack, ok := val.(*stack.Stack); ok {
		return stack, nil
	} else {
		return nil, errors.Errorf("invalid virtual network: expected *stack.Stack, got %T", val)
	}
}

func (me *VirtualNetworkRunner) ApplyVirtualNetwork(vn *virtualnetwork.VirtualNetwork) error {

	swtch, err := IsolateNetworkSwitch(vn)
	if err != nil {
		return errors.Errorf("isolating network switch: %w", err)
	}

	me.swich = swtch

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

	// Create a copy of the netConn to use with the tap.Switch
	// This way we can keep our original connections alive
	slog.InfoContext(ctx, "preparing connection for tap.Switch",
		"netConn_type", reflect.TypeOf(me.netConn).String(),
		"local_addr", me.netConn.LocalAddr().String(),
		"remote_addr", me.netConn.RemoteAddr().String())

	// Send test packets to verify connectivity
	// go sendTestPackets(ctx, me.hostConnUnix, me.vmConnUnix)

	// Wrap the connection with debug logging
	// wrappedConn := &debugLogConn{
	// 	Conn: me.netConn,
	// 	name: "VirtualNetworkRunner.netConn",
	// 	ctx:  ctx,
	// }

	slog.InfoContext(ctx, "calling switch.Accept on connection",
		"protocol", types.VfkitProtocol,
		"switch_type", reflect.TypeOf(me.swich).String())

	err := me.swich.Accept(ctx, me.netConn, types.VfkitProtocol)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			slog.InfoContext(ctx, "context done, disconnecting from tap.Switch", "error", err)
			return err
		} else {
			return errors.Errorf("tap.Switch.Accept error: %w", err)
		}
	}

	return nil
}

func (me *VirtualNetworkRunner) Close(ctx context.Context) error {
	slog.InfoContext(ctx, "closing VirtualNetworkRunner", "name", me.name)

	// First close the netConn since it's a wrapper and doesn't actually close the underlying sockets
	if me.netConn != nil {
		go me.netConn.Close()
		// slog.InfoContext(ctx, "marking netConn as closed", "addr", me.netConn.LocalAddr())
		// if err := me.netConn.Close(); err != nil {
		// 	slog.WarnContext(ctx, "error closing dgramVirtioNet netConn", "error", err)
		// }
	}

	// Keep track of any errors during closing
	var closeErrors []error

	me.cleanup()

	me.running = false
	slog.InfoContext(ctx, "VirtualNetworkRunner successfully closed", "name", me.name)

	// If any errors occurred during closing, return the first one
	if len(closeErrors) > 0 {
		return errors.Join(closeErrors...)
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
