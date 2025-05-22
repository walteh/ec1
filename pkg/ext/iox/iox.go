package iox

import (
	"context"
	"io"
	"log/slog"
	"sync"

	"gitlab.com/tozd/go/errors"
)

func PreservedNopCloser(r io.Reader) io.ReadCloser {
	// if rc, ok := r.(io.ReadCloser); ok {
	// 	return rc
	// }
	return io.NopCloser(r)
}

type ContextReader struct {
	ctx context.Context
	io.Reader
}

func NewContextReader(ctx context.Context, r io.Reader) *ContextReader {
	return &ContextReader{ctx: ctx, Reader: r}
}

func (r *ContextReader) Read(p []byte) (n int, err error) {
	if r.ctx.Err() != nil {
		return 0, r.ctx.Err()
	}
	return r.Reader.Read(p)
}

type ReadCounter struct {
	count int64
	io.Reader
	debug bool
}

func NewReadCounter(r io.Reader) *ReadCounter { return &ReadCounter{Reader: r} }

func (r *ReadCounter) Count() int64 { return r.count }

func (r *ReadCounter) Read(p []byte) (n int, err error) {
	n, err = r.Reader.Read(p)
	r.count += int64(n)
	if r.debug {
		slog.Debug("read", "count", r.count, "n", n, "err", err)
	}
	return
}

func (r *ReadCounter) SetDebug(debug bool) { r.debug = debug }

func CreateWriterPipeline(ctx context.Context, reader io.Reader, writerFunc func(io.Writer) (io.WriteCloser, error)) (io.ReadCloser, error) {

	pipeReader, pipeWriter := io.Pipe()

	wrtr, err := writerFunc(pipeWriter)
	if err != nil {
		pipeWriter.Close()
		return nil, errors.Errorf("opening compression writer: %w", err)
	}

	go func() {
		defer pipeWriter.Close()
		defer wrtr.Close()
		_, err := io.Copy(wrtr, reader)
		if err != nil {
			slog.Warn("error copying to pipeline writer", "error", err)
		}
	}()

	return pipeReader, nil
}

func CreateWriterPipelinez(ctx context.Context, reader io.Reader, writerFunc func(io.Writer) (io.WriteCloser, error)) (io.ReadCloser, error) {
	// Create a pipe for data flow
	pipeReader, pipeWriter := io.Pipe()

	// Create the writer using the provided function
	wrtr, err := writerFunc(pipeWriter)
	if err != nil {
		pipeWriter.Close() // Clean up the pipe writer
		pipeReader.Close() // Clean up the pipe reader
		return nil, errors.Errorf("opening compression writer: %w", err)
	}

	// Track whether the pipeline is already closed
	var closeOnce sync.Once
	cleanupFn := func(err error) {
		closeOnce.Do(func() {
			if err != nil {
				pipeWriter.CloseWithError(err)
			} else {
				pipeWriter.Close()
			}
			wrtr.Close()
		})
	}

	// Create a context that will be canceled when the pipe reader is closed
	pipeCtx, cancelPipe := context.WithCancel(context.Background())

	// Set up a cleanup when the pipe reader is closed
	originalCloseFn := pipeReader.Close
	wrappedReader := &contextReadCloser{
		ReadCloser: pipeReader,
		closeFunc: func() error {
			cancelPipe() // Cancel the pipe context when reader is closed
			return originalCloseFn()
		},
	}

	// Start the copy operation in a goroutine
	go func() {
		defer cleanupFn(nil) // Ensure everything is cleaned up when we exit

		// Create a channel to signal when copying is done
		done := make(chan struct{})
		var copyErr error

		// Start the actual copy in another goroutine
		go func() {
			defer close(done)
			_, copyErr = io.Copy(wrtr, reader)
		}()

		// Wait for copy to complete or context to be canceled
		select {
		case <-ctx.Done():
			// Context canceled by caller
			cleanupFn(ctx.Err())
			return
		case <-pipeCtx.Done():
			// Reader was closed by consumer
			cleanupFn(io.ErrClosedPipe)
			return
		case <-done:
			// Copy completed
			if copyErr != nil {
				slog.Warn("error copying to pipeline writer", "error", copyErr)
				cleanupFn(copyErr)
			}
		}
	}()

	return wrappedReader, nil
}

// contextReadCloser wraps a ReadCloser with a custom close function
type contextReadCloser struct {
	io.ReadCloser
	closeFunc func() error
}

func (c *contextReadCloser) Close() error {
	return c.closeFunc()
}
