package vmm

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/sync/errgroup"
	"golang.org/x/sys/unix"

	"github.com/containers/gvisor-tap-vsock/pkg/tcpproxy"
	slogctx "github.com/veqryn/slog-context"
	"gitlab.com/tozd/go/errors"

	"github.com/walteh/ec1/pkg/ec1init"
	"github.com/walteh/ec1/pkg/host"
	"github.com/walteh/ec1/pkg/virtio"
)

func ForwardStdio(ctx context.Context, vm VirtualMachine, stdin io.Reader, stdout io.Writer, stderr io.Writer) (err error) {

	errgroup := errgroup.Group{}

	if stdin != nil {

		errgroup.Go(func() error {
			ctx := slogctx.Append(ctx, slog.String("pipe", "stdin"))
			conn, err := connectToVsockWithRetry(ctx, vm, ec1init.VsockStdinPort)
			if err != nil {
				return errors.Errorf("error connecting to vsock stdin: %w", err)
			}
			defer conn.Close()
			slog.InfoContext(ctx, "connected to vsock", "port", ec1init.VsockStdinPort)
			_, err = io.Copy(conn, stdin)
			if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, net.ErrClosed) {
				return errors.Errorf("error copying stdin to vsock: %w", err)
			}
			return nil
		})

	}

	if stdout != nil {

		errgroup.Go(func() error {
			ctx := slogctx.Append(ctx, slog.String("pipe", "stdout"))
			conn, err := connectToVsockWithRetry(ctx, vm, ec1init.VsockStdoutPort)
			if err != nil {
				return errors.Errorf("error connecting to vsock stdout: %w", err)
			}
			defer conn.Close()
			slog.InfoContext(ctx, "connected to vsock", "port", ec1init.VsockStdoutPort)
			_, err = io.Copy(stdout, conn)
			if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, net.ErrClosed) {
				return errors.Errorf("error copying stdout to vsock: %w", err)
			}
			return nil
		})
	}

	if stderr != nil {

		errgroup.Go(func() error {
			ctx := slogctx.Append(ctx, slog.String("pipe", "stderr"))
			conn, err := connectToVsockWithRetry(ctx, vm, ec1init.VsockStderrPort)
			if err != nil {
				return errors.Errorf("error connecting to vsock stderr	: %w", err)
			}
			defer conn.Close()
			slog.InfoContext(ctx, "connected to vsock", "port", ec1init.VsockStderrPort)
			_, err = io.Copy(stderr, conn)
			if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, net.ErrClosed) {
				return errors.Errorf("error copying stderr to vsock: %w", err)
			}
			return nil
		})
	}

	defer func() {
		if r := recover(); r != nil {
			slog.ErrorContext(ctx, "panic in ForwardStdio", "error", r)
			err = errors.Errorf("panic in ForwardStdio: %v", r)
		}
	}()
	err = errgroup.Wait()
	if err != nil {
		slog.ErrorContext(ctx, "error forwarding stdio", "error", err)
	}

	return
}

func VSockProxyUnixAddr(ctx context.Context, vm VirtualMachine, proxiedDevice *virtio.VirtioVsock) (*net.UnixAddr, error) {
	empathicalCacheDir, err := host.EmphiricalVMCacheDir(ctx, vm.ID())
	if err != nil {
		return nil, err
	}

	vsockPath := filepath.Join(empathicalCacheDir, proxiedDevice.SocketURL)

	return &net.UnixAddr{Net: "unix", Name: vsockPath}, nil
}

func ListenVsock(ctx context.Context, vm VirtualMachine, proxiedDevice *virtio.VirtioVsock) (net.Listener, error) {
	if proxiedDevice.SocketURL == "" {
		return vm.VSockListen(ctx, proxiedDevice.Port)
	}

	addr, err := VSockProxyUnixAddr(ctx, vm, proxiedDevice)
	if err != nil {
		return nil, errors.Errorf("getting vsock proxy unix address: %w", err)
	}

	return net.ListenUnix("unix", addr)
}

func ConnectVsock(ctx context.Context, vm VirtualMachine, proxiedDevice *virtio.VirtioVsock) (net.Conn, error) {
	if proxiedDevice.SocketURL == "" {
		return vm.VSockConnect(ctx, proxiedDevice.Port)
	}

	addr, err := VSockProxyUnixAddr(ctx, vm, proxiedDevice)
	if err != nil {
		return nil, errors.Errorf("getting vsock proxy unix address: %w", err)
	}
	return net.DialUnix("unix", nil, addr)
}

func ExposeVsock(ctx context.Context, vm VirtualMachine, port uint32, direction virtio.VirtioVsockDirection) (net.Conn, net.Listener, func(), error) {
	if direction == virtio.VirtioVsockDirectionGuestListensAsServer {
		return ExposeConnectVsockProxy(ctx, vm, port)
	}
	return ExposeListenVsockProxy(ctx, vm, port)
}

// connectVsock proxies connections from a host unix socket to a vsock port
// This allows the host to initiate connections to the guest over vsock
func ExposeConnectVsockProxy(ctx context.Context, vm VirtualMachine, port uint32) (net.Conn, net.Listener, func(), error) {
	var proxy tcpproxy.Proxy

	fd, err := unix.Socket(unix.AF_UNIX, unix.SOCK_STREAM, 0)
	if err != nil {
		return nil, nil, nil, errors.Errorf("creating unix socket: %w", err)
	}

	vsockFile := os.NewFile(uintptr(fd), "vsock.socket")

	listener, err := net.FileListener(vsockFile)
	if err != nil {
		return nil, nil, nil, errors.Errorf("creating vsock listener: %w", err)
	}

	// listen for connections on the host unix socket
	proxy.ListenFunc = func(_, laddr string) (net.Listener, error) {
		return listener, nil
	}

	connz, err := vm.VSockConnect(ctx, port)
	if err != nil {
		return nil, nil, nil, errors.Errorf("connecting to vsock: %w", err)
	}

	proxy.AddRoute(fmt.Sprintf("unix://:%s", vsockFile.Name()), &tcpproxy.DialProxy{
		Addr: fmt.Sprintf("vsock:%d", port),
		// when there's a connection to the unix socket listener, connect to the specified vsock port
		DialContext: func(_ context.Context, _, addr string) (conn net.Conn, e error) {
			return vm.VSockConnect(ctx, port)
		},
	})

	err = proxy.Start()
	if err != nil {
		return nil, nil, nil, errors.Errorf("starting proxy: %w", err)
	}

	return connz, listener, func() {
		proxy.Close()
		vsockFile.Close()
		listener.Close()
		connz.Close()
	}, nil
}

// listenVsock proxies connections from a vsock port to a host unix socket.
// This allows the guest to initiate connections to the host over vsock
func ExposeListenVsockProxy(ctx context.Context, vm VirtualMachine, port uint32) (net.Conn, net.Listener, func(), error) {
	var proxy tcpproxy.Proxy

	fd, err := unix.Socket(unix.AF_UNIX, unix.SOCK_STREAM, 0)
	if err != nil {
		return nil, nil, nil, errors.Errorf("creating unix socket: %w", err)
	}

	vsockFile := os.NewFile(uintptr(fd), "vsock.socket")

	connz, err := vm.VSockConnect(ctx, port)
	if err != nil {
		return nil, nil, nil, errors.Errorf("connecting to vsock: %w", err)
	}

	listener, err := vm.VSockListen(ctx, uint32(port))
	if err != nil {
		return nil, nil, nil, errors.Errorf("listening to vsock: %w", err)
	}

	// listen for connections on the vsock port
	proxy.ListenFunc = func(_, laddr string) (net.Listener, error) {
		return listener, nil
	}

	proxy.AddRoute(fmt.Sprintf("vsock://:%d", port), &tcpproxy.DialProxy{
		Addr: fmt.Sprintf("unix:%s", vsockFile.Name()),
		// when there's a connection to the vsock listener, connect to the provided unix socket
		DialContext: func(ctx context.Context, _, addr string) (conn net.Conn, e error) {
			return vm.VSockConnect(ctx, port)
		},
	})

	err = proxy.Start()
	if err != nil {
		return nil, nil, nil, errors.Errorf("starting proxy: %w", err)
	}

	return connz, listener, func() {
		proxy.Close()
		vsockFile.Close()
		listener.Close()
		connz.Close()
	}, nil
}

type VsockClientConnection struct {
	// client connections can be
	// - unix file socket
	// - internal unix socket
	// - tcp
	// - udp
	// dgram or stream
	port         uint32
	conn         net.Conn
	startTime    time.Time
	connType     VsockClientConnectionType
	transferType VsockTransferType
}

type VsockServerListener struct {
	// server listeners are always internal unix sockets, either
	// dgram or stream
	port         uint32
	listener     net.Listener
	startTime    time.Time
	transferType VsockTransferType
}

type VsockClientConnectionType string

const (
	VsockClientConnectionTypeUnixFileSocket VsockClientConnectionType = "unix_file_socket"
	VsockClientConnectionTypeUnixSocket     VsockClientConnectionType = "unix_socket"
	VsockClientConnectionTypeTCP            VsockClientConnectionType = "tcp"
	VsockClientConnectionTypeUDP            VsockClientConnectionType = "udp"
)

type VsockTransferType string

const (
	VsockTransferTypeDgram  VsockTransferType = "dgram"
	VsockTransferTypeStream VsockTransferType = "stream"
)

func NewVSockStreamConnection(ctx context.Context, vm VirtualMachine, guestVSockPort uint32) (hostSideConn net.Conn, closerFunc func(), err error) {
	fd, closerFunc, err := NewVSockStreamFileProxy(ctx, vm, guestVSockPort)
	if err != nil {
		return nil, nil, errors.Errorf("creating unix socket stream connection: %w", err)
	}

	hostSideConn, err = net.FileConn(fd)
	if err != nil {
		return nil, nil, errors.Errorf("creating host side connection: %w", err)
	}

	return hostSideConn, func() {
		fd.Close()
		closerFunc()
	}, nil
}

// NewUnixSocketStreamConnection sets up a Unix socket pair.
// Data written to the returned 'hostSideConn' by the host application
// will be forwarded to the guest's VSock port.
// Data sent by the guest on that VSock port will be readable from 'hostSideConn'.
func NewVSockStreamFileProxy(ctx context.Context, vm VirtualMachine, guestVSockPort uint32) (hostSideConn *os.File, closerFunc func(), err error) {
	slog.InfoContext(ctx, "setting up unix socket pair to bridge to guest VSock port", "guestVSockPort", guestVSockPort)

	fds, err := unix.Socketpair(unix.AF_UNIX, unix.SOCK_STREAM, 0) // SOCK_STREAM is important
	if err != nil {
		return nil, nil, errors.Errorf("creating socket pair: %w", err)
	}

	// Create net.Conn from the file descriptors
	// fds[0] will be for the host side (returned to caller)
	// fds[1] will be used to connect to the guest's VSock
	hostFile := os.NewFile(uintptr(fds[0]), fmt.Sprintf("host-to-vsock-port-%d.sock", guestVSockPort))
	vmFile := os.NewFile(uintptr(fds[1]), fmt.Sprintf("internal-link-to-vsock-port-%d.sock", guestVSockPort))

	// hostConn, err := net.FileConn(hostFile)
	// if err != nil {
	// 	_ = hostFile.Close()
	// 	_ = vmFile.Close()
	// 	return nil, nil, errors.Errorf("creating hostConn from file: %w", err)
	// }

	internalConnToVmFile, err := net.FileConn(vmFile)
	if err != nil {
		// _ = hostConn.Close() // also closes hostFile
		_ = vmFile.Close()
		return nil, nil, errors.Errorf("creating internalConnToVmFile from file: %w", err)
	}

	// Now, connect the vmFile end of the socket pair to the actual guest VSock port
	slog.InfoContext(ctx, "dialing actual guest VSock port from internal link", "guestVSockPort", guestVSockPort)
	guestVSockConn, err := vm.VSockConnect(ctx, guestVSockPort)
	if err != nil {
		// _ = hostConn.Close()
		_ = internalConnToVmFile.Close() // also closes vmFile
		return nil, nil, errors.Errorf("connecting to guest VSock port %d: %w", guestVSockPort, err)
	}
	slog.InfoContext(ctx, "successfully connected to guest VSock port", "guestVSockPort", guestVSockPort)

	// Start goroutines to proxy data in both directions
	// Use a new context for these goroutines that can be cancelled by the closerFunc
	proxyCtx, cancelProxy := context.WithCancel(ctx)

	go func() {
		defer cancelProxy() // Always signal, regardless of which connection is closed first
		// If internalConnToVmFile closes, io.Copy finishes. We shouldn't aggressively close guestVSockConn
		// here, as the other goroutine might still be reading from it. Let the main closer handle it.
		// defer guestVSockConn.Close() // <--- REMOVE THIS LINE

		slog.DebugContext(proxyCtx, "starting proxy: internalConnToVmFile -> guestVSockConn")
		_, copyErr := io.Copy(guestVSockConn, internalConnToVmFile)
		if copyErr != nil && !errors.Is(copyErr, io.EOF) && !errors.Is(copyErr, net.ErrClosed) && !errors.Is(copyErr, os.ErrClosed) {
			// syscall.EPIPE is "broken pipe"
			if !errors.Is(copyErr, unix.EPIPE) { // Don't log broken pipe as error if it's expected
				slog.ErrorContext(proxyCtx, "error copying from internalConnToVmFile to guestVSockConn", "error", copyErr)
			} else {
				slog.DebugContext(proxyCtx, "internalConnToVmFile -> guestVSockConn copy saw broken pipe (expected if peer closed)", "error", copyErr)
			}
		}
		slog.DebugContext(proxyCtx, "stopped proxy: internalConnToVmFile -> guestVSockConn")
	}()

	go func() {
		defer cancelProxy() // Always signal
		// If guestVSockConn closes (EOF from guest), io.Copy finishes.
		// We don't need to aggressively close internalConnToVmFile here;
		// the main closer or the host app's closure of hostConn will handle it.
		// defer internalConnToVmFile.Close() // This was already removed, which is good.

		slog.DebugContext(proxyCtx, "starting proxy: guestVSockConn -> internalConnToVmFile")
		_, copyErr := io.Copy(internalConnToVmFile, guestVSockConn)
		if copyErr != nil && !errors.Is(copyErr, io.EOF) && !errors.Is(copyErr, net.ErrClosed) && !errors.Is(copyErr, os.ErrClosed) {
			slog.ErrorContext(proxyCtx, "error copying from guestVSockConn to internalConnToVmFile", "error", copyErr)
		}
		slog.DebugContext(proxyCtx, "stopped proxy: guestVSockConn -> internalConnToVmFile")
	}()

	closer := func() {
		slog.InfoContext(ctx, "closing UnixSocketStreamConnection resources", "guestVSockPort", guestVSockPort)
		cancelProxy()
		// Order of closing here can be important to ensure graceful shutdown.
		// Closing hostConn signals to internalConnToVmFile if its io.Copy is blocked on read.
		// _ = hostConn.Close()
		// Closing internalConnToVmFile signals to guestVSockConn if its io.Copy is blocked on read.
		_ = internalConnToVmFile.Close()
		// Finally, ensure the connection to the guest is closed.
		_ = guestVSockConn.Close()
	}

	return hostFile, closer, nil
}

func StartVSockDevices(ctx context.Context, vm VirtualMachine) error {
	vsockDevs := virtio.VirtioDevicesOfType[*virtio.VirtioVsock](vm.Devices())
	for _, vsock := range vsockDevs {
		port := vsock.Port
		socketURL := vsock.SocketURL
		if socketURL == "" {
			// the timesync code adds a vsock device without an associated URL.
			// the ones that don't have urls are already set up on the main vsock
			continue
		}
		var listenStr string
		if vsock.Direction == virtio.VirtioVsockDirectionGuestConnectsAsClient {
			listenStr = " (listening)"
		}
		slog.InfoContext(ctx, "Exposing vsock port", "port", port, "socketURL", socketURL, "listenStr", listenStr)
		_, _, closer, err := ExposeVsock(ctx, vm, vsock.Port, vsock.Direction)
		if err != nil {
			slog.WarnContext(ctx, "error exposing vsock port", "port", port, "error", err)
			continue
		}
		defer closer()
	}
	return nil
}
