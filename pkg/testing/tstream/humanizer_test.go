package tstream

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewResultsHumanizer(t *testing.T) {
	humanizer := NewResultsHumanizer()
	require.NotNil(t, humanizer, "humanizer should not be nil")
	assert.Equal(t, 0, len(humanizer.results), "results should be empty initially")
}

func TestResultsHumanizer_AddResult(t *testing.T) {
	humanizer := NewResultsHumanizer()
	
	result := BenchmarkResult{
		Name:        "test_implementation",
		Size:        "1MB",
		Duration:    time.Millisecond * 100,
		Throughput:  500.0,
		AllocsPerOp: 10,
		BytesPerOp:  1000,
		Iterations:  100,
	}
	
	humanizer.AddResult(result)
	
	assert.Equal(t, 1, len(humanizer.results), "should have one result")
	assert.Equal(t, result, humanizer.results[0], "result should match what was added")
}

func TestResultsHumanizer_AnalyzeEmpty(t *testing.T) {
	humanizer := NewResultsHumanizer()
	
	analysis := humanizer.Analyze()
	
	assert.Equal(t, BenchmarkResult{}, analysis.FastestOverall, "fastest should be empty")
	assert.Equal(t, 0, len(analysis.BySize), "size analysis should be empty")
	assert.Equal(t, 0, len(analysis.Recommendations), "recommendations should be empty")
}

func TestResultsHumanizer_AnalyzeSingleResult(t *testing.T) {
	humanizer := NewResultsHumanizer()
	
	result := BenchmarkResult{
		Name:        "single_test",
		Size:        "10MB", 
		Duration:    time.Millisecond * 200,
		Throughput:  1500.0,
		AllocsPerOp: 50,
		BytesPerOp:  2000,
		Iterations:  10,
	}
	
	humanizer.AddResult(result)
	analysis := humanizer.Analyze()
	
	assert.Equal(t, result, analysis.FastestOverall, "single result should be fastest")
	assert.Equal(t, result, analysis.SlowestOverall, "single result should be slowest")
	assert.Equal(t, result, analysis.MostEfficient, "single result should be most efficient")
	assert.Equal(t, result, analysis.LeastAllocations, "single result should have least allocations")
	assert.Equal(t, result, analysis.SmallestMemory, "single result should have smallest memory")
	
	assert.Equal(t, 1, len(analysis.BySize), "should have one size group")
	sizeAnalysis, exists := analysis.BySize["10MB"]
	assert.True(t, exists, "10MB analysis should exist")
	assert.Equal(t, result, sizeAnalysis.Fastest, "size fastest should match")
	assert.Equal(t, "single_test", sizeAnalysis.RecommendedChoice, "recommendation should be the only option")
}

func TestResultsHumanizer_AnalyzeMultipleResults(t *testing.T) {
	humanizer := NewResultsHumanizer()
	
	fast := BenchmarkResult{
		Name:        "fast_impl",
		Size:        "1MB",
		Duration:    time.Millisecond * 50,  // Fastest
		Throughput:  2000.0,                 // Most efficient
		AllocsPerOp: 5,                      // Least allocations
		BytesPerOp:  500,                    // Smallest memory
		Iterations:  100,
	}
	
	slow := BenchmarkResult{
		Name:        "slow_impl", 
		Size:        "1MB",
		Duration:    time.Millisecond * 500, // Slowest
		Throughput:  100.0,
		AllocsPerOp: 100,
		BytesPerOp:  5000,
		Iterations:  10,
	}
	
	medium := BenchmarkResult{
		Name:        "medium_impl",
		Size:        "1MB", 
		Duration:    time.Millisecond * 200,
		Throughput:  800.0,
		AllocsPerOp: 20,
		BytesPerOp:  1500,
		Iterations:  50,
	}
	
	humanizer.AddResult(fast)
	humanizer.AddResult(slow)
	humanizer.AddResult(medium)
	
	analysis := humanizer.Analyze()
	
	assert.Equal(t, fast, analysis.FastestOverall, "fast should be fastest overall")
	assert.Equal(t, slow, analysis.SlowestOverall, "slow should be slowest overall")
	assert.Equal(t, fast, analysis.MostEfficient, "fast should be most efficient")
	assert.Equal(t, fast, analysis.LeastAllocations, "fast should have least allocations")
	assert.Equal(t, fast, analysis.SmallestMemory, "fast should have smallest memory")
	
	// Performance spread should be 500ms / 50ms = 10x
	assert.InDelta(t, 10.0, analysis.PerformanceSpread, 0.1, "performance spread should be 10x")
	
	// Size analysis
	assert.Equal(t, 1, len(analysis.BySize), "should have one size group")
	sizeAnalysis := analysis.BySize["1MB"]
	assert.Equal(t, fast, sizeAnalysis.Fastest, "fast should be fastest for size")
	assert.Equal(t, slow, sizeAnalysis.Slowest, "slow should be slowest for size")
	assert.Equal(t, "fast_impl", sizeAnalysis.RecommendedChoice, "fast_impl should be recommended")
	assert.Equal(t, "10.0x faster", sizeAnalysis.SpeedDifference, "speed difference should be calculated")
}

func TestResultsHumanizer_AnalyzeMultipleSizes(t *testing.T) {
	humanizer := NewResultsHumanizer()
	
	small := BenchmarkResult{
		Name:        "impl_a",
		Size:        "1MB",
		Duration:    time.Millisecond * 100,
		Throughput:  1000.0,
		AllocsPerOp: 10,
		BytesPerOp:  1000,
	}
	
	large := BenchmarkResult{
		Name:        "impl_b",
		Size:        "100MB",
		Duration:    time.Second * 2,
		Throughput:  2000.0,
		AllocsPerOp: 100,
		BytesPerOp:  10000,
	}
	
	humanizer.AddResult(small)
	humanizer.AddResult(large)
	
	analysis := humanizer.Analyze()
	
	assert.Equal(t, 2, len(analysis.BySize), "should have two size groups")
	assert.Contains(t, analysis.BySize, "1MB", "should contain 1MB analysis")
	assert.Contains(t, analysis.BySize, "100MB", "should contain 100MB analysis")
}

func TestResultsHumanizer_HumanizeResults_Empty(t *testing.T) {
	humanizer := NewResultsHumanizer()
	
	output := humanizer.HumanizeResults()
	
	assert.Contains(t, output, "BENCHMARK ANALYSIS REPORT", "should contain report header")
	assert.Contains(t, output, "❌ No benchmark results to analyze", "should indicate no results")
}

func TestResultsHumanizer_HumanizeResults_WithData(t *testing.T) {
	humanizer := NewResultsHumanizer()
	
	result := BenchmarkResult{
		Name:        "test_impl",
		Size:        "5MB",
		Duration:    time.Millisecond * 150,
		Throughput:  1200.0,
		AllocsPerOp: 25,
		BytesPerOp:  2048,
		Iterations:  20,
	}
	
	humanizer.AddResult(result)
	output := humanizer.HumanizeResults()
	
	assert.Contains(t, output, "🏆 BENCHMARK ANALYSIS REPORT", "should contain report header")
	assert.Contains(t, output, "🥇 OVERALL WINNERS", "should contain winners section")
	assert.Contains(t, output, "test_impl", "should contain implementation name")
	assert.Contains(t, output, "150ms", "should contain duration")
	assert.Contains(t, output, "1200.00 MB/s", "should contain throughput")
	assert.Contains(t, output, "25 allocs/op", "should contain allocations")
	assert.Contains(t, output, "2.0 KB/op", "should contain memory usage")
	assert.Contains(t, output, "📏 SIZE-SPECIFIC ANALYSIS", "should contain size analysis")
	assert.Contains(t, output, "5MB", "should contain size")
	assert.Contains(t, output, "📋 PERFORMANCE MATRIX", "should contain performance matrix")
}

func TestResultsHumanizer_HumanizeResults_WithRecommendations(t *testing.T) {
	humanizer := NewResultsHumanizer()
	
	// Add results that will trigger recommendations
	fast := BenchmarkResult{
		Name:        "fast_impl",
		Duration:    time.Millisecond * 10,
		Throughput:  2000.0, // High throughput -> production ready recommendation
		AllocsPerOp: 5,
		BytesPerOp:  100,
	}
	
	slow := BenchmarkResult{
		Name:        "slow_impl", 
		Duration:    time.Millisecond * 200, // 20x difference -> large spread recommendation
		Throughput:  50.0,    // Low throughput -> bottleneck recommendation
		AllocsPerOp: 2000,    // High allocs -> memory optimization recommendation
		BytesPerOp:  50000,
	}
	
	humanizer.AddResult(fast)
	humanizer.AddResult(slow)
	
	output := humanizer.HumanizeResults()
	
	assert.Contains(t, output, "💡 RECOMMENDATIONS", "should contain recommendations section")
	// Should contain various recommendation types based on the data
	recommendationsSection := extractSection(output, "💡 RECOMMENDATIONS", "📋 PERFORMANCE MATRIX")
	assert.NotEmpty(t, recommendationsSection, "recommendations section should not be empty")
}

func TestHumanizeBytes(t *testing.T) {
	tests := []struct {
		name     string
		bytes    int64
		expected string
	}{
		{"zero", 0, "0 B"},
		{"bytes", 512, "512 B"},
		{"kilobytes", 1024, "1.0 KB"},
		{"kilobytes_decimal", 1536, "1.5 KB"},
		{"megabytes", 1048576, "1.0 MB"},
		{"megabytes_decimal", 2621440, "2.5 MB"},
		{"gigabytes", 1073741824, "1.0 GB"},
		{"large_number", 5368709120, "5.0 GB"},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := humanizeBytes(tt.bytes)
			assert.Equal(t, tt.expected, result, "humanizeBytes should format correctly")
		})
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{"short_string", "hello", 10, "hello"},
		{"exact_length", "hello", 5, "hello"},
		{"truncate_needed", "hello world", 8, "hello..."},
		{"very_short_limit", "hello world", 5, "he..."},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateString(tt.input, tt.maxLen)
			assert.Equal(t, tt.expected, result, "truncateString should truncate correctly")
			assert.LessOrEqual(t, len(result), tt.maxLen, "result should not exceed max length")
		})
	}
}

func TestResultsHumanizer_GenerateRecommendations(t *testing.T) {
	humanizer := NewResultsHumanizer()
	
	analysis := PerformanceAnalysis{
		PerformanceSpread: 15.0, // Large spread
		MostEfficient: BenchmarkResult{
			Throughput: 2500.0, // High throughput
		},
		LeastAllocations: BenchmarkResult{
			AllocsPerOp: 1500, // High allocations
		},
	}
	
	recommendations := humanizer.generateRecommendations(analysis)
	
	assert.GreaterOrEqual(t, len(recommendations), 2, "should generate multiple recommendations")
	
	// Convert to single string for easier searching
	allRecs := strings.Join(recommendations, " ")
	assert.Contains(t, allRecs, "Large performance differences", "should recommend using fastest for large spread")
	assert.Contains(t, allRecs, "production-ready", "should recommend production readiness for high throughput")
	assert.Contains(t, allRecs, "memory usage", "should recommend memory optimization for high allocations")
}

func TestResultsHumanizer_RecommendForSize(t *testing.T) {
	humanizer := NewResultsHumanizer()
	
	// Test empty results
	recommendation := humanizer.recommendForSize([]BenchmarkResult{})
	assert.Equal(t, "No data", recommendation, "should return 'No data' for empty results")
	
	// Test with clear winner (wins in all categories)
	winner := BenchmarkResult{
		Name:        "clear_winner",
		Duration:    time.Millisecond * 10,  // Fastest
		Throughput:  2000.0,                 // Most efficient
		AllocsPerOp: 5,                      // Least allocations
	}
	
	other := BenchmarkResult{
		Name:        "other_impl",
		Duration:    time.Millisecond * 100,
		Throughput:  500.0,
		AllocsPerOp: 50,
	}
	
	recommendation = humanizer.recommendForSize([]BenchmarkResult{winner, other})
	assert.Equal(t, "clear_winner", recommendation, "should recommend the clear winner")
}

// Helper function to extract a section from the humanized output
func extractSection(output, startMarker, endMarker string) string {
	startIdx := strings.Index(output, startMarker)
	if startIdx == -1 {
		return ""
	}
	
	endIdx := strings.Index(output[startIdx:], endMarker)
	if endIdx == -1 {
		return output[startIdx:]
	}
	
	return output[startIdx : startIdx+endIdx]
} 