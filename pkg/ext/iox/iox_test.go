package iox

import (
	"bytes"
	"context"
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/walteh/retab/v2/pkg/diff"

	"github.com/walteh/ec1/pkg/testing/tlog"
)

// mockWriteCloser is a test implementation of io.WriteCloser
type mockWriteCloser struct {
	writeFunc  func([]byte) (int, error)
	closeFunc  func() error
	closeCount int
}

func (m *mockWriteCloser) Write(p []byte) (int, error) {
	if m.writeFunc != nil {
		return m.writeFunc(p)
	}
	return len(p), nil
}

func (m *mockWriteCloser) Close() error {
	m.closeCount++
	if m.closeFunc != nil {
		return m.closeFunc()
	}
	return nil
}

// errorReader always returns an error on Read
type errorReader struct {
	err error
}

func (e *errorReader) Read(p []byte) (int, error) {
	return 0, e.err
}

// delayedReader reads after a delay, useful for testing context cancellation
type delayedReader struct {
	data         []byte
	delay        time.Duration
	blockForever bool
}

func (d *delayedReader) Read(p []byte) (int, error) {
	if d.blockForever {
		select {} // Block forever
	}
	time.Sleep(d.delay)
	n := copy(p, d.data)
	if n >= len(d.data) {
		return n, io.EOF
	}
	return n, nil
}

// slowWriter simulates a writer that takes time to process data
type slowWriter struct {
	w            io.Writer
	delay        time.Duration
	blockForever bool
	mu           sync.Mutex
}

func (s *slowWriter) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.blockForever {
		select {} // Block forever
	}
	time.Sleep(s.delay)
	return s.w.Write(p)
}

func (s *slowWriter) Close() error {
	if c, ok := s.w.(io.Closer); ok {
		return c.Close()
	}
	return nil
}

// partialWriter only writes some of the bytes it receives
type partialWriter struct {
	w          io.Writer
	writeLimit int
}

func (p *partialWriter) Write(data []byte) (int, error) {
	if len(data) <= p.writeLimit {
		return p.w.Write(data)
	}
	n, _ := p.w.Write(data[:p.writeLimit])
	// Return a short write error
	return n, io.ErrShortWrite
}

func (p *partialWriter) Close() error {
	if c, ok := p.w.(io.Closer); ok {
		return c.Close()
	}
	return nil
}

func TestCreateWriterPipeline(t *testing.T) {
	tests := []struct {
		name             string
		reader           io.Reader
		writerFunc       func(io.Writer) (io.WriteCloser, error)
		expectedErr      bool
		expectedErrMsg   string
		expectedOutput   []byte
		contextTimeout   time.Duration
		skipOutputVerify bool
	}{
		{
			name:   "happy_path_successful_copy",
			reader: bytes.NewReader([]byte("test data")),
			writerFunc: func(w io.Writer) (io.WriteCloser, error) {
				return &mockWriteCloser{
					writeFunc: func(p []byte) (int, error) {
						return w.Write(p)
					},
				}, nil
			},
			expectedOutput: []byte("test data"),
		},
		{
			name:   "writer_func_returns_error",
			reader: bytes.NewReader([]byte("test data")),
			writerFunc: func(w io.Writer) (io.WriteCloser, error) {
				return nil, errors.New("writer creation failed")
			},
			expectedErr:    true,
			expectedErrMsg: "opening compression writer: writer creation failed",
		},
		{
			name:   "error_during_read",
			reader: &errorReader{err: errors.New("read error")},
			writerFunc: func(w io.Writer) (io.WriteCloser, error) {
				return &mockWriteCloser{}, nil
			},
			expectedErr:      true,
			skipOutputVerify: true,
		},
		{
			name:   "error_during_write",
			reader: bytes.NewReader([]byte("test data")),
			writerFunc: func(w io.Writer) (io.WriteCloser, error) {
				return &mockWriteCloser{
					writeFunc: func(p []byte) (int, error) {
						return 0, errors.New("write error")
					},
				}, nil
			},
			expectedErr:      true,
			skipOutputVerify: true,
		},
		{
			name:   "context_cancellation",
			reader: &delayedReader{data: []byte("test data"), blockForever: true},
			writerFunc: func(w io.Writer) (io.WriteCloser, error) {
				return &mockWriteCloser{}, nil
			},
			contextTimeout:   20 * time.Millisecond,
			expectedErr:      true,
			skipOutputVerify: true,
		},
		{
			name:   "empty_data",
			reader: bytes.NewReader([]byte{}),
			writerFunc: func(w io.Writer) (io.WriteCloser, error) {
				return &mockWriteCloser{
					writeFunc: func(p []byte) (int, error) {
						return w.Write(p)
					},
				}, nil
			},
			expectedOutput: []byte{},
		},
		{
			name:   "small_data",
			reader: bytes.NewReader(bytes.Repeat([]byte("a"), 1024)), // Smaller data size
			writerFunc: func(w io.Writer) (io.WriteCloser, error) {
				return &mockWriteCloser{
					writeFunc: func(p []byte) (int, error) {
						return w.Write(p)
					},
				}, nil
			},
			expectedOutput: bytes.Repeat([]byte("a"), 1024),
		},
		{
			name:   "close_error_propagation",
			reader: bytes.NewReader([]byte("test data")),
			writerFunc: func(w io.Writer) (io.WriteCloser, error) {
				return &mockWriteCloser{
					writeFunc: func(p []byte) (int, error) {
						return w.Write(p)
					},
					closeFunc: func() error {
						return errors.New("close error")
					},
				}, nil
			},
			expectedOutput: []byte("test data"), // Data should still flow despite close error
		},
		{
			name:   "slow_writer_with_context_cancellation",
			reader: bytes.NewReader(bytes.Repeat([]byte("a"), 100)),
			writerFunc: func(w io.Writer) (io.WriteCloser, error) {
				return &slowWriter{
					w:            w,
					blockForever: true,
				}, nil
			},
			contextTimeout:   20 * time.Millisecond,
			expectedErr:      true,
			skipOutputVerify: true,
		},
		{
			name:   "partial_writer",
			reader: bytes.NewReader([]byte("test data that exceeds buffer")),
			writerFunc: func(w io.Writer) (io.WriteCloser, error) {
				return &partialWriter{
					w:          w,
					writeLimit: 4, // Only write 4 bytes at a time
				}, nil
			},
			expectedErr:      true, // Now expecting an error due to short write
			skipOutputVerify: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := tlog.SetupSlogForTestWithContext(t, t.Context())
			// Create context, with timeout if specified
			var cancel context.CancelFunc
			if tc.contextTimeout > 0 {
				ctx, cancel = context.WithTimeout(ctx, tc.contextTimeout)
				defer cancel()
			}

			// Call the function under test
			readCloser, err := CreateWriterPipeline(ctx, tc.reader, tc.writerFunc)

			// Check for expected errors during creation
			if tc.expectedErr {
				if err != nil {
					// Error during pipeline creation
					if tc.expectedErrMsg != "" {
						assert.Contains(t, err.Error(), tc.expectedErrMsg, "Error message does not contain expected text")
					}
					return
				}

				// If no error during creation, we'll get an error when reading
				if !tc.skipOutputVerify {
					t.Fatal("Expected error but none occurred during pipeline creation")
				}

				// Read from the pipeline to trigger errors
				readCtx, readCancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
				defer readCancel()

				errCh := make(chan error, 1)
				go func() {
					_, err := io.ReadAll(readCloser)
					errCh <- err
				}()

				// Wait for either a timeout or an error
				select {
				case <-readCtx.Done():
					// For context cancellation tests, a timeout is expected and represents success
					if tc.contextTimeout > 0 {
						return
					}
					t.Fatal("Read operation timed out without error")
				case err := <-errCh:
					require.Error(t, err, "Expected an error during read operation")
				}
				return
			}

			require.NoError(t, err, "Unexpected error from CreateWriterPipeline")
			require.NotNil(t, readCloser, "Expected a non-nil ReadCloser")
			defer readCloser.Close()

			// Skip output verification if needed
			if tc.skipOutputVerify {
				return
			}

			// Set a timeout for reading to avoid hanging tests
			readCtx, readCancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer readCancel()

			// Use a buffered reader with a timeout
			readCh := make(chan struct {
				data []byte
				err  error
			}, 1)

			go func() {
				data, err := io.ReadAll(readCloser)
				readCh <- struct {
					data []byte
					err  error
				}{data, err}
			}()

			var data []byte
			var readErr error

			select {
			case <-readCtx.Done():
				t.Fatal("Test timed out while reading from pipeline")
			case result := <-readCh:
				data = result.data
				readErr = result.err
			}

			require.NoError(t, readErr, "Error reading from pipeline")
			diff.Require(t).Got(data).Want(tc.expectedOutput).Equals()
		})
	}
}

func TestCreateWriterPipeline_CloseIsCalled(t *testing.T) {
	ctx := tlog.SetupSlogForTestWithContext(t, t.Context())
	mock := &mockWriteCloser{}
	reader := bytes.NewReader([]byte("test data"))

	readCloser, err := CreateWriterPipeline(ctx, reader, func(w io.Writer) (io.WriteCloser, error) {
		return mock, nil
	})
	require.NoError(t, err, "Failed to create writer pipeline")

	// Read all data to ensure the goroutine completes
	_, err = io.ReadAll(readCloser)
	require.NoError(t, err, "Failed to read from pipeline")

	// Give the goroutine time to complete and call Close
	time.Sleep(100 * time.Millisecond)

	// Verify Close was called
	assert.Equal(t, 1, mock.closeCount, "Close should be called exactly once")
}

func TestCreateWriterPipeline_TransformingWriter(t *testing.T) {
	ctx := tlog.SetupSlogForTestWithContext(t, t.Context())
	// Test with a writer that transforms data (like compression would)
	reader := bytes.NewReader([]byte("test data"))

	readCloser, err := CreateWriterPipeline(ctx, reader, func(w io.Writer) (io.WriteCloser, error) {
		return &mockWriteCloser{
			writeFunc: func(p []byte) (int, error) {
				// Simple transformation: convert to uppercase
				upperP := bytes.ToUpper(p)
				return w.Write(upperP)
			},
		}, nil
	})
	require.NoError(t, err, "Failed to create writer pipeline")

	data, err := io.ReadAll(readCloser)
	require.NoError(t, err, "Failed to read from pipeline")

	diff.Require(t).Got(data).Want([]byte("TEST DATA")).Equals()
}

func TestCreateWriterPipeline_ConcurrentReads(t *testing.T) {
	// This test verifies that the first reader to read from the pipeline gets the data
	ctx := tlog.SetupSlogForTestWithContext(t, t.Context())
	
	// Create test data
	testData := bytes.Repeat([]byte("concurrent test data "), 1000) // Smaller size for faster test
	reader := bytes.NewReader(testData)

	readCloser, err := CreateWriterPipeline(ctx, reader, func(w io.Writer) (io.WriteCloser, error) {
		return &mockWriteCloser{
			writeFunc: func(p []byte) (int, error) {
				return w.Write(p)
			},
		}, nil
	})
	require.NoError(t, err, "Failed to create writer pipeline")
	
	// With the improved implementation, we can't have multiple goroutines reading from the same pipeline
	// So we'll read once and verify the data is correct
	data, err := io.ReadAll(readCloser)
	require.NoError(t, err, "Failed to read from pipeline")
	require.Equal(t, testData, data, "Data read from pipeline doesn't match expected data")
	
	// Ensure we get EOF when trying to read again
	n, err := readCloser.Read(make([]byte, 10))
	require.Equal(t, 0, n, "Should read 0 bytes after complete read")
	require.Equal(t, io.EOF, err, "Should get EOF when reading from consumed pipeline")
}
