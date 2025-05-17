package tapsock

import (
	"context"
	"io"
	"log/slog"
	"net"
	"os"
	"sync"
	"syscall"
	"time"

	"golang.org/x/sys/unix"

	"gitlab.com/tozd/go/errors"

	"github.com/walteh/ec1/pkg/virtio"
)

func unixFd(fd uintptr) int {
	// On unix the underlying fd is int, overflow is not possible.
	return int(fd) //#nosec G115 -- potential integer overflow
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func NewDgramVirtioNet(ctx context.Context, macstr string) (*virtio.VirtioNet, *VirtualNetworkRunner, error) {
	slog.InfoContext(ctx, "setting up unix socket pair", "macstr", macstr)

	mac, err := net.ParseMAC(macstr)
	if err != nil {
		return nil, nil, errors.Errorf("parsing mac: %w", err)
	}

	fds, err := unix.Socketpair(unix.AF_UNIX, unix.SOCK_DGRAM, unix.AF_UNSPEC)
	if err != nil {
		return nil, nil, errors.Errorf("creating socket pair: %w", err)
	}

	toCleanup := []io.Closer{}

	cleanups := func() {
		for _, closer := range toCleanup {
			closer.Close()
		}
	}

	defer func() {
		cleanups()
	}()

	slog.InfoContext(ctx, "created socketpair", "hostFd", fds[0], "vmFd", fds[1])

	hostSocket := os.NewFile(uintptr(fds[0]), "host.virtual.socket")
	vmSocket := os.NewFile(uintptr(fds[1]), "vm.virtual.socket")
	toCleanup = append(toCleanup, hostSocket, vmSocket)
	// IMPORTANT: we need to make a copy of vmSocket file descriptor for VirtioNet
	// Duplicate the file descriptor using syscall
	vmFdCopy, err := unix.Dup(fds[1])
	if err != nil {
		return nil, nil, errors.Errorf("duplicating VM file descriptor: %w", err)
	}
	vmSocketCopy := os.NewFile(uintptr(vmFdCopy), "vm.virtual.socket.copy")
	toCleanup = append(toCleanup, vmSocketCopy)

	hostConn, err := net.FilePacketConn(hostSocket)
	if err != nil {
		return nil, nil, errors.Errorf("creating hostConn: %w", err)
	}
	toCleanup = append(toCleanup, hostConn)
	// hostSocket.Close()
	// IMPORTANT: Don't close the underlying file descriptor after creating connections
	// hostSocket.Close() // close raw file now that hostConn holds the FD

	vmConn, err := net.FilePacketConn(vmSocket)
	if err != nil {
		return nil, nil, errors.Errorf("creating vmConn: %w", err)
	}
	toCleanup = append(toCleanup, vmConn)
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

	slog.InfoContext(ctx, "starting proxy goroutines")

	hostNetConn := NewBidirectionalDgramNetConn(hostConnUnix, vmConnUnix)

	virtioNet := &virtio.VirtioNet{
		MacAddress: mac,
		Nat:        false,
		Socket:     vmSocketCopy, // Use the duplicated socket for VirtioNet
		LocalAddr:  vmConnUnix.LocalAddr().(*net.UnixAddr),
	}

	// delegate cleanup to the VirtualNetworkRunner
	cleanups = func() {}

	runner := &VirtualNetworkRunner{
		name:    "virtual-network-runner(" + macstr + ")",
		netConn: hostNetConn,
		cleanup: func() error {
			for _, closer := range toCleanup {
				closer.Close()
			}
			return nil
		},
	}

	return virtioNet, runner, nil
}

// CloseFunc is a helper type that implements io.Closer
type CloseFunc func() error

func (f CloseFunc) Close() error {
	return f()
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
		remote: vmConn,
		host:   hostConn,
		closed: false,
	}
}

// tbh this thing should prob just be a couple funcitons, like close and remoteAddr on top od the host connection
type bidirectionalDgramNetConn struct {
	remote *net.UnixConn
	host   *net.UnixConn
	closed bool       // Track if this connection has been marked as closed
	mu     sync.Mutex // Protects closed flag
}

func (conn *bidirectionalDgramNetConn) RemoteAddr() net.Addr {
	return conn.remote.LocalAddr()
}

func (conn *bidirectionalDgramNetConn) Write(b []byte) (int, error) {

	// Log packet size and first few bytes for debugging
	// packetInfo := fmt.Sprintf("size=%d first_bytes=%x", len(b), b[:min(16, len(b))])
	// slog.Info("bidirectionalDgramNetConn.Write", "bytes", len(b), "packet_info", packetInfo)

	// // Try to detect SSH traffic for debugging
	// if len(b) >= 4 && string(b[:4]) == "SSH-" {
	// 	slog.Info("SSH packet detected in Write", "data", string(b[:min(64, len(b))]))
	// }

	// // Write the data with a deadline to prevent blocking forever
	// err := conn.host.SetWriteDeadline(time.Now().Add(1 * time.Second))
	// if err != nil {
	// 	slog.Error("bidirectionalDgramNetConn failed to set write deadline", "error", err)
	// }

	n, err := conn.host.Write(b)

	if err != nil {
		slog.Error("bidirectionalDgramNetConn.Write error", "error", err, "bytes_attempted", len(b))
	} else {
		slog.Info("bidirectionalDgramNetConn.Write success", "bytes", n)
	}
	return n, err
}

func (conn *bidirectionalDgramNetConn) Read(b []byte) (int, error) {

	// // Set a read deadline to prevent blocking forever
	// err := conn.host.SetReadDeadline(time.Now().Add(1 * time.Second))
	// if err != nil {
	// 	slog.Error("bidirectionalDgramNetConn failed to set read deadline", "error", err)
	// }

	n, err := conn.host.Read(b)

	if err != nil {
		slog.Error("bidirectionalDgramNetConn.Read error", "error", err)
	} else {
		slog.Info("bidirectionalDgramNetConn.Read success", "bytes", n)
	}
	// Clear the deadline
	// conn.host.SetReadDeadline(time.Time{})

	// if err != nil && err != io.EOF {
	// 	// Handle timeout by returning a temporary error
	// 	if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
	// 		slog.Debug("bidirectionalDgramNetConn.Read timeout", "error", err)
	// 		return 0, err
	// 	}

	// 	slog.Error("bidirectionalDgramNetConn.Read error", "error", err)
	// 	if isClosedConnError(err) {
	// 		conn.mu.Lock()
	// 		conn.closed = true
	// 		conn.mu.Unlock()
	// 	}
	// } else if n > 0 {
	// 	// Log packet details for debugging
	// 	packetInfo := fmt.Sprintf("size=%d first_bytes=%x", n, b[:min(16, n)])
	// 	slog.Info("bidirectionalDgramNetConn.Read success", "bytes", n, "packet_info", packetInfo)

	// 	// Try to detect SSH traffic for debugging
	// 	if n >= 4 && string(b[:4]) == "SSH-" {
	// 		slog.Info("SSH packet detected in Read", "data", string(b[:min(64, n)]))
	// 	}
	// }

	return n, err
}

func (conn *bidirectionalDgramNetConn) Close() error {
	conn.mu.Lock()
	defer conn.mu.Unlock()

	if conn.closed {
		return nil // Already closed
	}

	conn.closed = true
	slog.Info("bidirectionalDgramNetConn.Close - connections will be kept alive for tap.Switch")

	// Don't actually close the connections, just mark as closed
	// This allows tap.Switch to continue using them even after we're "closed"
	// The actual file descriptors will be closed when the VirtualNetworkRunner is shut down

	return nil
}

func (conn *bidirectionalDgramNetConn) LocalAddr() net.Addr {
	return conn.host.LocalAddr()
}

func (conn *bidirectionalDgramNetConn) SetDeadline(t time.Time) error {
	conn.mu.Lock()
	closed := conn.closed
	conn.mu.Unlock()

	if closed {
		return errors.New("use of closed network connection")
	}
	return conn.host.SetDeadline(t)
}

func (conn *bidirectionalDgramNetConn) SetReadDeadline(t time.Time) error {
	conn.mu.Lock()
	closed := conn.closed
	conn.mu.Unlock()

	if closed {
		return errors.New("use of closed network connection")
	}
	return conn.host.SetReadDeadline(t)
}

func (conn *bidirectionalDgramNetConn) SetWriteDeadline(t time.Time) error {
	conn.mu.Lock()
	closed := conn.closed
	conn.mu.Unlock()

	if closed {
		return errors.New("use of closed network connection")
	}
	return conn.host.SetWriteDeadline(t)
}

func (conn *bidirectionalDgramNetConn) SyscallConn() (syscall.RawConn, error) {
	conn.mu.Lock()
	closed := conn.closed
	conn.mu.Unlock()

	if closed {
		return nil, errors.New("use of closed network connection")
	}
	return conn.host.SyscallConn()
}
