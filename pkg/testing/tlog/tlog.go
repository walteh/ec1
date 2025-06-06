package tlog

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	slogctx "github.com/veqryn/slog-context"

	"github.com/walteh/ec1/pkg/host"
	"github.com/walteh/ec1/pkg/logging"
	"github.com/walteh/ec1/pkg/logging/logrusshim"
)

func init() {
	logrusshim.ForwardLogrusToSlogGlobally()
}

func SetupSlogForTestWithContext(t testing.TB, ctx context.Context) context.Context {
	var simpctx context.Context

	existing := slogctx.FromCtx(ctx)
	if existing != nil {
		simpctx = ctx
	} else {
		simpctx = logging.SetupSlogSimple(ctx)
	}

	cached, err := host.CacheDirPrefix()
	require.NoError(t, err)
	logging.RegisterRedactedLogValue(ctx, os.TempDir()+"/", "[os-tmp-dir]")
	logging.RegisterRedactedLogValue(ctx, cached, "[vm-cache-dir]")
	logging.RegisterRedactedLogValue(ctx, filepath.Dir(t.TempDir()), "[test-tmp-dir]") // higher priority than os-tmp-dir

	return simpctx
}

func SetupSlogForTest(t testing.TB) context.Context {
	return SetupSlogForTestWithContext(t, t.Context())
}

func TeeToDownloadsFolder(rdr io.Reader, filename string) (io.Reader, io.Closer) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	fle, err := os.Create(filepath.Join(homeDir, "Downloads", fmt.Sprintf("golang-test.%d.%s", time.Now().Unix(), filename)))
	if err != nil {
		panic(err)
	}

	return io.TeeReader(rdr, fle), fle
}

type BCompare struct {
	A io.Reader
	B io.Reader
}

func NewBComare(t testing.TB, a, b io.Reader) (*BCompare, io.ReadCloser, io.ReadCloser) {
	bwA := bytes.NewBuffer(nil)
	bwB := bytes.NewBuffer(nil)
	trA := io.TeeReader(a, bwA)
	trB := io.TeeReader(b, bwB)
	return &BCompare{
		A: bwA,
		B: bwB,
	}, io.NopCloser(trA), io.NopCloser(trB)
}

func (b *BCompare) TeeA(rdr io.Reader) io.Reader {
	bw := bytes.NewBuffer(nil)
	tr := io.TeeReader(rdr, bw)
	b.A = tr
	return tr
}

func (b *BCompare) TeeB(rdr io.Reader) io.Reader {
	bw := bytes.NewBuffer(nil)
	tr := io.TeeReader(rdr, bw)
	b.B = tr
	return tr
}

func (b *BCompare) Close() error {
	return nil
}

func (b *BCompare) Compare(t testing.TB) error {

	allA, err := io.ReadAll(b.A)
	if err != nil {
		return err
	}

	allB, err := io.ReadAll(b.B)
	if err != nil {
		return err
	}

	require.Equal(t, allA, allB)

	return nil
}

// FileProxyConfig configures the file proxy behavior
type FileProxyConfig struct {
	// FilePath is the path to the file to tail
	FilePath string
	// Output is where to write the content (defaults to os.Stdout)
	Output io.Writer
	// PollInterval is how often to check for new content (default: 100ms)
	PollInterval time.Duration
	// LogPrefix is an optional prefix for log lines
	LogPrefix string
	// FollowRename whether to follow file renames/rotations (default: true)
	FollowRename bool
}

// FileProxy represents an active file proxy operation
type FileProxy struct {
	config FileProxyConfig
	file   *os.File
	offset int64
	done   chan struct{}
}

// NewFileProxy creates a new file proxy with the given configuration
func NewFileProxy(config FileProxyConfig) *FileProxy {
	if config.Output == nil {
		config.Output = os.Stdout
	}
	if config.PollInterval == 0 {
		config.PollInterval = 100 * time.Millisecond
	}
	if config.LogPrefix == "" {
		config.LogPrefix = "[file-proxy] "
	}

	return &FileProxy{
		config: config,
		done:   make(chan struct{}),
	}
}

// ProxyFileToStdout is a convenience function that creates and starts a file proxy to stdout
func ProxyFileToStdout(ctx context.Context, filePath string) (*FileProxy, error) {
	proxy := NewFileProxy(FileProxyConfig{
		FilePath: filePath,
		Output:   os.Stdout,
	})

	err := proxy.Start(ctx)
	if err != nil {
		return nil, err
	}

	return proxy, nil
}

// Start begins proxying the file content to the configured output
func (fp *FileProxy) Start(ctx context.Context) error {
	logger := slogctx.FromCtx(ctx)
	if logger == nil {
		logger = slog.Default()
	}

	// Try to open the file initially
	var err error
	fp.file, err = os.Open(fp.config.FilePath)
	if err != nil {
		// If file doesn't exist yet, we'll wait for it
		if !os.IsNotExist(err) {
			return fmt.Errorf("opening file %s: %w", fp.config.FilePath, err)
		}
		logger.InfoContext(ctx, "Waiting for file to be created", "file", fp.config.FilePath)
	} else {
		// Seek to end of file to only read new content
		fp.offset, err = fp.file.Seek(0, io.SeekEnd)
		if err != nil {
			fp.file.Close()
			return fmt.Errorf("seeking to end of file: %w", err)
		}
		logger.InfoContext(ctx, "Started proxying file", "file", fp.config.FilePath, "offset", fp.offset)
	}

	go fp.proxyLoop(ctx)
	return nil
}

// Stop stops the file proxy
func (fp *FileProxy) Stop() {
	close(fp.done)
	if fp.file != nil {
		fp.file.Close()
	}
}

// proxyLoop is the main loop that reads and proxies file content
func (fp *FileProxy) proxyLoop(ctx context.Context) {
	logger := slogctx.FromCtx(ctx)
	if logger == nil {
		logger = slog.Default()
	}

	ticker := time.NewTicker(fp.config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.InfoContext(ctx, "File proxy stopped due to context cancellation")
			return
		case <-fp.done:
			logger.InfoContext(ctx, "File proxy stopped")
			return
		case <-ticker.C:
			if err := fp.readAndProxy(ctx); err != nil {
				logger.WarnContext(ctx, "Error reading file", "error", err)
			}
		}
	}
}

// readAndProxy reads new content from the file and writes it to output
func (fp *FileProxy) readAndProxy(ctx context.Context) error {
	logger := slogctx.FromCtx(ctx)
	if logger == nil {
		logger = slog.Default()
	}

	// If file isn't open yet, try to open it
	if fp.file == nil {
		var err error
		fp.file, err = os.Open(fp.config.FilePath)
		if err != nil {
			if os.IsNotExist(err) {
				// File still doesn't exist, keep waiting
				return nil
			}
			return fmt.Errorf("opening file: %w", err)
		}
		logger.InfoContext(ctx, "File created, started proxying", "file", fp.config.FilePath)
		fp.offset = 0
	}

	// Check if file was truncated or rotated
	stat, err := fp.file.Stat()
	if err != nil {
		return fmt.Errorf("getting file stats: %w", err)
	}

	// If file is smaller than our offset, it was likely truncated
	if stat.Size() < fp.offset {
		logger.InfoContext(ctx, "File appears to have been truncated, resetting to beginning")
		fp.offset = 0
		fp.file.Seek(0, io.SeekStart)
	}

	// If follow rename is enabled, check if the file was rotated
	if fp.config.FollowRename {
		if err := fp.checkFileRotation(ctx); err != nil {
			logger.WarnContext(ctx, "Error checking file rotation", "error", err)
		}
	}

	// Read new content
	currentOffset, err := fp.file.Seek(0, io.SeekCurrent)
	if err != nil {
		return fmt.Errorf("getting current offset: %w", err)
	}

	if currentOffset != fp.offset {
		fp.file.Seek(fp.offset, io.SeekStart)
	}

	buffer := make([]byte, 4096)
	for {
		n, err := fp.file.Read(buffer)
		if n > 0 {
			// Write the content with optional prefix
			content := buffer[:n]
			if fp.config.LogPrefix != "" {
				// Add prefix to each line
				lines := bytes.Split(content, []byte("\n"))
				for i, line := range lines {
					if len(line) > 0 || (i < len(lines)-1) { // Don't prefix empty last line
						fp.config.Output.Write([]byte(fp.config.LogPrefix))
						fp.config.Output.Write(line)
						if i < len(lines)-1 {
							fp.config.Output.Write([]byte("\n"))
						}
					}
				}
			} else {
				fp.config.Output.Write(content)
			}
			fp.offset += int64(n)
		}
		if err != nil {
			if err == io.EOF {
				break // No more data available right now
			}
			return fmt.Errorf("reading file: %w", err)
		}
	}

	return nil
}

// checkFileRotation checks if the file was rotated and reopens if necessary
func (fp *FileProxy) checkFileRotation(ctx context.Context) error {
	logger := slogctx.FromCtx(ctx)
	if logger == nil {
		logger = slog.Default()
	}

	// Get current file stat
	currentStat, err := fp.file.Stat()
	if err != nil {
		return fmt.Errorf("getting current file stat: %w", err)
	}

	// Get stat of file by name
	newStat, err := os.Stat(fp.config.FilePath)
	if err != nil {
		if os.IsNotExist(err) {
			// File was removed, close current handle and wait for new file
			fp.file.Close()
			fp.file = nil
			fp.offset = 0
			logger.InfoContext(ctx, "File was removed, waiting for new file")
			return nil
		}
		return fmt.Errorf("getting new file stat: %w", err)
	}

	// Compare inodes (on Unix systems) or other attributes
	if !os.SameFile(currentStat, newStat) {
		logger.InfoContext(ctx, "File was rotated, reopening")
		fp.file.Close()
		fp.file, err = os.Open(fp.config.FilePath)
		if err != nil {
			return fmt.Errorf("reopening rotated file: %w", err)
		}
		fp.offset = 0
	}

	return nil
}

// ProxyFileToStdoutForTest is a test helper that proxies a file to stdout with test context
func ProxyFileToStdoutForTest(t testing.TB, filePath string) *FileProxy {
	ctx := SetupSlogForTest(t)

	proxy, err := ProxyFileToStdout(ctx, filePath)
	require.NoError(t, err, "Failed to start file proxy")

	// Clean up when test ends
	t.Cleanup(func() {
		proxy.Stop()
	})

	return proxy
}
