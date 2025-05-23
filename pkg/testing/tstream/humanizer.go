package tstream

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// BenchmarkResult represents a single benchmark result
type BenchmarkResult struct {
	Name        string
	Size        string
	Duration    time.Duration
	Throughput  float64 // MB/s
	AllocsPerOp int64
	BytesPerOp  int64
	Iterations  int
}

// ResultsHumanizer analyzes benchmark results and provides human-readable insights
type ResultsHumanizer struct {
	results []BenchmarkResult
}

// NewResultsHumanizer creates a new results humanizer
func NewResultsHumanizer() *ResultsHumanizer {
	return &ResultsHumanizer{
		results: make([]BenchmarkResult, 0),
	}
}

// AddResult adds a benchmark result for analysis
func (rh *ResultsHumanizer) AddResult(result BenchmarkResult) {
	rh.results = append(rh.results, result)
}

// PerformanceAnalysis contains the analysis of benchmark results
type PerformanceAnalysis struct {
	FastestOverall     BenchmarkResult
	SlowestOverall     BenchmarkResult
	MostEfficient      BenchmarkResult  // Best throughput
	LeastAllocations   BenchmarkResult  // Fewest allocations
	SmallestMemory     BenchmarkResult  // Least memory per op
	BySize             map[string]SizeAnalysis
	Recommendations    []string
	PerformanceSpread  float64 // Ratio of fastest to slowest
}

// SizeAnalysis contains analysis for a specific data size
type SizeAnalysis struct {
	Size               string
	Fastest            BenchmarkResult
	Slowest            BenchmarkResult
	MostEfficient      BenchmarkResult
	LeastAllocations   BenchmarkResult
	SmallestMemory     BenchmarkResult
	SpeedDifference    string // Human readable speed difference
	RecommendedChoice  string
}

// Analyze performs comprehensive analysis of all benchmark results
func (rh *ResultsHumanizer) Analyze() PerformanceAnalysis {
	if len(rh.results) == 0 {
		return PerformanceAnalysis{}
	}

	analysis := PerformanceAnalysis{
		BySize: make(map[string]SizeAnalysis),
	}

	// Find overall winners
	analysis.FastestOverall = rh.findFastest(rh.results)
	analysis.SlowestOverall = rh.findSlowest(rh.results)
	analysis.MostEfficient = rh.findMostEfficient(rh.results)
	analysis.LeastAllocations = rh.findLeastAllocations(rh.results)
	analysis.SmallestMemory = rh.findSmallestMemory(rh.results)

	// Calculate performance spread
	if analysis.SlowestOverall.Duration > 0 {
		analysis.PerformanceSpread = float64(analysis.SlowestOverall.Duration) / float64(analysis.FastestOverall.Duration)
	}

	// Group by size and analyze
	sizeGroups := rh.groupBySize()
	for size, results := range sizeGroups {
		sizeAnalysis := SizeAnalysis{
			Size:               size,
			Fastest:            rh.findFastest(results),
			Slowest:            rh.findSlowest(results),
			MostEfficient:      rh.findMostEfficient(results),
			LeastAllocations:   rh.findLeastAllocations(results),
			SmallestMemory:     rh.findSmallestMemory(results),
		}

		// Calculate speed difference
		if sizeAnalysis.Slowest.Duration > 0 && sizeAnalysis.Fastest.Duration > 0 {
			speedRatio := float64(sizeAnalysis.Slowest.Duration) / float64(sizeAnalysis.Fastest.Duration)
			sizeAnalysis.SpeedDifference = fmt.Sprintf("%.1fx faster", speedRatio)
		}

		// Recommend best choice for this size
		sizeAnalysis.RecommendedChoice = rh.recommendForSize(results)

		analysis.BySize[size] = sizeAnalysis
	}

	// Generate recommendations
	analysis.Recommendations = rh.generateRecommendations(analysis)

	return analysis
}

// HumanizeResults returns a human-readable analysis of the benchmark results
func (rh *ResultsHumanizer) HumanizeResults() string {
	analysis := rh.Analyze()
	var builder strings.Builder

	builder.WriteString("🏆 BENCHMARK ANALYSIS REPORT\n")
	builder.WriteString("═══════════════════════════════════════════════════════════\n\n")

	if len(rh.results) == 0 {
		builder.WriteString("❌ No benchmark results to analyze\n")
		return builder.String()
	}

	// Overall Winners
	builder.WriteString("🥇 OVERALL WINNERS:\n")
	builder.WriteString(fmt.Sprintf("  ⚡ Fastest:           %s (%v)\n", analysis.FastestOverall.Name, analysis.FastestOverall.Duration))
	builder.WriteString(fmt.Sprintf("  🚀 Most Efficient:    %s (%.2f MB/s)\n", analysis.MostEfficient.Name, analysis.MostEfficient.Throughput))
	builder.WriteString(fmt.Sprintf("  🧠 Least Allocations: %s (%d allocs/op)\n", analysis.LeastAllocations.Name, analysis.LeastAllocations.AllocsPerOp))
	builder.WriteString(fmt.Sprintf("  💾 Least Memory:      %s (%s/op)\n", analysis.SmallestMemory.Name, humanizeBytes(analysis.SmallestMemory.BytesPerOp)))
	
	if analysis.PerformanceSpread > 1 {
		builder.WriteString(fmt.Sprintf("  📊 Performance Spread: %.1fx (fastest vs slowest)\n", analysis.PerformanceSpread))
	}
	builder.WriteString("\n")

	// Size-specific analysis
	builder.WriteString("📏 SIZE-SPECIFIC ANALYSIS:\n")
	
	// Sort sizes for consistent output
	sizes := make([]string, 0, len(analysis.BySize))
	for size := range analysis.BySize {
		sizes = append(sizes, size)
	}
	sort.Strings(sizes)

	for _, size := range sizes {
		sizeAnalysis := analysis.BySize[size]
		builder.WriteString(fmt.Sprintf("  📦 %s:\n", size))
		builder.WriteString(fmt.Sprintf("    Winner: %s (%s)\n", sizeAnalysis.Fastest.Name, sizeAnalysis.SpeedDifference))
		builder.WriteString(fmt.Sprintf("    Choice: %s\n", sizeAnalysis.RecommendedChoice))
		builder.WriteString("\n")
	}

	// Recommendations
	if len(analysis.Recommendations) > 0 {
		builder.WriteString("💡 RECOMMENDATIONS:\n")
		for i, rec := range analysis.Recommendations {
			builder.WriteString(fmt.Sprintf("  %d. %s\n", i+1, rec))
		}
		builder.WriteString("\n")
	}

	// Performance Matrix
	builder.WriteString("📋 PERFORMANCE MATRIX:\n")
	builder.WriteString(rh.generatePerformanceMatrix())

	return builder.String()
}

// Helper methods
func (rh *ResultsHumanizer) findFastest(results []BenchmarkResult) BenchmarkResult {
	if len(results) == 0 {
		return BenchmarkResult{}
	}
	
	fastest := results[0]
	for _, result := range results[1:] {
		if result.Duration < fastest.Duration {
			fastest = result
		}
	}
	return fastest
}

func (rh *ResultsHumanizer) findSlowest(results []BenchmarkResult) BenchmarkResult {
	if len(results) == 0 {
		return BenchmarkResult{}
	}
	
	slowest := results[0]
	for _, result := range results[1:] {
		if result.Duration > slowest.Duration {
			slowest = result
		}
	}
	return slowest
}

func (rh *ResultsHumanizer) findMostEfficient(results []BenchmarkResult) BenchmarkResult {
	if len(results) == 0 {
		return BenchmarkResult{}
	}
	
	mostEfficient := results[0]
	for _, result := range results[1:] {
		if result.Throughput > mostEfficient.Throughput {
			mostEfficient = result
		}
	}
	return mostEfficient
}

func (rh *ResultsHumanizer) findLeastAllocations(results []BenchmarkResult) BenchmarkResult {
	if len(results) == 0 {
		return BenchmarkResult{}
	}
	
	leastAllocs := results[0]
	for _, result := range results[1:] {
		if result.AllocsPerOp < leastAllocs.AllocsPerOp {
			leastAllocs = result
		}
	}
	return leastAllocs
}

func (rh *ResultsHumanizer) findSmallestMemory(results []BenchmarkResult) BenchmarkResult {
	if len(results) == 0 {
		return BenchmarkResult{}
	}
	
	smallestMem := results[0]
	for _, result := range results[1:] {
		if result.BytesPerOp < smallestMem.BytesPerOp {
			smallestMem = result
		}
	}
	return smallestMem
}

func (rh *ResultsHumanizer) groupBySize() map[string][]BenchmarkResult {
	groups := make(map[string][]BenchmarkResult)
	for _, result := range rh.results {
		groups[result.Size] = append(groups[result.Size], result)
	}
	return groups
}

func (rh *ResultsHumanizer) recommendForSize(results []BenchmarkResult) string {
	if len(results) == 0 {
		return "No data"
	}

	fastest := rh.findFastest(results)
	mostEfficient := rh.findMostEfficient(results)
	leastAllocs := rh.findLeastAllocations(results)

	// Simple scoring system
	scores := make(map[string]int)
	scores[fastest.Name]++
	scores[mostEfficient.Name]++
	scores[leastAllocs.Name]++

	maxScore := 0
	recommended := ""
	for name, score := range scores {
		if score > maxScore {
			maxScore = score
			recommended = name
		}
	}

	return recommended
}

func (rh *ResultsHumanizer) generateRecommendations(analysis PerformanceAnalysis) []string {
	var recommendations []string

	// Performance spread recommendation
	if analysis.PerformanceSpread > 10 {
		recommendations = append(recommendations, "⚠️  Large performance differences detected - consider using the fastest implementation")
	} else if analysis.PerformanceSpread < 2 {
		recommendations = append(recommendations, "✅ Performance is consistent across implementations - choose based on other factors")
	}

	// Memory recommendations
	if analysis.LeastAllocations.AllocsPerOp > 1000 {
		recommendations = append(recommendations, "🧠 High allocation count detected - consider optimizing memory usage")
	}

	// Throughput recommendations
	if analysis.MostEfficient.Throughput > 1000 {
		recommendations = append(recommendations, "🚀 Excellent throughput achieved - this implementation is production-ready")
	} else if analysis.MostEfficient.Throughput < 100 {
		recommendations = append(recommendations, "🐌 Low throughput detected - investigate potential bottlenecks")
	}

	return recommendations
}

func (rh *ResultsHumanizer) generatePerformanceMatrix() string {
	var builder strings.Builder
	
	builder.WriteString("  Implementation        | Duration    | Throughput  | Allocs/op | Memory/op\n")
	builder.WriteString("  ---------------------|-------------|-------------|-----------|----------\n")
	
	for _, result := range rh.results {
		name := fmt.Sprintf("%-20s", truncateString(result.Name, 20))
		duration := fmt.Sprintf("%-11v", result.Duration)
		throughput := fmt.Sprintf("%-11s", fmt.Sprintf("%.2f MB/s", result.Throughput))
		allocs := fmt.Sprintf("%-9d", result.AllocsPerOp)
		memory := fmt.Sprintf("%-9s", humanizeBytes(result.BytesPerOp))
		
		builder.WriteString(fmt.Sprintf("  %s | %s | %s | %s | %s\n", 
			name, duration, throughput, allocs, memory))
	}
	
	return builder.String()
}

// Utility functions
func humanizeBytes(bytes int64) string {
	if bytes == 0 {
		return "0 B"
	}
	
	units := []string{"B", "KB", "MB", "GB"}
	size := float64(bytes)
	unit := 0
	
	for size >= 1024 && unit < len(units)-1 {
		size /= 1024
		unit++
	}
	
	if size < 10 && unit > 0 {
		return fmt.Sprintf("%.1f %s", size, units[unit])
	}
	return fmt.Sprintf("%.0f %s", size, units[unit])
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
} 