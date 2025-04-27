package gvproxy

import (
	"context"
	"io"
	"log/slog"
	"net"

	"gitlab.com/tozd/go/errors"
	"golang.org/x/sync/errgroup"
)

func ForwardListenerToPort(ctx context.Context, listener net.Listener, port string, errgroup *errgroup.Group) error {
	for {
		// Accept connection with timeout
		clientConn, err := listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return nil // Normal shutdown
			}
			return errors.Errorf("failed to accept: %w", err)
		}

		// Handle each client in a separate goroutine
		errgroup.Go(func() error {
			defer clientConn.Close()
			slog.InfoContext(ctx, "forwarding connection", "client", clientConn.RemoteAddr(), "backend", port)
			// Connect to the backend FOR THIS CLIENT
			backend, err := net.Dial("tcp", port)
			if err != nil {
				return errors.Errorf("failed to connect to backend: %w", err)
			}

			defer backend.Close()

			slog.InfoContext(ctx, "connected to backend", "backend", backend.RemoteAddr())

			// Use proper copying with context cancellation
			done := make(chan struct{}, 2)
			go func() {
				io.Copy(backend, clientConn)
				done <- struct{}{}
			}()
			go func() {
				io.Copy(clientConn, backend)
				done <- struct{}{}
			}()

			// Wait for either copy to finish or context to cancel
			select {
			case <-done:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		})
	}
}
