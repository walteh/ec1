package hypervisors

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/sys/unix"

	"gitlab.com/tozd/go/errors"
	"inet.af/tcpproxy"

	"github.com/walteh/ec1/pkg/machines/host"
	"github.com/walteh/ec1/pkg/machines/virtio"
)

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
