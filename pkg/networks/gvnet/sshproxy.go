package gvnet

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"

	"golang.org/x/sync/errgroup"

	"gitlab.com/tozd/go/errors"
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

			// muxconn := clientConn.(*cmux.MuxConn)

			slog.InfoContext(ctx, "forwarding connection", "client", clientConn.RemoteAddr(), "backend", port, "conntype", fmt.Sprintf("%T", clientConn))
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
				CopyWithLoggingData(ctx, "client", clientConn, backend)
				done <- struct{}{}
			}()
			go func() {
				CopyWithLoggingData(ctx, "backend", backend, clientConn)
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

func CopyWithLoggingData(ctx context.Context, name string, src io.Reader, dst io.Writer) error {

	done := make(chan struct{}, 1)

	pr, pw := io.Pipe()

	tee := io.TeeReader(src, pw)

	go func() {
		io.Copy(dst, tee)
		done <- struct{}{}
	}()

	go func() {
		bufreader := bufio.NewScanner(pr)
		bufreader.Split(bufio.ScanBytes)
		slog.InfoContext(ctx, "starting to read from pipe")
		// log all content from the pw
		for bufreader.Scan() {
			// read each datagram from the pipe
			datagram := bufreader.Text()
			slog.InfoContext(ctx, "data", "name", name, "data", datagram)
		}
	}()

	<-done
	return nil
}
