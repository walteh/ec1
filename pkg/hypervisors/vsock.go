package hypervisors

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"

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

func ExposeVsock(ctx context.Context, vm VirtualMachine, port uint32, direction virtio.VirtioVsockDirection) (net.Conn, io.Closer, error) {
	if direction == virtio.VirtioVsockDirectionGuestListensAsServer {
		return exposeConnectVsockProxy(ctx, vm, port, direction)
	}
	return exposeListenVsockProxy(ctx, vm, port, direction)
}

// connectVsock proxies connections from a host unix socket to a vsock port
// This allows the host to initiate connections to the guest over vsock
func exposeConnectVsockProxy(ctx context.Context, vm VirtualMachine, port uint32, direction virtio.VirtioVsockDirection) (net.Conn, io.Closer, error) {
	var proxy tcpproxy.Proxy

	fd, err := unix.Socket(unix.AF_UNIX, unix.SOCK_STREAM, 0)
	if err != nil {
		return nil, nil, errors.Errorf("creating unix socket: %w", err)
	}

	vsockFile := os.NewFile(uintptr(fd), "vsock.socket")

	// vsockConn, err := net.FileConn(vsockFile)
	// if err != nil {
	// 	return nil, errors.Errorf("creating vsock conn: %w", err)
	// }

	// unixVsockConn := vsockConn.(*net.UnixConn)

	// listen for connections on the host unix socket
	proxy.ListenFunc = func(_, laddr string) (net.Listener, error) {
		return net.FileListener(vsockFile)
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
		return nil, nil, errors.Errorf("starting proxy: %w", err)
	}

	return conn, nil, nil
}

// listenVsock proxies connections from a vsock port to a host unix socket.
// This allows the guest to initiate connections to the host over vsock
func exposeListenVsockProxy(ctx context.Context, vm VirtualMachine, port uint32, direction virtio.VirtioVsockDirection) (net.Conn, io.Closer, error) {
	var proxy tcpproxy.Proxy

	fd, err := unix.Socket(unix.AF_UNIX, unix.SOCK_STREAM, 0)
	if err != nil {
		return nil, nil, errors.Errorf("creating unix socket: %w", err)
	}

	vsockFile := os.NewFile(uintptr(fd), "vsock.socket")

	// listen for connections on the vsock port
	proxy.ListenFunc = func(_, laddr string) (net.Listener, error) {
		return vm.VSockListen(ctx, uint32(port))
	}

	proxy.AddRoute(fmt.Sprintf("vsock://:%d", port), &tcpproxy.DialProxy{
		Addr: fmt.Sprintf("unix:%s", vsockFile.Name()),
		// when there's a connection to the vsock listener, connect to the provided unix socket
		DialContext: func(ctx context.Context, _, addr string) (conn net.Conn, e error) {
			return net.FileConn(vsockFile)
		},
	})

	return proxy.Start()
}
