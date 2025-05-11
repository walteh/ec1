package gvnet

// import (
// 	"context"
// 	"net"
// 	"net/http"

// 	"golang.org/x/sync/errgroup"

// 	"github.com/containers/gvisor-tap-vsock/pkg/services/forwarder"
// 	d "github.com/containers/gvisor-tap-vsock/pkg/tcpproxy"
// 	"github.com/walteh/run"
// 	"gitlab.com/tozd/go/errors"
// 	import "inet.af/tcpproxy"

// )

// var _ run.Runnable = (*PortForwarderProcess)(nil)

// type PortForwarderProcess struct {
// 	listener net.Listener
// 	port     string
// }

// func (me *PortForwarderProcess) Alive() bool {
// 	return me.listener != nil
// }

// func ForwardListenerToPort(ctx context.Context, listener net.Listener, port string, errgroup *errgroup.Group) error {

// 	prox := tcpproxy.To(port)

// 	proxy := tcpproxy.Proxy{
// 		ListenFunc: func(network, addr string) (net.Listener, error) {
// 			return listener, nil
// 		},
// 	}

// var proxy tcpproxy.Proxy
// for hostAddr := range virtualPortMap {
//     proxy.AddRouteListener(
//         /* listener: */ mux.Match(cmux.PrefixMatcher("SSH-")),
//         /* target: */ tcpproxy.To(hostAddr),
//     ) // route SSH traffic to reserved port  [oai_citation:9‡Go Packages](https://pkg.go.dev/inet.af/tcpproxy?utm_source=chatgpt.com)
// }

// 	pf := forwarder.NewPortsForwarder(stack)
// 	pf.Expose(types.TransportProtocolTCP, ":6443", "192.168.127.2:6443")
// 	http.Handle("/unix/services/forwarder", pf.Mux())

// 	proxy.AddRoute(port, prox)

// 	for {
// 		// Accept connection with timeout
// 		clientConn, err := listener.Accept()
// 		if err != nil {
// 			if errors.Is(err, net.ErrClosed) {
// 				return nil // Normal shutdown
// 			}
// 			return errors.Errorf("failed to accept: %w", err)
// 		}

// 		prox.HandleConn(clientConn)

// 		// // Handle each client in a separate goroutine
// 		// errgroup.Go(func() error {
// 		// 	defer clientConn.Close()
// 		// 	slog.InfoContext(ctx, "forwarding connection", "client", clientConn.RemoteAddr(), "backend", port)
// 		// 	// Connect to the backend FOR THIS CLIENT
// 		// 	backend, err := net.Dial("tcp", port)
// 		// 	if err != nil {
// 		// 		return errors.Errorf("failed to connect to backend: %w", err)
// 		// 	}

// 		// 	defer backend.Close()

// 		// 	slog.InfoContext(ctx, "connected to backend", "backend", backend.RemoteAddr())

// 		// 	// Use proper copying with context cancellation
// 		// 	done := make(chan struct{}, 2)
// 		// 	go func() {
// 		// 		io.Copy(backend, clientConn)
// 		// 		done <- struct{}{}
// 		// 	}()
// 		// 	go func() {
// 		// 		io.Copy(clientConn, backend)
// 		// 		done <- struct{}{}
// 		// 	}()

// 		// 	// Wait for either copy to finish or context to cancel
// 		// 	select {
// 		// 	case <-done:
// 		// 		return nil
// 		// 	case <-ctx.Done():
// 		// 		return ctx.Err()
// 		// 	}
// 		// })
// 	}
// }
