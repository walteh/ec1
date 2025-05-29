package binembed

import (
	"bytes"
	"io"
	"log/slog"
	"sync"
	"time"

	"github.com/mholt/archives"
	"gitlab.com/tozd/go/errors"
)

// binaryEntry holds the binary data and decompression state
type binaryEntry struct {
	compressed   []byte
	decompressor archives.Compression
	isCompressed bool // New field to track if data is compressed

	// Lazy decompression with sync.Once for thread safety
	decompressOnce sync.Once
	decompressed   []byte
	decompressErr  error
}

// Global registry with a single RWMutex for better performance
var (
	registry      = make(map[string]*binaryEntry)
	registryMutex sync.RWMutex
)

// RegisterXZ registers a compressed binary with XZ compression.
// This is typically called during package init() functions.
func RegisterXZ(checkSum string, binary []byte) {
	registryMutex.Lock()
	defer registryMutex.Unlock()

	registry[checkSum] = &binaryEntry{
		compressed:   binary,
		decompressor: &archives.Xz{},
		isCompressed: true,
	}
}

// RegisterRaw registers an uncompressed binary.
// This is useful for config files or other data that doesn't need decompression.
func RegisterRaw(checkSum string, binary []byte) {
	registryMutex.Lock()
	defer registryMutex.Unlock()

	registry[checkSum] = &binaryEntry{
		compressed:   binary,
		decompressor: nil,
		isCompressed: false,
		// For raw data, we can set decompressed immediately
		decompressed: binary,
	}
}

// GetDecompressed returns a reader for the decompressed binary data.
// The decompression is performed lazily and cached for subsequent calls.
// This function is safe for concurrent use and handles both compressed and raw data.
func GetDecompressed(checkSum string) (io.Reader, error) {
	// Fast path: read-only access to find the entry
	registryMutex.RLock()
	entry, exists := registry[checkSum]
	registryMutex.RUnlock()
	
	if !exists {
		return nil, errors.Errorf("binary not found: %s", checkSum)
	}
	
	// For uncompressed data, return immediately
	if !entry.isCompressed {
		return bytes.NewReader(entry.decompressed), nil
	}
	
	// Lazy decompression with sync.Once ensures thread safety
	// and prevents duplicate decompression work
	entry.decompressOnce.Do(func() {
		entry.decompressed, entry.decompressErr = decompressBinary(entry.compressed, entry.decompressor)
	})
	
	if entry.decompressErr != nil {
		return nil, errors.Errorf("decompressing binary %s: %w", checkSum, entry.decompressErr)
	}
	
	return bytes.NewReader(entry.decompressed), nil
}

// decompressBinary performs the actual decompression work
func decompressBinary(compressed []byte, decompressor archives.Compression) ([]byte, error) {
	reader, err := decompressor.OpenReader(bytes.NewReader(compressed))
	if err != nil {
		return nil, errors.Errorf("opening decompressor: %w", err)
	}

	decompressed, err := io.ReadAll(reader)
	if err != nil {
		return nil, errors.Errorf("reading decompressed data: %w", err)
	}

	return decompressed, nil
}

// MustGetDecompressed is like GetDecompressed but panics on error.
// Use this when you're certain the binary exists and decompression will succeed.
func MustGetDecompressed(checkSum string) io.Reader {
	reader, err := GetDecompressed(checkSum)
	if err != nil {
		panic(err)
	}
	return reader
}

// PreloadAsync starts decompression of the specified binary in a background goroutine.
// This is useful for init-time preloading to reduce first-access latency.
// The function returns immediately and decompression happens asynchronously.
func PreloadAsync(checkSum string) {
	go func() {
		startTime := time.Now()
		_, _ = GetDecompressed(checkSum) // Ignore errors in background preload
		slog.Info("preloaded binary", "checksum", checkSum, "duration", time.Since(startTime))
	}()
}

// PreloadSync synchronously decompresses the specified binary and waits for completion.
// This is useful when you need to ensure a binary is ready before proceeding.
func PreloadSync(checkSum string) error {
	_, err := GetDecompressed(checkSum)
	return err
}

// PreloadAllSync concurrently decompresses all registered binaries and waits for completion.
// This is useful during initialization when you want to ensure all binaries are ready
// before proceeding with the application startup. All decompression happens in parallel
// for maximum performance.
func PreloadAllSync() error {
	registryMutex.RLock()
	checksums := make([]string, 0, len(registry))
	for checksum := range registry {
		checksums = append(checksums, checksum)
	}
	registryMutex.RUnlock()

	if len(checksums) == 0 {
		return nil // Nothing to preload
	}

	// Use a channel to collect errors from goroutines
	errChan := make(chan error, len(checksums))

	// Start all decompression operations concurrently
	for _, checksum := range checksums {
		go func(cs string) {
			startTime := time.Now()
			_, err := GetDecompressed(cs)
			if err != nil {
				errChan <- errors.Errorf("preloading binary %s: %w", cs, err)
			} else {
				slog.Debug("preloaded binary", "checksum", cs, "duration", time.Since(startTime))
				errChan <- nil
			}
		}(checksum)
	}

	// Wait for all goroutines to complete and collect any errors
	for i := 0; i < len(checksums); i++ {
		if err := <-errChan; err != nil {
			return err // Return first error encountered
		}
	}

	return nil
}

// PreloadAsyncAll starts decompression of all registered binaries in background goroutines.
// This is useful for warming up all binaries during application startup.
func PreloadAsyncAll() {
	registryMutex.RLock()
	checksums := make([]string, 0, len(registry))
	for checksum := range registry {
		checksums = append(checksums, checksum)
	}
	registryMutex.RUnlock()

	for _, checksum := range checksums {
		PreloadAsync(checksum)
	}
}

// GetRegisteredChecksums returns a slice of all registered binary checksums.
// This is useful for debugging or inventory purposes.
func GetRegisteredChecksums() []string {
	registryMutex.RLock()
	defer registryMutex.RUnlock()

	checksums := make([]string, 0, len(registry))
	for checksum := range registry {
		checksums = append(checksums, checksum)
	}
	return checksums
}

// IsRegistered checks if a binary with the given checksum is registered.
func IsRegistered(checkSum string) bool {
	registryMutex.RLock()
	defer registryMutex.RUnlock()

	_, exists := registry[checkSum]
	return exists
}
