package proc

import (
	"net"
	"os"
	"sync"
	"time"

	"golang.org/x/sys/unix"

	"gitlab.com/tozd/go/errors"
)

// --- Stream Socket Pair ---

// bidirectionalStreamConn wraps a net.Conn for state management within a pair.
// It implements the net.Conn interface.
type bidirectionalStreamConn struct {
	net.Conn
	mu     sync.Mutex
	closed bool
}

// Close marks the connection as closed and closes the underlying net.Conn.
func (c *bidirectionalStreamConn) Close() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return errors.New("connection already closed")
	}
	c.closed = true
	c.mu.Unlock()
	return c.Conn.Close()
}

// Check if the connection is closed before allowing operations.
func (c *bidirectionalStreamConn) checkClosed() error {
	c.mu.Lock()
	closed := c.closed
	c.mu.Unlock()
	if closed {
		// Use a standard error type that net operations might expect.
		return net.ErrClosed
	}
	return nil
}

// Override methods that should check the closed state first.
func (c *bidirectionalStreamConn) Read(b []byte) (n int, err error) {
	if err := c.checkClosed(); err != nil {
		return 0, err
	}
	return c.Conn.Read(b)
}

func (c *bidirectionalStreamConn) Write(b []byte) (n int, err error) {
	if err := c.checkClosed(); err != nil {
		return 0, err
	}
	return c.Conn.Write(b)
}

func (c *bidirectionalStreamConn) SetDeadline(t time.Time) error {
	if err := c.checkClosed(); err != nil {
		return err
	}
	return c.Conn.SetDeadline(t)
}

func (c *bidirectionalStreamConn) SetReadDeadline(t time.Time) error {
	if err := c.checkClosed(); err != nil {
		return err
	}
	return c.Conn.SetReadDeadline(t)
}

func (c *bidirectionalStreamConn) SetWriteDeadline(t time.Time) error {
	if err := c.checkClosed(); err != nil {
		return err
	}
	return c.Conn.SetWriteDeadline(t)
}

// CreateStreamSocketPair creates a connected pair of Unix domain sockets of type SOCK_STREAM.
// It returns two *bidirectionalStreamConn interfaces representing the connected endpoints
// and a cleanup function that closes both connections.
func CreateStreamSocketPair() (*bidirectionalStreamConn, *bidirectionalStreamConn, func() error, error) {
	fds, err := unix.Socketpair(unix.AF_UNIX, unix.SOCK_STREAM, 0)
	if err != nil {
		return nil, nil, nil, errors.Errorf("unix.Socketpair(SOCK_STREAM): %w", err)
	}

	file1 := os.NewFile(uintptr(fds[0]), "socketpair-stream-1")
	file2 := os.NewFile(uintptr(fds[1]), "socketpair-stream-2")

	rawConn1, err := net.FileConn(file1)
	if err != nil {
		_ = file1.Close()
		_ = file2.Close()
		return nil, nil, nil, errors.Errorf("net.FileConn for fd %d: %w", fds[0], err)
	}
	if err := file1.Close(); err != nil {
		_ = rawConn1.Close()
		_ = file2.Close()
		return nil, nil, nil, errors.Errorf("closing file wrapper for fd %d: %w", fds[0], err)
	}

	rawConn2, err := net.FileConn(file2)
	if err != nil {
		_ = rawConn1.Close()
		_ = file2.Close()
		return nil, nil, nil, errors.Errorf("net.FileConn for fd %d: %w", fds[1], err)
	}
	if err := file2.Close(); err != nil {
		_ = rawConn1.Close()
		_ = rawConn2.Close()
		return nil, nil, nil, errors.Errorf("closing file wrapper for fd %d: %w", fds[1], err)
	}

	// Wrap the raw connections
	conn1 := &bidirectionalStreamConn{Conn: rawConn1}
	conn2 := &bidirectionalStreamConn{Conn: rawConn2}

	// Cleanup function closes the two wrapper objects
	cleanup := func() error {
		var errs []error
		if err := conn1.Close(); err != nil && !errors.Is(err, net.ErrClosed) && err.Error() != "connection already closed" {
			errs = append(errs, errors.Errorf("closing conn1: %w", err))
		}
		if err := conn2.Close(); err != nil && !errors.Is(err, net.ErrClosed) && err.Error() != "connection already closed" {
			errs = append(errs, errors.Errorf("closing conn2: %w", err))
		}
		if len(errs) > 0 {
			return errors.Errorf("errors during stream cleanup: %v", errs)
		}
		return nil
	}

	return conn1, conn2, cleanup, nil
}

// --- Datagram Socket Pair ---

// bidirectionalDgramConn wraps a net.PacketConn for state management within a pair.
// It implements the net.PacketConn interface.
type bidirectionalDgramConn struct {
	net.PacketConn
	mu     sync.Mutex
	closed bool
}

// Close marks the connection as closed and closes the underlying net.PacketConn.
func (c *bidirectionalDgramConn) Close() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return errors.New("connection already closed")
	}
	c.closed = true
	c.mu.Unlock()
	return c.PacketConn.Close()
}

// Check if the connection is closed before allowing operations.
func (c *bidirectionalDgramConn) checkClosed() error {
	c.mu.Lock()
	closed := c.closed
	c.mu.Unlock()
	if closed {
		return net.ErrClosed
	}
	return nil
}

// Override methods that should check the closed state first.
func (c *bidirectionalDgramConn) ReadFrom(p []byte) (n int, addr net.Addr, err error) {
	if err := c.checkClosed(); err != nil {
		return 0, nil, err
	}
	return c.PacketConn.ReadFrom(p)
}

func (c *bidirectionalDgramConn) WriteTo(p []byte, addr net.Addr) (n int, err error) {
	if err := c.checkClosed(); err != nil {
		return 0, err
	}
	return c.PacketConn.WriteTo(p, addr)
}

func (c *bidirectionalDgramConn) SetDeadline(t time.Time) error {
	if err := c.checkClosed(); err != nil {
		return err
	}
	return c.PacketConn.SetDeadline(t)
}

func (c *bidirectionalDgramConn) SetReadDeadline(t time.Time) error {
	if err := c.checkClosed(); err != nil {
		return err
	}
	return c.PacketConn.SetReadDeadline(t)
}

func (c *bidirectionalDgramConn) SetWriteDeadline(t time.Time) error {
	if err := c.checkClosed(); err != nil {
		return err
	}
	return c.PacketConn.SetWriteDeadline(t)
}

// CreateDgramSocketPair creates a connected pair of Unix domain sockets of type SOCK_DGRAM.
// It returns two *bidirectionalDgramConn interfaces representing the connected endpoints
// and a cleanup function that closes both connections.
func CreateDgramSocketPair() (*bidirectionalDgramConn, *bidirectionalDgramConn, func() error, error) {
	fds, err := unix.Socketpair(unix.AF_UNIX, unix.SOCK_DGRAM, 0)
	if err != nil {
		return nil, nil, nil, errors.Errorf("unix.Socketpair(SOCK_DGRAM): %w", err)
	}

	file1 := os.NewFile(uintptr(fds[0]), "socketpair-dgram-1")
	file2 := os.NewFile(uintptr(fds[1]), "socketpair-dgram-2")

	rawConn1, err := net.FilePacketConn(file1)
	if err != nil {
		_ = file1.Close()
		_ = file2.Close()
		return nil, nil, nil, errors.Errorf("net.FilePacketConn for fd %d: %w", fds[0], err)
	}
	if err := file1.Close(); err != nil {
		_ = rawConn1.Close()
		_ = file2.Close()
		return nil, nil, nil, errors.Errorf("closing file wrapper for fd %d: %w", fds[0], err)
	}

	rawConn2, err := net.FilePacketConn(file2)
	if err != nil {
		_ = rawConn1.Close()
		_ = file2.Close()
		return nil, nil, nil, errors.Errorf("net.FilePacketConn for fd %d: %w", fds[1], err)
	}
	if err := file2.Close(); err != nil {
		_ = rawConn1.Close()
		_ = rawConn2.Close()
		return nil, nil, nil, errors.Errorf("closing file wrapper for fd %d: %w", fds[1], err)
	}

	// Wrap the raw connections
	conn1 := &bidirectionalDgramConn{PacketConn: rawConn1}
	conn2 := &bidirectionalDgramConn{PacketConn: rawConn2}

	// Cleanup function closes the two wrapper objects
	cleanup := func() error {
		var errs []error
		if err := conn1.Close(); err != nil && !errors.Is(err, net.ErrClosed) && err.Error() != "connection already closed" {
			errs = append(errs, errors.Errorf("closing conn1: %w", err))
		}
		if err := conn2.Close(); err != nil && !errors.Is(err, net.ErrClosed) && err.Error() != "connection already closed" {
			errs = append(errs, errors.Errorf("closing conn2: %w", err))
		}
		if len(errs) > 0 {
			return errors.Errorf("errors during dgram cleanup: %v", errs)
		}
		return nil
	}

	return conn1, conn2, cleanup, nil
}

// --- Common Helpers ---

// Helper type since sync.OnceFunc is Go 1.21+
type onceErrorCloser struct {
	close func() error
	once  sync.Once
	err   error
}

func (o *onceErrorCloser) Close() error {
	o.once.Do(func() {
		o.err = o.close()
	})
	return o.err
}

// NewOnceErrorCloser wraps a function with sync.Once for closing.
func NewOnceErrorCloser(closeFunc func() error) *onceErrorCloser {
	return &onceErrorCloser{close: closeFunc}
}
