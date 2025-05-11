package tapsock

import (
	"context"
	"io"
	"log/slog"
	"net"
	"net/url"
	"os"
	"syscall"

	"golang.org/x/sync/errgroup"
	"golang.org/x/sys/unix"

	"github.com/containers/gvisor-tap-vsock/pkg/tap"
	"github.com/containers/gvisor-tap-vsock/pkg/transport"
	"github.com/containers/gvisor-tap-vsock/pkg/types"
	"gitlab.com/tozd/go/errors"

	"github.com/walteh/ec1/pkg/machines/virtio"
)

func unixFd(fd uintptr) int {
	// On unix the underlying fd is int, overflow is not possible.
	return int(fd) //#nosec G115 -- potential integer overflow
}

func NewDgramVirtioNet(ctx context.Context, macstr string) (*virtio.VirtioNet, *VirtualNetworkRunner, error) {

	mac, err := net.ParseMAC(macstr)
	if err != nil {
		return nil, nil, errors.Errorf("parsing mac: %w", err)
	}

	fds, err := unix.Socketpair(unix.AF_UNIX, unix.SOCK_DGRAM, unix.AF_UNSPEC)
	if err != nil {
		return nil, nil, errors.Errorf("creating socket pair: %w", err)
	}

	hostSocket := os.NewFile(uintptr(fds[0]), "host.virtual.socket")
	vmSocket := os.NewFile(uintptr(fds[1]), "vm.virtual.socket")

	hostConn, err := net.FilePacketConn(hostSocket)
	if err != nil {
		return nil, nil, errors.Errorf("creating hostConn: %w", err)
	}
	// hostSocket.Close() // close raw file now that hostConn holds the FD

	vmConn, err := net.FilePacketConn(vmSocket)
	if err != nil {
		return nil, nil, errors.Errorf("creating vmConn: %w", err)
	}
	// vmSocket.Close() // close raw file now that vmConn holds the FD

	hostConnUnix, ok := hostConn.(*net.UnixConn)
	if !ok {
		return nil, nil, errors.New("hostConn is not a UnixConn")
	}

	vmConnUnix, ok := vmConn.(*net.UnixConn)
	if !ok {
		return nil, nil, errors.New("vmConn is not a UnixConn")
	}

	err = setDgramUnixBuffers(hostConnUnix)
	if err != nil {
		return nil, nil, errors.Errorf("setting host unix buffers: %w", err)
	}

	err = setDgramUnixBuffers(vmConnUnix)
	if err != nil {
		return nil, nil, errors.Errorf("setting vm unix buffers: %w", err)
	}

	hostNetConn := NewBidirectionalDgramNetConn(vmConnUnix, hostConnUnix)

	virtioNet := &virtio.VirtioNet{
		MacAddress: mac,
		Nat:        false,
		Socket:     vmSocket,
		LocalAddr:  vmConnUnix.LocalAddr().(*net.UnixAddr),
	}

	runner := &VirtualNetworkRunner{
		name:    "virtual-network-runner(" + macstr + ")",
		netConn: hostNetConn,
		toClose: map[string]io.Closer{
			"vmConnUnix":   vmConnUnix,
			"hostConnUnix": hostConnUnix,
			"vmConn":       vmConn,
			"hostConn":     hostConn,
		},
	}

	return virtioNet, runner, nil
}

func setDgramUnixBuffers(conn *net.UnixConn) error {
	rawConn, err := conn.SyscallConn()
	if err != nil {
		return err
	}

	err = rawConn.Control(func(fd uintptr) {
		if err = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_SNDBUF, 1*1024*1024); err != nil {
			return
		}
		if err = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_RCVBUF, 4*1024*1024); err != nil {
			return
		}
	})
	if err != nil {
		return err
	}
	return nil
}

var _ net.Conn = (*bidirectionalDgramNetConn)(nil)

func NewBidirectionalDgramNetConn(hostConn *net.UnixConn, vmConn *net.UnixConn) *bidirectionalDgramNetConn {
	return &bidirectionalDgramNetConn{
		remote:   vmConn,
		UnixConn: hostConn,
	}
}

type bidirectionalDgramNetConn struct {
	remote *net.UnixConn
	*net.UnixConn
}

func (conn *bidirectionalDgramNetConn) RemoteAddr() net.Addr {
	return conn.remote.LocalAddr()
}

func (conn *bidirectionalDgramNetConn) Write(b []byte) (int, error) {
	return conn.UnixConn.WriteTo(b, conn.remote.LocalAddr())
}

func (s *VFKitSocket) badUnixDialCode(ctx context.Context, mac net.HardwareAddr, g *errgroup.Group, vn *tap.Switch) (*virtio.VirtioNet, error) {
	localAddr := &net.UnixAddr{Name: s.Path, Net: "unixgram"}

	conn, err := net.DialUnix("unixgram", localAddr, localAddr)
	if err != nil {
		return nil, errors.Errorf("vfkit listen error %w", err)
	}

	g.Go(func() error {
		<-ctx.Done()
		if err := conn.Close(); err != nil {
			slog.WarnContext(ctx, "error closing vfkit socket", "socket", s.Path, "error", err)
		}
		vfkitSocketURI, _ := url.Parse(s.Path)
		return os.Remove(vfkitSocketURI.Path)
	})

	g.Go(func() error {
		vfkitConn, err := transport.AcceptVfkit(conn)
		if err != nil {
			return errors.Errorf("vfkit accept error %w", err)
		}
		return vn.Accept(ctx, vfkitConn, types.VfkitProtocol)
	})

	rawConn, err := conn.SyscallConn()
	if err != nil {
		return nil, errors.Errorf("getting syscall conn: %w", err)
	}

	err = rawConn.Control(func(fd uintptr) {
		err := syscall.SetsockoptInt(unixFd(fd), syscall.SOL_SOCKET, syscall.SO_SNDBUF, 1*1024*1024)
		if err != nil {
			return
		}
		err = syscall.SetsockoptInt(unixFd(fd), syscall.SOL_SOCKET, syscall.SO_RCVBUF, 4*1024*1024)
		if err != nil {
			return
		}
	})
	if err != nil {
		return nil, errors.Errorf("setting socket options: %w", err)
	}

	fd, err := conn.File()
	if err != nil {
		return nil, errors.Errorf("getting file: %w", err)
	}

	virtioNet := &virtio.VirtioNet{
		MacAddress: mac,
		Nat:        false,
		Socket:     fd,
		LocalAddr:  localAddr,
	}

	return virtioNet, nil
}
