package tstream

import (
	"context"
	"fmt"
	"io"
	"runtime"
	"testing"
	"time"
)

// StreamBenchmark provides utilities for benchmarking stream operations
type StreamBenchmark struct {
	name     string
	sizes    []int
	warmup   int
	ctx      context.Context
}

func NewStreamBenchmark(name string) *StreamBenchmark {
	return &StreamBenchmark{
		name:   name,
		sizes:  []int{1024, 64 * 1024, 1024 * 1024, 10 * 1024 * 1024, 100 * 1024 * 1024},
		warmup: 3,
		ctx:    context.Background(),
	}
}

func (sb *StreamBenchmark) WithSizes(sizes ...int) *StreamBenchmark {
	sb.sizes = sizes
	return sb
}

func (sb *StreamBenchmark) WithWarmup(warmup int) *StreamBenchmark {
	sb.warmup = warmup
	return sb
}

// BenchmarkFunc represents a function that processes a stream
type BenchmarkFunc func(ctx context.Context, input io.Reader, size int) (io.ReadCloser, error)

// ComparativeResult holds results from comparing multiple implementations
type ComparativeResult struct {
	Name        string
	Size        int
	Duration    time.Duration
	Throughput  float64 // MB/s
	AllocsPerOp int64
	BytesPerOp  int64
}

// RunComparative runs multiple implementations against each other
func (sb *StreamBenchmark) RunComparative(b *testing.B, implementations map[string]BenchmarkFunc, dataGenerator func(size int) io.Reader) {
	humanizer := NewResultsHumanizer()
	
	for _, size := range sb.sizes {
		sizeName := formatSize(size)
		
		for implName, implFunc := range implementations {
			b.Run(fmt.Sprintf("%s/%s/%s", sb.name, implName, sizeName), func(b *testing.B) {
				// Warmup
				for i := 0; i < sb.warmup; i++ {
					input := dataGenerator(size)
					result, err := implFunc(sb.ctx, input, size)
					if err != nil {
						b.Fatalf("warmup failed: %v", err)
					}
					io.Copy(io.Discard, result)
					result.Close()
				}
				
				// Benchmark
				b.ResetTimer()
				b.SetBytes(int64(size))
				
				start := time.Now()
				var m1, m2 runtime.MemStats
				runtime.ReadMemStats(&m1)
				
				for i := 0; i < b.N; i++ {
					input := dataGenerator(size)
					result, err := implFunc(sb.ctx, input, size)
					if err != nil {
						b.Fatalf("implementation failed: %v", err)
					}
					
					_, err = io.Copy(io.Discard, result)
					if err != nil {
						b.Fatalf("reading result failed: %v", err)
					}
					result.Close()
				}
				
				duration := time.Since(start)
				runtime.ReadMemStats(&m2)
				
				// Calculate metrics
				avgDuration := duration / time.Duration(b.N)
				throughput := float64(size*b.N) / duration.Seconds() / 1024 / 1024
				allocsPerOp := int64(m2.Mallocs-m1.Mallocs) / int64(b.N)
				bytesPerOp := int64(m2.TotalAlloc-m1.TotalAlloc) / int64(b.N)
				
				// Add to humanizer
				humanizer.AddResult(BenchmarkResult{
					Name:        implName,
					Size:        sizeName,
					Duration:    avgDuration,
					Throughput:  throughput,
					AllocsPerOp: allocsPerOp,
					BytesPerOp:  bytesPerOp,
					Iterations:  b.N,
				})
			})
		}
	}
	
	// Print humanized analysis after all benchmarks complete
	analysis := humanizer.HumanizeResults()
	b.Logf("\n%s", analysis)
}

// RunDetailed provides detailed timing and allocation information
func (sb *StreamBenchmark) RunDetailed(b *testing.B, impl BenchmarkFunc, dataGenerator func(size int) io.Reader) []ComparativeResult {
	var results []ComparativeResult
	
	for _, size := range sb.sizes {
		sizeName := formatSize(size)
		
		b.Run(fmt.Sprintf("%s/detailed/%s", sb.name, sizeName), func(b *testing.B) {
			// Warmup
			for i := 0; i < sb.warmup; i++ {
				input := dataGenerator(size)
				result, err := impl(sb.ctx, input, size)
				if err != nil {
					b.Fatalf("warmup failed: %v", err)
				}
				io.Copy(io.Discard, result)
				result.Close()
			}
			
			// Force GC before measurement
			runtime.GC()
			
			// Detailed measurement
			var m1, m2 runtime.MemStats
			runtime.ReadMemStats(&m1)
			
			start := time.Now()
			
			for i := 0; i < b.N; i++ {
				input := dataGenerator(size)
				result, err := impl(sb.ctx, input, size)
				if err != nil {
					b.Fatalf("implementation failed: %v", err)
				}
				
				_, err = io.Copy(io.Discard, result)
				if err != nil {
					b.Fatalf("reading result failed: %v", err)
				}
				result.Close()
			}
			
			duration := time.Since(start)
			runtime.ReadMemStats(&m2)
			
			result := ComparativeResult{
				Name:        sizeName,
				Size:        size,
				Duration:    duration / time.Duration(b.N),
				Throughput:  float64(size) / (duration.Seconds() / float64(b.N)) / 1024 / 1024,
				AllocsPerOp: int64(m2.Mallocs-m1.Mallocs) / int64(b.N),
				BytesPerOp:  int64(m2.TotalAlloc-m1.TotalAlloc) / int64(b.N),
			}
			
			results = append(results, result)
			
			// Log detailed results
			b.Logf("Size: %s, Duration: %v, Throughput: %.2f MB/s, Allocs: %d, Bytes: %d",
				result.Name, result.Duration, result.Throughput, result.AllocsPerOp, result.BytesPerOp)
		})
	}
	
	return results
}

// formatSize converts bytes to human readable format
func formatSize(bytes int) string {
	if bytes < 1024 {
		return fmt.Sprintf("%dB", bytes)
	}
	if bytes < 1024*1024 {
		return fmt.Sprintf("%dKB", bytes/1024)
	}
	if bytes < 1024*1024*1024 {
		return fmt.Sprintf("%dMB", bytes/(1024*1024))
	}
	return fmt.Sprintf("%dGB", bytes/(1024*1024*1024))
}

// humanizeSize is an alias to formatSize for consistency
func humanizeSize(bytes int) string {
	return formatSize(bytes)
}

// MemoryProfiler helps track memory usage during stream operations
type MemoryProfiler struct {
	name     string
	ctx      context.Context
	baseline runtime.MemStats
}

func NewMemoryProfiler(ctx context.Context, name string) *MemoryProfiler {
	runtime.GC()
	var baseline runtime.MemStats
	runtime.ReadMemStats(&baseline)
	
	return &MemoryProfiler{
		name:     name,
		ctx:      ctx,
		baseline: baseline,
	}
}

func (mp *MemoryProfiler) Checkpoint(label string) {
	var current runtime.MemStats
	runtime.ReadMemStats(&current)
	
	allocDiff := current.TotalAlloc - mp.baseline.TotalAlloc
	heapDiff := current.HeapAlloc - mp.baseline.HeapAlloc
	
	fmt.Printf("Memory checkpoint [%s - %s]: Alloc: +%d bytes, Heap: +%d bytes\n",
		mp.name, label, allocDiff, heapDiff)
} 