package hypervisors

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/url"
	"path/filepath"
	"strconv"

	"github.com/walteh/ec1/pkg/machines/host"
	"github.com/walteh/ec1/pkg/machines/virtio"
	"gitlab.com/tozd/go/errors"
	"inet.af/tcpproxy"
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

func ExposeVsock(ctx context.Context, vm VirtualMachine, proxiedDevice *virtio.VirtioVsock) (io.Closer, error) {
	if proxiedDevice.Direction == virtio.VirtioVsockDirectionGuestListensAsServer {
		return exposeConnectVsockProxy(ctx, vm, proxiedDevice)
	}
	return exposeListenVsockProxy(ctx, vm, proxiedDevice)
}

// connectVsock proxies connections from a host unix socket to a vsock port
// This allows the host to initiate connections to the guest over vsock
func exposeConnectVsockProxy(ctx context.Context, vm VirtualMachine, proxiedDevice *virtio.VirtioVsock) (io.Closer, error) {
	var proxy tcpproxy.Proxy

	empathicalCacheDir, err := host.EmphiricalVMCacheDir(ctx, vm.ID())
	if err != nil {
		return nil, err
	}

	vsockPath := filepath.Join(empathicalCacheDir, proxiedDevice.SocketURL)

	// listen for connections on the host unix socket
	proxy.ListenFunc = func(_, laddr string) (net.Listener, error) {
		parsed, err := url.Parse(laddr)
		if err != nil {
			return nil, err
		}
		switch parsed.Scheme {
		case "unix":
			addr := net.UnixAddr{Net: "unix", Name: parsed.EscapedPath()}
			return net.ListenUnix("unix", &addr)
		default:
			return nil, fmt.Errorf("unexpected scheme '%s'", parsed.Scheme)
		}
	}

	proxy.AddRoute(fmt.Sprintf("unix://:%s", vsockPath), &tcpproxy.DialProxy{
		Addr: fmt.Sprintf("vsock:%d", proxiedDevice.Port),
		// when there's a connection to the unix socket listener, connect to the specified vsock port
		DialContext: func(_ context.Context, _, addr string) (conn net.Conn, e error) {
			parsed, err := url.Parse(addr)
			if err != nil {
				return nil, err
			}
			switch parsed.Scheme {
			case "vsock":
				return vm.VSockConnect(ctx, proxiedDevice.Port)
			default:
				return nil, fmt.Errorf("unexpected scheme '%s'", parsed.Scheme)
			}
		},
	})
	return &proxy, proxy.Start()
}

// listenVsock proxies connections from a vsock port to a host unix socket.
// This allows the guest to initiate connections to the host over vsock
func exposeListenVsockProxy(ctx context.Context, vm VirtualMachine, proxiedDevice *virtio.VirtioVsock) (io.Closer, error) {
	var proxy tcpproxy.Proxy

	empathicalCacheDir, err := host.EmphiricalVMCacheDir(ctx, vm.ID())
	if err != nil {
		return nil, err
	}

	vsockPath := filepath.Join(empathicalCacheDir, proxiedDevice.SocketURL)

	// listen for connections on the vsock port
	proxy.ListenFunc = func(_, laddr string) (net.Listener, error) {
		parsed, err := url.Parse(laddr)
		if err != nil {
			return nil, err
		}
		switch parsed.Scheme {
		case "vsock":
			port, err := strconv.ParseUint(parsed.Port(), 10, 32)
			if err != nil {
				return nil, err
			}
			return vm.VSockListen(ctx, uint32(port)) //#nosec G115 -- strconv.ParseUint(_, _, 32) guarantees no overflow
		default:
			return nil, fmt.Errorf("unexpected scheme '%s'", parsed.Scheme)
		}
	}

	proxy.AddRoute(fmt.Sprintf("vsock://:%d", proxiedDevice.Port), &tcpproxy.DialProxy{
		Addr: fmt.Sprintf("unix:%s", vsockPath),
		// when there's a connection to the vsock listener, connect to the provided unix socket
		DialContext: func(ctx context.Context, _, addr string) (conn net.Conn, e error) {
			parsed, err := url.Parse(addr)
			if err != nil {
				return nil, err
			}
			switch parsed.Scheme {
			case "unix":
				var d net.Dialer
				return d.DialContext(ctx, parsed.Scheme, parsed.Path)
			default:
				return nil, fmt.Errorf("unexpected scheme '%s'", parsed.Scheme)
			}
		},
	})

	return &proxy, proxy.Start()
}
