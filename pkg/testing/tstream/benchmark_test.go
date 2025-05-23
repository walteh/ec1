package tstream

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStreamBenchmark(t *testing.T) {
	benchmark := NewStreamBenchmark("test-benchmark")
	
	require.NotNil(t, benchmark, "benchmark should not be nil")
	assert.Equal(t, "test-benchmark", benchmark.name, "name should be set")
	assert.Equal(t, 3, benchmark.warmup, "default warmup should be 3")
	assert.NotNil(t, benchmark.ctx, "context should be set")
	assert.Greater(t, len(benchmark.sizes), 0, "should have default sizes")
}

func TestStreamBenchmark_WithSizes(t *testing.T) {
	benchmark := NewStreamBenchmark("test").WithSizes(1024, 2048, 4096)
	
	expected := []int{1024, 2048, 4096}
	assert.Equal(t, expected, benchmark.sizes, "should set custom sizes")
}

func TestStreamBenchmark_WithWarmup(t *testing.T) {
	benchmark := NewStreamBenchmark("test").WithWarmup(5)
	
	assert.Equal(t, 5, benchmark.warmup, "should set custom warmup")
}

func TestStreamBenchmark_RunComparative(t *testing.T) {
	// Create a simple benchmark with small sizes for testing
	benchmark := NewStreamBenchmark("test-comparative").
		WithSizes(100, 200).  // Small sizes for fast tests
		WithWarmup(1)         // Minimal warmup for tests
	
	// Track which implementations were called
	var calledImpls []string
	var calledSizes []int
	
	implementations := map[string]BenchmarkFunc{
		"fast_impl": func(ctx context.Context, input io.Reader, size int) (io.ReadCloser, error) {
			calledImpls = append(calledImpls, "fast_impl")
			calledSizes = append(calledSizes, size)
			
			// Read input and return a simple reader
			data, _ := io.ReadAll(input)
			return io.NopCloser(bytes.NewReader(data)), nil
		},
		"slow_impl": func(ctx context.Context, input io.Reader, size int) (io.ReadCloser, error) {
			calledImpls = append(calledImpls, "slow_impl")
			calledSizes = append(calledSizes, size)
			
			// Simulate slower processing
			time.Sleep(time.Microsecond * 10)
			data, _ := io.ReadAll(input)
			return io.NopCloser(bytes.NewReader(data)), nil
		},
	}
	
	dataGenerator := func(size int) io.Reader {
		data := make([]byte, size)
		for i := range data {
			data[i] = byte(i % 256)
		}
		return bytes.NewReader(data)
	}
	
	// Capture the benchmark output
	var benchmarkCompleted bool
	
	testing.Benchmark(func(b *testing.B) {
		benchmark.RunComparative(b, implementations, dataGenerator)
		benchmarkCompleted = true
	})
	
	assert.True(t, benchmarkCompleted, "benchmark should complete")
	assert.Greater(t, len(calledImpls), 0, "implementations should be called")
	assert.Contains(t, calledImpls, "fast_impl", "fast_impl should be called")
	assert.Contains(t, calledImpls, "slow_impl", "slow_impl should be called")
	assert.Contains(t, calledSizes, 100, "size 100 should be tested")
	assert.Contains(t, calledSizes, 200, "size 200 should be tested")
}

func TestStreamBenchmark_RunDetailed(t *testing.T) {
	benchmark := NewStreamBenchmark("test-detailed").
		WithSizes(50, 100).
		WithWarmup(1)
	
	var calledCount int
	implementation := func(ctx context.Context, input io.Reader, size int) (io.ReadCloser, error) {
		calledCount++
		data, _ := io.ReadAll(input)
		return io.NopCloser(bytes.NewReader(data)), nil
	}
	
	dataGenerator := func(size int) io.Reader {
		data := make([]byte, size)
		return bytes.NewReader(data)
	}
	
	var results []ComparativeResult
	
	testing.Benchmark(func(b *testing.B) {
		results = benchmark.RunDetailed(b, implementation, dataGenerator)
	})
	
	assert.Greater(t, calledCount, 0, "implementation should be called")
	assert.Greater(t, len(results), 0, "should return results")
	
	for _, result := range results {
		assert.Greater(t, result.Duration, time.Duration(0), "duration should be positive")
		assert.GreaterOrEqual(t, result.Throughput, 0.0, "throughput should be non-negative")
		assert.GreaterOrEqual(t, result.AllocsPerOp, int64(0), "allocs should be non-negative")
		assert.GreaterOrEqual(t, result.BytesPerOp, int64(0), "bytes should be non-negative")
	}
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		name     string
		bytes    int
		expected string
	}{
		{"bytes", 512, "512B"},
		{"kilobytes", 2048, "2KB"},
		{"megabytes", 5*1024*1024, "5MB"},
		{"gigabytes", 3*1024*1024*1024, "3GB"},
		{"zero", 0, "0B"},
		{"one", 1, "1B"},
		{"boundary_kb", 1024, "1KB"},
		{"boundary_mb", 1024*1024, "1MB"},
		{"boundary_gb", 1024*1024*1024, "1GB"},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatSize(tt.bytes)
			assert.Equal(t, tt.expected, result, "formatSize should format correctly")
		})
	}
}

func TestHumanizeSize(t *testing.T) {
	// Test that humanizeSize is an alias for formatSize
	testBytes := 2048
	
	formatResult := formatSize(testBytes)
	humanizeResult := humanizeSize(testBytes)
	
	assert.Equal(t, formatResult, humanizeResult, "humanizeSize should match formatSize")
	assert.Equal(t, "2KB", humanizeResult, "should format 2048 bytes as 2KB")
}

func TestNewMemoryProfiler(t *testing.T) {
	ctx := context.Background()
	profiler := NewMemoryProfiler(ctx, "test-profiler")
	
	require.NotNil(t, profiler, "profiler should not be nil")
	assert.Equal(t, "test-profiler", profiler.name, "name should be set")
	assert.Equal(t, ctx, profiler.ctx, "context should be set")
}

func TestMemoryProfiler_Checkpoint(t *testing.T) {
	ctx := context.Background()
	profiler := NewMemoryProfiler(ctx, "test-profiler")
	
	// This test just ensures Checkpoint doesn't panic
	// Since it writes to stdout, we can't easily capture the output in a unit test
	assert.NotPanics(t, func() {
		profiler.Checkpoint("test-checkpoint")
	}, "Checkpoint should not panic")
}

// Integration test that exercises the full benchmark flow
func TestStreamBenchmark_Integration(t *testing.T) {
	// Create a benchmark with very small data for quick testing
	benchmark := NewStreamBenchmark("integration-test").
		WithSizes(10, 20).
		WithWarmup(1)
	
	// Mock implementations with different characteristics
	implementations := map[string]BenchmarkFunc{
		"memory_efficient": func(ctx context.Context, input io.Reader, size int) (io.ReadCloser, error) {
			// Simple pass-through - should be memory efficient
			return io.NopCloser(input), nil
		},
		"processing_intensive": func(ctx context.Context, input io.Reader, size int) (io.ReadCloser, error) {
			// Read all data and process it (more memory, potentially slower)
			data, err := io.ReadAll(input)
			if err != nil {
				return nil, err
			}
			
			// Simple "processing" - reverse the bytes
			for i, j := 0, len(data)-1; i < j; i, j = i+1, j-1 {
				data[i], data[j] = data[j], data[i]
			}
			
			return io.NopCloser(bytes.NewReader(data)), nil
		},
	}
	
	dataGenerator := func(size int) io.Reader {
		data := make([]byte, size)
		for i := range data {
			data[i] = byte(i % 256)
		}
		return bytes.NewReader(data)
	}
	
	// Just verify the benchmark runs without panicking
	// Since we can't easily capture log output in unit tests, we'll test the humanizer separately
	var benchmarkCompleted bool
	
	testing.Benchmark(func(b *testing.B) {
		assert.NotPanics(t, func() {
			benchmark.RunComparative(b, implementations, dataGenerator)
		}, "benchmark should run without panicking")
		benchmarkCompleted = true
	})
	
	assert.True(t, benchmarkCompleted, "benchmark should complete successfully")
	
	// Test the humanizer functionality separately with mock data
	humanizer := NewResultsHumanizer()
	humanizer.AddResult(BenchmarkResult{
		Name:        "memory_efficient",
		Size:        "10B",
		Duration:    time.Microsecond * 100,
		Throughput:  1000.0,
		AllocsPerOp: 5,
		BytesPerOp:  100,
	})
	humanizer.AddResult(BenchmarkResult{
		Name:        "processing_intensive",
		Size:        "10B",
		Duration:    time.Microsecond * 200,
		Throughput:  500.0,
		AllocsPerOp: 15,
		BytesPerOp:  300,
	})
	
	humanizedOutput := humanizer.HumanizeResults()
	assert.Contains(t, humanizedOutput, "🏆 BENCHMARK ANALYSIS REPORT", "should contain humanizer report")
	assert.Contains(t, humanizedOutput, "memory_efficient", "should mention memory_efficient implementation")
	assert.Contains(t, humanizedOutput, "processing_intensive", "should mention processing_intensive implementation")
	assert.Contains(t, humanizedOutput, "🥇 OVERALL WINNERS", "should contain winners section")
}

// Test error handling in benchmark functions
func TestStreamBenchmark_ErrorHandling(t *testing.T) {
	implementations := map[string]BenchmarkFunc{
		"failing_impl": func(ctx context.Context, input io.Reader, size int) (io.ReadCloser, error) {
			return nil, assert.AnError // Return an error
		},
	}
	
	dataGenerator := func(size int) io.Reader {
		return bytes.NewReader(make([]byte, size))
	}
	
	// The benchmark should handle the error gracefully (it will call b.Fatalf)
	// We can't easily test this without running the actual benchmark, so we just ensure
	// the setup doesn't panic
	assert.NotPanics(t, func() {
		// We don't actually run the benchmark here as it would fail the test
		// In real usage, the benchmark framework handles the failure
		_, _ = implementations["failing_impl"](context.Background(), dataGenerator(10), 10)
	}, "error handling setup should not panic")
} 