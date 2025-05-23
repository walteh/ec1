package tstream

import (
	"fmt"
	"testing"
	"time"
)

// TestHumanizerDemo demonstrates the humanizer with realistic benchmark data
func TestHumanizerDemo(t *testing.T) {
	humanizer := NewResultsHumanizer()
	
	// Add realistic performance data that might come from actual benchmarks
	results := []BenchmarkResult{
		{
			Name:        "stream_original",
			Size:        "1MB",
			Duration:    time.Millisecond * 250,
			Throughput:  4.0,
			AllocsPerOp: 1200,
			BytesPerOp:  2048000,
			Iterations:  100,
		},
		{
			Name:        "stream_optimized", 
			Size:        "1MB",
			Duration:    time.Millisecond * 180,
			Throughput:  5.6,
			AllocsPerOp: 800,
			BytesPerOp:  1536000,
			Iterations:  100,
		},
		{
			Name:        "stream_hyper",
			Size:        "1MB", 
			Duration:    time.Millisecond * 120,
			Throughput:  8.3,
			AllocsPerOp: 400,
			BytesPerOp:  1024000,
			Iterations:  100,
		},
		{
			Name:        "fast_blazing",
			Size:        "1MB",
			Duration:    time.Millisecond * 45,
			Throughput:  22.2,
			AllocsPerOp: 150,
			BytesPerOp:  512000,
			Iterations:  100,
		},
		// 10MB results
		{
			Name:        "stream_original",
			Size:        "10MB", 
			Duration:    time.Millisecond * 2500,
			Throughput:  4.0,
			AllocsPerOp: 12000,
			BytesPerOp:  20480000,
			Iterations:  10,
		},
		{
			Name:        "stream_optimized",
			Size:        "10MB",
			Duration:    time.Millisecond * 1800,
			Throughput:  5.6,
			AllocsPerOp: 8000,
			BytesPerOp:  15360000,
			Iterations:  10,
		},
		{
			Name:        "stream_hyper",
			Size:        "10MB",
			Duration:    time.Millisecond * 1200, 
			Throughput:  8.3,
			AllocsPerOp: 4000,
			BytesPerOp:  10240000,
			Iterations:  10,
		},
		{
			Name:        "fast_blazing",
			Size:        "10MB",
			Duration:    time.Millisecond * 450,
			Throughput:  22.2,
			AllocsPerOp: 1500,
			BytesPerOp:  5120000,
			Iterations:  10,
		},
	}
	
	// Add all results to humanizer
	for _, result := range results {
		humanizer.AddResult(result)
	}
	
	// Generate and print the humanized analysis
	analysis := humanizer.HumanizeResults()
	
	fmt.Println("=== HUMANIZER DEMO OUTPUT ===")
	fmt.Println(analysis)
	fmt.Println("=== END DEMO OUTPUT ===")
	
	// Verify key components are present
	if len(analysis) < 100 {
		t.Error("Analysis seems too short")
	}
	
	// The analysis should clearly show fast_blazing as the winner
	t.Logf("Analysis generated successfully with %d characters", len(analysis))
} 