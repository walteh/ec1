package tapsock

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"reflect"
	"time"

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

	// Direct references to Unix connections for better debugging
	hostConnUnix *net.UnixConn
	vmConnUnix   *net.UnixConn

	toClose       map[string]io.Closer
	proxyErrChans []<-chan error
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

	// Set up error monitoring
	if len(me.proxyErrChans) > 0 {
		errCh := make(chan error, len(me.proxyErrChans))

		// Monitor proxy errors
		for i, ch := range me.proxyErrChans {
			go func(idx int, errChan <-chan error) {
				if err := <-errChan; err != nil {
					slog.ErrorContext(ctx, "proxy error", "index", idx, "error", err)
					errCh <- err
				} else {
					slog.InfoContext(ctx, "proxy exited normally", "index", idx)
				}
			}(i, ch)
		}

		// Check if any errors occurred while monitoring
		go func() {
			select {
			case err := <-errCh:
				if err != nil {
					slog.ErrorContext(ctx, "proxy failed", "error", err)
				}
			case <-ctx.Done():
				// Context cancelled, normal exit
			}
		}()
	}

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

	// Use goroutine to handle the tap.Switch.Accept, which will process network packets
	acceptErr := make(chan error, 1)
	go func() {
		err := me.swich.Accept(ctx, me.netConn, types.VfkitProtocol)
		acceptErr <- err
	}()

	// Keep the process running, monitoring for context cancellation or errors
	select {
	case err := <-acceptErr:
		if err != nil {
			slog.ErrorContext(ctx, "error in tap.Switch.Accept", "error", err)
			return errors.Errorf("tap.Switch.Accept error: %w", err)
		}
		slog.InfoContext(ctx, "tap.Switch.Accept completed successfully")
		return nil
	case <-ctx.Done():
		slog.InfoContext(ctx, "context cancelled, shutting down VirtualNetworkRunner", "error", ctx.Err())
		return ctx.Err()
	}
}

// testNetConnDirectly sends a simple test message through the connection
func testNetConnDirectly(ctx context.Context, conn net.Conn) {
	testData := []byte("EC1_DIRECT_NETCONN_TEST")
	slog.InfoContext(ctx, "testing direct netConn writing before Accept")

	n, err := conn.Write(testData)
	if err != nil {
		slog.ErrorContext(ctx, "failed to write test data to netConn",
			"error", err,
			"error_type", fmt.Sprintf("%T", err))
	} else {
		slog.InfoContext(ctx, "wrote test data to netConn", "bytes", n)
	}
}

// debugLogConn is a net.Conn wrapper that logs all reads and writes
type debugLogConn struct {
	net.Conn
	name string
	ctx  context.Context
}

func (d *debugLogConn) Read(b []byte) (n int, err error) {
	n, err = d.Conn.Read(b)
	if err != nil && err != io.EOF {
		slog.ErrorContext(d.ctx, "debugLogConn read error", "name", d.name, "error", err)
	} else if n > 0 {
		slog.InfoContext(d.ctx, "debugLogConn read data", "name", d.name, "bytes", n, "data", b[:n])
	}
	return
}

func (d *debugLogConn) Write(b []byte) (n int, err error) {
	slog.InfoContext(d.ctx, "debugLogConn write attempt", "name", d.name, "bytes", len(b))
	n, err = d.Conn.Write(b)
	if err != nil {
		slog.ErrorContext(d.ctx, "debugLogConn write error", "name", d.name, "error", err)
	} else {
		slog.InfoContext(d.ctx, "debugLogConn wrote data", "name", d.name, "bytes", n)
	}
	return
}

func (me *VirtualNetworkRunner) Close(ctx context.Context) error {
	slog.InfoContext(ctx, "closing VirtualNetworkRunner", "name", me.name)

	// Don't wait for context to be done - close resources immediately
	// This was previously blocking on <-ctx.Done()

	// First check if already closing to prevent duplicate close
	if !me.running {
		slog.InfoContext(ctx, "VirtualNetworkRunner already closed", "name", me.name)
		return nil
	}

	// Wait a moment to ensure all operations complete
	time.Sleep(100 * time.Millisecond)

	// First close the netConn since it's a wrapper and doesn't actually close the underlying sockets
	if me.netConn != nil {
		slog.InfoContext(ctx, "marking netConn as closed", "addr", me.netConn.LocalAddr())
		if err := me.netConn.Close(); err != nil {
			slog.WarnContext(ctx, "error closing dgramVirtioNet netConn", "error", err)
		}
	}

	// Now close all the actual resources in reverse order
	// The order matters to ensure sockets are released in the right order
	closeOrder := []string{
		"proxyCancel",  // First cancel the proxies
		"vmSocketCopy", // Then close the socket copy
		"vmConn",       // Then close the connections
		"hostConn",
		"vmConnUnix", // Finally close the Unix connections
		"hostConnUnix",
	}

	// Keep track of any errors during closing
	var closeErrors []error

	for _, name := range closeOrder {
		if closer, ok := me.toClose[name]; ok {
			slog.InfoContext(ctx, "closing component", "name", name)
			if err := closer.Close(); err != nil {
				slog.WarnContext(ctx, "error closing component", "name", name, "error", err)
				closeErrors = append(closeErrors, err)
			} else {
				slog.InfoContext(ctx, "successfully closed component", "name", name)
			}
		}
	}

	me.running = false
	slog.InfoContext(ctx, "VirtualNetworkRunner successfully closed", "name", me.name)

	// If any errors occurred during closing, return the first one
	if len(closeErrors) > 0 {
		return closeErrors[0]
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
