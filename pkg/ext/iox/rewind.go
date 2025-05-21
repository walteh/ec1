// Package iox provides IO utilities.
package iox

import (
	"context"
	"io"
	"sync"

	"gitlab.com/tozd/go/errors"
)

var (
	_ io.ReadSeeker = (*LazyReader)(nil)
	_ io.ReaderAt   = (*LazyReader)(nil)
	_ io.Reader     = (*LazyReader)(nil)
	_ io.Closer     = (*LazyReader)(nil)
	_ io.Seeker     = (*LazyReader)(nil)
)

// LazyReader provides on-demand access to a reader's data with caching to
// support seeking. It only reads data that is explicitly requested.
type LazyReader struct {
	reader    io.Reader          // Original reader
	tempFile  io.ReadWriteSeeker // Temporary file for caching
	mutex     sync.Mutex         // Protects concurrent access
	readPos   int64              // Current read position in the stream
	cachedEnd int64              // Position of last byte cached in tempFile
	eof       bool               // Whether we've reached EOF on the reader
	closed    bool               // Whether this reader has been closed
}

// NewLazyReader creates a reader that supports seeking by caching
// data on-demand in a temporary file.
func NewLazyReader(r io.Reader, cache io.ReadWriteSeeker) (*LazyReader, error) {
	// tempFile, err := os.CreateTemp("", "lazy-reader-*")
	// if err != nil {
	// 	return nil, errors.Errorf("creating temporary file: %w", err)
	// }

	return &LazyReader{
		reader:    r,
		tempFile:  cache,
		readPos:   0,
		cachedEnd: 0,
	}, nil
}

// ensureDataCached ensures that data is cached up to at least the specified position.
// Returns true if successful, false if EOF was reached before the target position.
func (l *LazyReader) ensureDataCached(targetPos int64) (bool, error) {
	if l.closed {
		return false, errors.New("reader is closed")
	}

	// If data already cached, nothing to do
	if targetPos <= l.cachedEnd {
		return true, nil
	}

	// If we've already reached EOF, we can't cache more data
	if l.eof {
		return false, nil
	}

	// Position the file at the end of cached data for appending
	_, err := l.tempFile.Seek(l.cachedEnd, io.SeekStart)
	if err != nil {
		return false, errors.Errorf("seeking to end of cache: %w", err)
	}

	// Calculate how much more data we need
	toRead := targetPos - l.cachedEnd

	// Create a limited reader to read only what we need
	limited := io.LimitReader(l.reader, toRead)

	// Copy data to the temp file
	n, err := io.Copy(l.tempFile, limited)
	l.cachedEnd += n

	if err != nil && err != io.EOF {
		return false, errors.Errorf("caching data: %w", err)
	}

	// Check if we reached EOF
	if n < toRead || err == io.EOF {
		l.eof = true
		return false, nil
	}

	return true, nil
}

// Read implements io.Reader.
func (l *LazyReader) Read(p []byte) (n int, err error) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	// If we've reached the end of cached data and not EOF,
	// try to read more directly from the source reader
	if l.readPos == l.cachedEnd && !l.eof {
		// Try to read directly from the source, bypassing the cache
		// for efficiency when doing sequential reads
		n, err := l.reader.Read(p)
		if n > 0 {
			// Cache what we read
			_, seekErr := l.tempFile.Seek(l.cachedEnd, io.SeekStart)
			if seekErr != nil {
				return n, errors.Errorf("seeking in cache: %w", seekErr)
			}

			_, writeErr := l.tempFile.Write(p[:n])
			if writeErr != nil {
				return n, errors.Errorf("caching data: %w", writeErr)
			}

			l.readPos += int64(n)
			l.cachedEnd += int64(n)

			if err == io.EOF {
				l.eof = true
			}

			return n, err
		}

		// If direct read returned nothing or error, fall through to cached approach
		if err == io.EOF {
			l.eof = true
		}
	}

	// Position the file for reading
	_, err = l.tempFile.Seek(l.readPos, io.SeekStart)
	if err != nil {
		return 0, errors.Errorf("seeking in cache: %w", err)
	}

	// Read data from cache
	n, err = l.tempFile.Read(p)
	l.readPos += int64(n)

	// If we've reached the end of cached data and we need more,
	// try to get more data
	if n < len(p) && l.readPos == l.cachedEnd && !l.eof {
		// Try to get more data from the source
		m, err := l.reader.Read(p[n:])
		if m > 0 {
			// Cache what we read
			_, seekErr := l.tempFile.Seek(l.cachedEnd, io.SeekStart)
			if seekErr != nil {
				return n + m, errors.Errorf("seeking in cache: %w", seekErr)
			}

			_, writeErr := l.tempFile.Write(p[n : n+m])
			if writeErr != nil {
				return n + m, errors.Errorf("caching data: %w", writeErr)
			}

			l.readPos += int64(m)
			l.cachedEnd += int64(m)
			n += m
		}

		if err == io.EOF {
			l.eof = true
		}

		return n, err
	}

	// If we've reached the end of cached data and it's EOF, return EOF
	if n < len(p) && l.readPos == l.cachedEnd && l.eof {
		return n, io.EOF
	}

	return n, nil
}

// ReadAt implements io.ReaderAt.
func (l *LazyReader) ReadAt(p []byte, off int64) (n int, err error) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	// Ensure the data is cached up to at least off+len(p)
	needToCacheUntil := off + int64(len(p))
	foundAll, err := l.ensureDataCached(needToCacheUntil)
	if err != nil {
		return 0, err
	}

	// Position the file for reading
	_, err = l.tempFile.Seek(off, io.SeekStart)
	if err != nil {
		return 0, errors.Errorf("seeking in cache: %w", err)
	}

	// Read the data from cache
	n, err = l.tempFile.Read(p)

	// If we couldn't cache all requested data, return EOF if appropriate
	if !foundAll && off+int64(n) >= l.cachedEnd {
		return n, io.EOF
	}

	return n, err
}

// Seek implements io.Seeker.
func (l *LazyReader) Seek(offset int64, whence int) (int64, error) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	var abs int64
	switch whence {
	case io.SeekStart:
		abs = offset
	case io.SeekCurrent:
		abs = l.readPos + offset
	case io.SeekEnd:
		// If we haven't reached EOF yet, we need to read to the end
		if !l.eof {
			if _, err := l.ensureDataCached(1 << 60); err != nil { // Use very large value to indicate "all"
				return 0, err
			}
		}
		abs = l.cachedEnd + offset
	default:
		return 0, errors.New("invalid whence value")
	}

	if abs < 0 {
		return 0, errors.New("negative seek position")
	}

	// Update read position
	l.readPos = abs
	return abs, nil
}

// Close implements io.Closer.
func (l *LazyReader) Close() error {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	if l.closed {
		return nil
	}
	l.closed = true

	if rc, ok := l.tempFile.(io.ReadCloser); ok {
		if err := rc.Close(); err != nil {
			return errors.Errorf("closing temporary file: %w", err)
		}
	}

	// Close the file

	// // Delete the temporary file
	// if err := os.Remove(l.tempFile.Name()); err != nil {
	// 	return errors.Errorf("removing temporary file: %w", err)
	// }

	return nil
}

// Size returns the size of the data, reading to the end if necessary.
func (l *LazyReader) Size() (int64, error) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	// If we've already reached EOF, return cachedEnd
	if l.eof {
		return l.cachedEnd, nil
	}

	// Otherwise, read to the end
	if _, err := l.ensureDataCached(1 << 60); err != nil { // Use very large value to indicate "all"
		return 0, err
	}

	return l.cachedEnd, nil
}

// ReadWithContext implements context-aware reading.
func (l *LazyReader) ReadWithContext(ctx context.Context, p []byte) (int, error) {
	if ctx.Err() != nil {
		return 0, ctx.Err()
	}

	done := make(chan struct {
		n   int
		err error
	}, 1)

	go func() {
		n, err := l.Read(p)
		done <- struct {
			n   int
			err error
		}{n, err}
	}()

	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	case result := <-done:
		return result.n, result.err
	}
}
