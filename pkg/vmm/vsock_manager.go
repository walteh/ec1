package vmm

import (
	"context"
	"io"
	"log/slog"
	"time"

	"gitlab.com/tozd/go/errors"
)

type VSockManagerState string

const (
	// DefaultRetryInterval is the default time to wait between retry attempts
	DefaultRetryInterval = 5 * time.Millisecond // Much more aggressive: 5ms instead of 100ms
	// DefaultMaxRetries is the default maximum number of retry attempts
	DefaultMaxRetries = 4000 // Keep total time reasonable: 20 seconds max

	StateConnected    VSockManagerState = "connected"
	StateDisconnected VSockManagerState = "disconnected"
)

// VSockManager provides a retry mechanism for vsock connections
type VSockManager struct {
	state          VSockManagerState
	stateNotifiers []chan<- VSockManagerState
	// ConnectFunc is the function used to establish a vsock connection
	ConnectFunc func(ctx context.Context) (io.ReadWriteCloser, error)
	conn        io.ReadWriteCloser
	// RetryInterval is the time to wait between retry attempts
	RetryInterval time.Duration
	// MaxRetries is the maximum number of retry attempts (0 = infinite)
	MaxRetries int
}

// NewVSockManager creates a new VSockManager with default settings
func NewVSockManager(connectFunc func(ctx context.Context) (io.ReadWriteCloser, error)) *VSockManager {
	return &VSockManager{
		stateNotifiers: make([]chan<- VSockManagerState, 0),
		ConnectFunc:    connectFunc,
		RetryInterval:  DefaultRetryInterval,
		state:          StateDisconnected,
		MaxRetries:     DefaultMaxRetries,
	}
}

func (m *VSockManager) updateState(state VSockManagerState) {
	m.state = state
	for _, notifier := range m.stateNotifiers {
		go func() {
			notifier <- state
		}()
	}
}

func (m *VSockManager) State() VSockManagerState {
	return m.state
}

func (m *VSockManager) AddStateNotifier() chan VSockManagerState {
	notifier := make(chan VSockManagerState)
	m.stateNotifiers = append(m.stateNotifiers, notifier)
	return notifier
}

// Connect attempts to establish a vsock connection with retries
func (m *VSockManager) Dial(ctx context.Context) (io.ReadWriteCloser, error) {
	if m.conn != nil {
		return m.conn, nil
	}

	var lastErr error
	attempts := 0

	for {
		select {
		case <-ctx.Done():
			if lastErr != nil {
				return nil, errors.Errorf("vsock connection canceled: %w", lastErr)
			}
			return nil, errors.Errorf("vsock connection canceled: %w", ctx.Err())
		default:
			if m.conn != nil {
				return m.conn, nil
			}

			// Try to connect
			conn, err := m.ConnectFunc(ctx)
			if err == nil {
				slog.DebugContext(ctx, "vsock connection successful", "attempts", attempts)
				m.conn = NewManagedConnectionChild(conn, m)
				return m.conn, nil
			}

			slog.ErrorContext(ctx, "vsock connection failed", "error", err)

			lastErr = err
			attempts++

			// Check if max retries reached
			if m.MaxRetries > 0 && attempts >= m.MaxRetries {
				return nil, errors.Errorf("vsock connection failed after %d attempts: %w", attempts, lastErr)
			}

			// Wait before retrying
			select {
			case <-ctx.Done():
				return nil, errors.Errorf("vsock connection canceled during retry wait: %w", ctx.Err())
			case <-time.After(m.RetryInterval):
				// Continue to next retry
			}
		}
	}
}

// NewVSockTransport creates a transport that uses VSockManager for connections
func NewManagedConnectionChild(conn io.ReadWriteCloser, manager *VSockManager) io.ReadWriteCloser {
	manager.updateState(StateConnected)
	return &ManagedConnectionChild{
		conn:    conn,
		manager: manager,
	}
}

// VSockTransport implements io.ReadWriteCloser using VSockManager
type ManagedConnectionChild struct {
	manager *VSockManager
	conn    io.ReadWriteCloser
}

// Read implements io.Reader
func (t *ManagedConnectionChild) Read(p []byte) (n int, err error) {

	i, err := t.conn.Read(p)
	if err != nil {
		t.manager.updateState(StateDisconnected)
	}
	return i, err
}

// Write implements io.Writer
func (t *ManagedConnectionChild) Write(p []byte) (n int, err error) {
	i, err := t.conn.Write(p)
	if err != nil {
		t.manager.updateState(StateDisconnected)
	}
	return i, err
}

// Close implements io.Closer
func (t *ManagedConnectionChild) Close() error {
	t.manager.updateState(StateDisconnected)
	if t.conn != nil {
		err := t.conn.Close()
		return err
	}
	return nil
}
