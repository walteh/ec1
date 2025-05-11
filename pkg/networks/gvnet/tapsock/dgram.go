package tapsock

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/url"
	"os"
	"sync"
	"syscall"
	"time"

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

	slog.InfoContext(ctx, "created socketpair", "hostFd", fds[0], "vmFd", fds[1])

	hostSocket := os.NewFile(uintptr(fds[0]), "host.virtual.socket")
	vmSocket := os.NewFile(uintptr(fds[1]), "vm.virtual.socket")

	// IMPORTANT: we need to make a copy of vmSocket file descriptor for VirtioNet
	// Duplicate the file descriptor using syscall
	vmFdCopy, err := unix.Dup(fds[1])
	if err != nil {
		hostSocket.Close()
		vmSocket.Close()
		return nil, nil, errors.Errorf("duplicating VM file descriptor: %w", err)
	}
	vmSocketCopy := os.NewFile(uintptr(vmFdCopy), "vm.virtual.socket.copy")

	hostConn, err := net.FilePacketConn(hostSocket)
	if err != nil {
		hostSocket.Close()
		vmSocket.Close()
		vmSocketCopy.Close()
		return nil, nil, errors.Errorf("creating hostConn: %w", err)
	}
	// hostSocket.Close()
	// IMPORTANT: Don't close the underlying file descriptor after creating connections
	// hostSocket.Close() // close raw file now that hostConn holds the FD

	vmConn, err := net.FilePacketConn(vmSocket)
	if err != nil {
		hostConn.Close()
		vmSocketCopy.Close()
		return nil, nil, errors.Errorf("creating vmConn: %w", err)
	}
	// vmSocket.Close() // close raw file now that vmConn holds the FD

	hostConnUnix, ok := hostConn.(*net.UnixConn)
	if !ok {
		hostConn.Close()
		vmConn.Close()
		vmSocketCopy.Close()
		return nil, nil, errors.New("hostConn is not a UnixConn")
	}

	vmConnUnix, ok := vmConn.(*net.UnixConn)
	if !ok {
		hostConnUnix.Close()
		vmConn.Close()
		vmSocketCopy.Close()
		return nil, nil, errors.New("vmConn is not a UnixConn")
	}

	err = setDgramUnixBuffers(hostConnUnix)
	if err != nil {
		hostConnUnix.Close()
		vmConnUnix.Close()
		vmSocketCopy.Close()
		return nil, nil, errors.Errorf("setting host unix buffers: %w", err)
	}

	err = setDgramUnixBuffers(vmConnUnix)
	if err != nil {
		hostConnUnix.Close()
		vmConnUnix.Close()
		vmSocketCopy.Close()
		return nil, nil, errors.Errorf("setting vm unix buffers: %w", err)
	}

	// Create a cancellable context for the proxy goroutines
	// proxyCtx, proxyCancel := context.WithCancel(ctx)

	slog.InfoContext(ctx, "starting proxy goroutines")
	hostToVmErrChan := make(chan error, 1)
	vmToHostErrChan := make(chan error, 1)

	// Start the "host to VM" proxy
	// go func() {
	// 	proxyName := "from hostConnUnix to vmConnUnix"
	// 	slog.InfoContext(proxyCtx, "starting proxy goroutine", "name", proxyName)
	// 	bytes, err := unixConnLogProxy(proxyCtx, proxyName, hostConnUnix, vmConnUnix)
	// 	if err != nil {
	// 		slog.ErrorContext(ctx, "proxy exited with error", "name", proxyName, "bytes", bytes, "error", err)
	// 	} else {
	// 		slog.InfoContext(ctx, "proxy exited normally", "name", proxyName, "bytes", bytes)
	// 	}
	// 	hostToVmErrChan <- err
	// 	close(hostToVmErrChan)
	// }()

	// // Start the "VM to host" proxy
	// go func() {
	// 	proxyName := "from vmConnUnix to hostConnUnix"
	// 	slog.InfoContext(proxyCtx, "starting proxy goroutine", "name", proxyName)
	// 	bytes, err := unixConnLogProxy(proxyCtx, proxyName, vmConnUnix, hostConnUnix)
	// 	if err != nil {
	// 		slog.ErrorContext(ctx, "proxy exited with error", "name", proxyName, "bytes", bytes, "error", err)
	// 	} else {
	// 		slog.InfoContext(ctx, "proxy exited normally", "name", proxyName, "bytes", bytes)
	// 	}
	// 	vmToHostErrChan <- err
	// 	close(vmToHostErrChan)
	// }()

	// // Send test packets in both directions to test connectivity
	// go sendTestPackets(proxyCtx, hostConnUnix, vmConnUnix)

	hostNetConn := NewBidirectionalDgramNetConn(hostConnUnix, vmConnUnix)

	virtioNet := &virtio.VirtioNet{
		MacAddress: mac,
		Nat:        false,
		Socket:     vmSocketCopy, // Use the duplicated socket for VirtioNet
		LocalAddr:  vmConnUnix.LocalAddr().(*net.UnixAddr),
	}

	runner := &VirtualNetworkRunner{
		name:         "virtual-network-runner(" + macstr + ")",
		netConn:      hostNetConn,
		hostConnUnix: hostConnUnix, // Set direct reference to hostConnUnix
		vmConnUnix:   vmConnUnix,   // Set direct reference to vmConnUnix
		toClose: map[string]io.Closer{
			"vmConnUnix":   vmConnUnix,
			"hostConnUnix": hostConnUnix,
			"vmConn":       vmConn,
			"hostConn":     hostConn,
			"vmSocketCopy": vmSocketCopy,
			// "proxyCancel":  CloseFunc(func() error { proxyCancel(); return nil }),
		},
		proxyErrChans: []<-chan error{hostToVmErrChan, vmToHostErrChan},
	}

	return virtioNet, runner, nil
}

// CloseFunc is a helper type that implements io.Closer
type CloseFunc func() error

func (f CloseFunc) Close() error {
	return f()
}

func unixConnLogProxy(ctx context.Context, name string, from *net.UnixConn, to *net.UnixConn) (int, error) {
	slog.InfoContext(ctx, "=== PROXY STARTED ===", "name", name,
		"from", from.LocalAddr().String(),
		"to", to.LocalAddr().String())

	var totalBytes int
	var readAttempts int
	var writeAttempts int
	var readSuccess int
	var writeSuccess int

	// Create a buffer with a reasonable size for network packets
	// SSH packets can be larger, so increase the buffer size
	buf := make([]byte, 65536) // 64KB buffer for better performance

	for {
		readAttempts++

		// Check if context is done before each read
		select {
		case <-ctx.Done():
			slog.InfoContext(ctx, "Context done in proxy", "name", name)
			return totalBytes, nil
		default:
			// Continue with the normal flow
		}

		// Set a shorter read deadline to prevent blocking for too long
		readDeadlineErr := from.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		if readDeadlineErr != nil {
			if isClosedConnError(readDeadlineErr) {
				slog.InfoContext(ctx, "=== PROXY FINISHED (CONNECTION CLOSED) ===", "name", name)
				return totalBytes, nil
			}
			slog.WarnContext(ctx, "failed to set read deadline", "name", name, "error", readDeadlineErr)
		}

		// Add blocking indicator so we know if we're stuck on read
		slog.DebugContext(ctx, "waiting to read data", "name", name, "attempt", readAttempts)

		n, err := from.Read(buf)

		// Very detailed error logging
		if err != nil {
			errStr := fmt.Sprintf("%v", err)
			errType := fmt.Sprintf("%T", err)

			// Handle timeout by continuing
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				// Just a timeout, not an error - continue with the loop
				continue
			}

			// Handle closed connection gracefully
			if isClosedConnError(err) {
				slog.InfoContext(ctx, "=== PROXY FINISHED (CONN CLOSED DURING READ) ===",
					"name", name,
					"read_attempts", readAttempts,
					"read_success", readSuccess,
					"write_attempts", writeAttempts,
					"write_success", writeSuccess)
				return totalBytes, nil // Return nil error since this is expected
			}

			slog.ErrorContext(ctx, "=== PROXY ERROR (READ) ===",
				"name", name,
				"error", err,
				"error_type", errType,
				"error_string", errStr)

			// Report the real error for debugging but don't propagate
			return totalBytes, fmt.Errorf("reading from unix conn (%s): %w", name, err)
		}

		readSuccess++

		// Clear the deadline after successful read
		clearDeadlineErr := from.SetReadDeadline(time.Time{})
		if clearDeadlineErr != nil {
			if isClosedConnError(clearDeadlineErr) {
				slog.InfoContext(ctx, "=== PROXY FINISHED (CONN CLOSED WHEN CLEARING DEADLINE) ===", "name", name)
				return totalBytes, nil
			}
			slog.WarnContext(ctx, "failed to clear read deadline", "name", name, "error", clearDeadlineErr)
		}

		slog.InfoContext(ctx, "!!! PACKET RECEIVED !!!",
			"name", name,
			"bytes", n,
			"first_bytes_hex", fmt.Sprintf("%x", buf[:min(n, 16)])) // Log at most first 16 bytes as hex

		// Check for SSH protocol signature in the first few bytes
		if n >= 4 {
			// SSH protocol starts with "SSH-" (0x5353482D)
			if string(buf[:4]) == "SSH-" {
				slog.InfoContext(ctx, "!!! SSH PACKET DETECTED !!!",
					"name", name,
					"bytes", n,
					"data", string(buf[:min(n, 64)]))
			}
		}

		writeAttempts++

		// Set a write deadline to avoid blocking on write
		writeDeadlineErr := to.SetWriteDeadline(time.Now().Add(500 * time.Millisecond))
		if writeDeadlineErr != nil {
			if isClosedConnError(writeDeadlineErr) {
				slog.InfoContext(ctx, "=== PROXY FINISHED (CONN CLOSED WHEN SETTING WRITE DEADLINE) ===", "name", name)
				return totalBytes, nil
			}
			slog.WarnContext(ctx, "failed to set write deadline", "name", name, "error", writeDeadlineErr)
		}

		written, err := to.Write(buf[:n])

		// Clear write deadline
		to.SetWriteDeadline(time.Time{})

		if err != nil {
			// Handle closed connection gracefully
			if isClosedConnError(err) {
				slog.InfoContext(ctx, "=== PROXY FINISHED (CONN CLOSED DURING WRITE) ===", "name", name)
				return totalBytes, nil
			}

			slog.ErrorContext(ctx, "=== PROXY ERROR (WRITE) ===", "name", name, "error", err)
			// Report the real error for debugging but don't propagate
			return totalBytes, fmt.Errorf("writing to unix conn (%s): %w", name, err)
		}

		writeSuccess++
		slog.InfoContext(ctx, "!!! PACKET FORWARDED !!!", "name", name, "bytes", written)

		totalBytes += n
	}
}

// isClosedConnError returns true if the error is related to using a closed connection
func isClosedConnError(err error) bool {
	if err == nil {
		return false
	}

	// Check for standard EOF
	if err == io.EOF {
		return true
	}

	// Check for "use of closed network connection"
	errStr := err.Error()
	return errStr == "use of closed network connection" ||
		errStr == "read unixgram ->: use of closed network connection" ||
		errStr == "write unixgram ->: use of closed network connection"
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
	slog.Info("creating new bidirectionalDgramNetConn",
		"host_addr", hostConn.LocalAddr().String(),
		"vm_addr", vmConn.LocalAddr().String())

	return &bidirectionalDgramNetConn{
		remote: vmConn,
		host:   hostConn,
		closed: false,
	}
}

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
	conn.mu.Lock()
	closed := conn.closed
	conn.mu.Unlock()

	if closed {
		return 0, errors.New("use of closed network connection")
	}

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

	// Clear the deadline
	// conn.host.SetWriteDeadline(time.Time{})

	if err != nil {
		slog.Error("bidirectionalDgramNetConn.Write error", "error", err, "bytes_attempted", len(b))
		if isClosedConnError(err) {
			conn.mu.Lock()
			conn.closed = true
			conn.mu.Unlock()
		}
	} else {
		slog.Info("bidirectionalDgramNetConn.Write success", "bytes", n)
	}
	return n, err
}

func (conn *bidirectionalDgramNetConn) Read(b []byte) (int, error) {
	conn.mu.Lock()
	closed := conn.closed
	conn.mu.Unlock()

	if closed {
		return 0, errors.New("use of closed network connection")
	}

	slog.Info("bidirectionalDgramNetConn.Read attempt", "buffer_size", len(b))

	// // Set a read deadline to prevent blocking forever
	// err := conn.host.SetReadDeadline(time.Now().Add(1 * time.Second))
	// if err != nil {
	// 	slog.Error("bidirectionalDgramNetConn failed to set read deadline", "error", err)
	// }

	n, err := conn.host.Read(b)

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

// sendTestPackets sends test packets in both directions to verify socket connectivity
func sendTestPackets(ctx context.Context, hostConn, vmConn *net.UnixConn) {
	// Wait a bit for everything to initialize
	time.Sleep(500 * time.Millisecond)

	// Create a simple Ethernet frame (14 byte header + payload)
	// Destination MAC: ff:ff:ff:ff:ff:ff (broadcast)
	// Source MAC: 00:00:00:00:00:01 (made-up source)
	// EtherType: 0x0800 (IPv4)
	// Then some dummy payload
	ethernetHeader := []byte{
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, // Destination MAC: broadcast
		0x00, 0x00, 0x00, 0x00, 0x00, 0x01, // Source MAC: made-up
		0x08, 0x00, // EtherType: IPv4
	}

	// Create test packets of different sizes
	smallTestPayload := []byte("EC1_TEST_PACKET_FOR_SOCKET_DEBUGGING")
	largeTestPayload := make([]byte, 4096) // 4KB payload
	for i := range largeTestPayload {
		largeTestPayload[i] = byte(i % 256) // Fill with pattern
	}

	// Copy SSH header pattern for testing
	sshTestPayload := []byte("SSH-2.0-OpenSSH_8.9 EC1_TEST_SSH_PACKET")

	// Combine header with payloads
	smallTestPacket := append(ethernetHeader, smallTestPayload...)
	largeTestPacket := append(ethernetHeader, largeTestPayload...)
	sshTestPacket := append(ethernetHeader, sshTestPayload...)

	slog.InfoContext(ctx, "created test packets",
		"small_size", len(smallTestPacket),
		"large_size", len(largeTestPacket),
		"ssh_size", len(sshTestPacket))

	// Set up a ticker to send packets every 5 seconds
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	// Send initial test packets
	sendTestPacketBatch(ctx, hostConn, vmConn, smallTestPacket, largeTestPacket, sshTestPacket)

	// Keep sending test packets at regular intervals
	for {
		select {
		case <-ctx.Done():
			slog.InfoContext(ctx, "stopping test packet sender due to context cancellation")
			return
		case <-ticker.C:
			sendTestPacketBatch(ctx, hostConn, vmConn, smallTestPacket, largeTestPacket, sshTestPacket)
		}
	}
}

// sendTestPacketBatch sends a batch of test packets in both directions
func sendTestPacketBatch(ctx context.Context, hostConn, vmConn *net.UnixConn, smallPacket, largePacket, sshPacket []byte) {
	sendPacket := func(conn *net.UnixConn, packet []byte, direction, packetType string) {
		slog.InfoContext(ctx, "sending test packet",
			"direction", direction,
			"type", packetType,
			"size", len(packet))

		// Set a write deadline
		err := conn.SetWriteDeadline(time.Now().Add(500 * time.Millisecond))
		if err != nil {
			slog.ErrorContext(ctx, "failed to set write deadline",
				"direction", direction,
				"error", err)
			return
		}

		n, err := conn.Write(packet)

		// Clear the deadline
		conn.SetWriteDeadline(time.Time{})

		if err != nil {
			slog.ErrorContext(ctx, "failed to send test packet",
				"direction", direction,
				"type", packetType,
				"error", err)
		} else {
			slog.InfoContext(ctx, "sent test packet successfully",
				"direction", direction,
				"type", packetType,
				"bytes", n)
		}
	}

	// Host to VM direction
	sendPacket(hostConn, smallPacket, "host->vm", "small")
	time.Sleep(100 * time.Millisecond)
	sendPacket(hostConn, sshPacket, "host->vm", "ssh")
	time.Sleep(100 * time.Millisecond)
	sendPacket(hostConn, largePacket, "host->vm", "large")

	time.Sleep(500 * time.Millisecond)

	// VM to Host direction
	sendPacket(vmConn, smallPacket, "vm->host", "small")
	time.Sleep(100 * time.Millisecond)
	sendPacket(vmConn, sshPacket, "vm->host", "ssh")
	time.Sleep(100 * time.Millisecond)
	sendPacket(vmConn, largePacket, "vm->host", "large")
}
