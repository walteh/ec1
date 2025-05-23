package initramfs_test

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/walteh/ec1/pkg/initramfs"
	"github.com/walteh/ec1/pkg/testing/tlog"
	"github.com/walteh/ec1/pkg/testing/tstream"
)

// BenchmarkInitramfsProcessingPipeline tests the complete pipeline with timing
func BenchmarkInitramfsProcessingPipeline(b *testing.B) {
	// Create test data generator that returns actual decompressed CPIO data
	dataGenerator := func(size int) io.Reader {
		// Use the large CPIO test data from the existing function
		return openLargeCpio(b)
	}

	mockInitBinary := make([]byte, 1024*1024) // 1MB init binary
	for i := range mockInitBinary {
		mockInitBinary[i] = byte(i % 256)
	}

	// Define implementations to compare
	implementations := map[string]tstream.BenchmarkFunc{
		"stream_original": func(ctx context.Context, input io.Reader, size int) (io.ReadCloser, error) {
			timedInput := tstream.NewTimingReader(ctx, input, "stream-original-input")
			defer timedInput.Close()

			result := initramfs.StreamInject(ctx, timedInput, initramfs.NewExecHeader("init"), mockInitBinary)
			return tstream.NewTimingReader(ctx, result, "stream-original-output"), nil
		},

		"stream_optimized": func(ctx context.Context, input io.Reader, size int) (io.ReadCloser, error) {
			timedInput := tstream.NewTimingReader(ctx, input, "stream-optimized-input")
			defer timedInput.Close()

			result := initramfs.StreamInjectOptimized(ctx, timedInput, initramfs.NewExecHeader("init"), mockInitBinary)
			return tstream.NewTimingReader(ctx, result, "stream-optimized-output"), nil
		},

		"stream_hyper": func(ctx context.Context, input io.Reader, size int) (io.ReadCloser, error) {
			timedInput := tstream.NewTimingReader(ctx, input, "stream-hyper-input")
			defer timedInput.Close()

			result := initramfs.StreamInjectHyper(ctx, timedInput, initramfs.NewExecHeader("init"), mockInitBinary)
			return tstream.NewTimingReader(ctx, result, "stream-hyper-output"), nil
		},

		"fast_blazing": func(ctx context.Context, input io.Reader, size int) (io.ReadCloser, error) {
			timedInput := tstream.NewTimingReader(ctx, input, "fast-blazing-input")
			defer timedInput.Close()

			result, err := initramfs.FastInjectFileToCpioBlazingFast(ctx, timedInput, initramfs.NewExecHeader("init"), mockInitBinary)
			if err != nil {
				return nil, err
			}
			return tstream.NewTimingReader(ctx, result, "fast-blazing-output"), nil
		},
	}

	// Run comparative benchmark with humanizer analysis
	benchmark := tstream.NewStreamBenchmark("initramfs-processing").
		WithSizes(1024*1024, 10*1024*1024). // 1MB, 10MB (reduced for faster testing)
		WithWarmup(1) // Reduced warmup for faster testing

	benchmark.RunComparative(b, implementations, dataGenerator)
}

// BenchmarkInitramfsMemoryUsage tracks memory usage patterns
func BenchmarkInitramfsMemoryUsage(b *testing.B) {
	ctx := context.Background()
	profiler := tstream.NewMemoryProfiler(ctx, "initramfs-memory")

	mockInitBinary := make([]byte, 1024*1024) // 1MB
	profiler.Checkpoint("initial")

	for i := 0; i < b.N; i++ {
		cpioPath := openLargeCpio(b)
		defer cpioPath.Close()

		profiler.Checkpoint("before-processing")

		result, err := initramfs.FastInjectFileToCpioBlazingFast(ctx, cpioPath, initramfs.NewExecHeader("init"), mockInitBinary)
		require.NoError(b, err, "FastInjectFileToCpioBlazingFast should succeed")

		profiler.Checkpoint("after-processing")

		_, err = io.Copy(io.Discard, result)
		require.NoError(b, err, "reading result should succeed")
		result.Close()

		profiler.Checkpoint("after-consumption")
	}
}

// BenchmarkCompressionMethods compares different compression approaches
func BenchmarkCompressionMethods(b *testing.B) {
	ctx := context.Background()

	// Create uncompressed test data
	testData := make([]byte, 50*1024*1024) // 50MB
	for i := range testData {
		testData[i] = byte(i % 256)
	}

	b.Run("gzip_default", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			reader := tstream.NewTimingReader(ctx, bytes.NewReader(testData), "gzip-default")

			// Simulate default gzip compression timing
			// This would wrap with your actual compression pipeline
			_, err := io.Copy(io.Discard, reader)
			require.NoError(b, err, "copy should succeed")
			reader.Close()
		}
	})

	b.Run("gzip_fast", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			reader := tstream.NewTimingReader(ctx, bytes.NewReader(testData), "gzip-fast")

			// Simulate fast gzip compression timing
			_, err := io.Copy(io.Discard, reader)
			require.NoError(b, err, "copy should succeed")
			reader.Close()
		}
	})
}

// TestStreamTimingIntegration shows how to use timing in regular tests
func TestStreamTimingIntegration(t *testing.T) {
	ctx := tlog.SetupSlogForTestWithContext(t, context.Background())

	cpioPath := generateTestCpio(t, map[string][]byte{
		"init": []byte("#!/bin/sh\necho 'original_init'\n"),
	})
	defer cpioPath.Close()

	mockInitBinary := "#!/bin/sh\necho 'mock_init'\n"

	// Wrap the input stream with timing
	cpioPath.Seek(0, io.SeekStart)
	timedInput := tstream.NewTimingReader(ctx, cpioPath, "test-input")
	defer timedInput.Close()

	// Process with timing
	result := initramfs.StreamInjectHyper(ctx, timedInput, initramfs.NewExecHeader("init"), []byte(mockInitBinary))
	timedOutput := tstream.NewTimingReader(ctx, result, "test-output")
	defer timedOutput.Close()

	// Consume the output
	_, err := io.Copy(io.Discard, timedOutput)
	require.NoError(t, err, "stream processing should succeed")

	// Timing information will be logged automatically when readers are closed
}
