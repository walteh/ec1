package tstream

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"sync/atomic"
	"time"
)

// TimingReader wraps an io.Reader to measure read performance
type TimingReader struct {
	reader    io.Reader
	name      string
	ctx       context.Context
	
	// Metrics
	totalBytes   int64
	totalReads   int64
	totalTime    int64 // nanoseconds
	slowestRead  int64 // nanoseconds
	startTime    time.Time
}

// NewTimingReader creates a new timing reader wrapper
func NewTimingReader(ctx context.Context, reader io.Reader, name string) *TimingReader {
	return &TimingReader{
		reader:    reader,
		name:      name,
		ctx:       ctx,
		startTime: time.Now(),
	}
}

func (tr *TimingReader) Read(p []byte) (n int, err error) {
	start := time.Now()
	n, err = tr.reader.Read(p)
	elapsed := time.Since(start)
	
	atomic.AddInt64(&tr.totalBytes, int64(n))
	atomic.AddInt64(&tr.totalReads, 1)
	atomic.AddInt64(&tr.totalTime, elapsed.Nanoseconds())
	
	// Track slowest read
	for {
		slowest := atomic.LoadInt64(&tr.slowestRead)
		if elapsed.Nanoseconds() <= slowest {
			break
		}
		if atomic.CompareAndSwapInt64(&tr.slowestRead, slowest, elapsed.Nanoseconds()) {
			break
		}
	}
	
	// Log slow reads (> 100ms)
	if elapsed > 100*time.Millisecond {
		slog.WarnContext(tr.ctx, "slow read detected", 
			"stream", tr.name,
			"bytes", n,
			"duration", elapsed,
			"rate_mbps", float64(n)/elapsed.Seconds()/1024/1024)
	}
	
	return n, err
}

func (tr *TimingReader) Close() error {
	totalDuration := time.Since(tr.startTime)
	totalBytes := atomic.LoadInt64(&tr.totalBytes)
	totalReads := atomic.LoadInt64(&tr.totalReads)
	totalTime := time.Duration(atomic.LoadInt64(&tr.totalTime))
	slowestRead := time.Duration(atomic.LoadInt64(&tr.slowestRead))
	
	var avgReadSize float64
	var avgReadTime time.Duration
	if totalReads > 0 {
		avgReadSize = float64(totalBytes) / float64(totalReads)
		avgReadTime = totalTime / time.Duration(totalReads)
	}
	
	var throughputMBps float64
	if totalDuration.Seconds() > 0 {
		throughputMBps = float64(totalBytes) / totalDuration.Seconds() / 1024 / 1024
	}
	
	slog.InfoContext(tr.ctx, "stream timing summary",
		"stream", tr.name,
		"total_bytes", totalBytes,
		"total_reads", totalReads,
		"total_duration", totalDuration,
		"avg_read_size", fmt.Sprintf("%.1f bytes", avgReadSize),
		"avg_read_time", avgReadTime,
		"slowest_read", slowestRead,
		"throughput_mbps", fmt.Sprintf("%.2f MB/s", throughputMBps))
	
	if closer, ok := tr.reader.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

// ProgressReader shows real-time progress for large streams
type ProgressReader struct {
	reader      io.Reader
	name        string
	ctx         context.Context
	totalSize   int64
	currentSize int64
	lastReport  time.Time
	startTime   time.Time
}

func NewProgressReader(ctx context.Context, reader io.Reader, name string, totalSize int64) *ProgressReader {
	return &ProgressReader{
		reader:     reader,
		name:       name,
		ctx:        ctx,
		totalSize:  totalSize,
		startTime:  time.Now(),
		lastReport: time.Now(),
	}
}

func (pr *ProgressReader) Read(p []byte) (n int, err error) {
	n, err = pr.reader.Read(p)
	pr.currentSize += int64(n)
	
	// Report progress every 5 seconds or 10MB
	now := time.Now()
	if now.Sub(pr.lastReport) > 5*time.Second || pr.currentSize-atomic.LoadInt64(&pr.currentSize) > 10*1024*1024 {
		pr.reportProgress()
		pr.lastReport = now
	}
	
	return n, err
}

func (pr *ProgressReader) reportProgress() {
	elapsed := time.Since(pr.startTime)
	var percent float64
	if pr.totalSize > 0 {
		percent = float64(pr.currentSize) / float64(pr.totalSize) * 100
	}
	
	var rate float64
	if elapsed.Seconds() > 0 {
		rate = float64(pr.currentSize) / elapsed.Seconds() / 1024 / 1024
	}
	
	var eta time.Duration
	if pr.currentSize > 0 && pr.totalSize > 0 {
		eta = time.Duration(float64(elapsed) * (float64(pr.totalSize)/float64(pr.currentSize) - 1))
	}
	
	slog.InfoContext(pr.ctx, "stream progress",
		"stream", pr.name,
		"progress", fmt.Sprintf("%.1f%%", percent),
		"current_mb", pr.currentSize/1024/1024,
		"total_mb", pr.totalSize/1024/1024,
		"rate_mbps", fmt.Sprintf("%.2f MB/s", rate),
		"eta", eta.Round(time.Second))
} 