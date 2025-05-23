package tstream

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"time"
)

// ProfiledOperation wraps a stream operation with CPU and memory profiling
type ProfiledOperation struct {
	name        string
	profileDir  string
	ctx         context.Context
	cpuProfile  *os.File
	memProfile  string
	startTime   time.Time
}

// NewProfiledOperation creates a new profiled operation
func NewProfiledOperation(ctx context.Context, name string, profileDir string) (*ProfiledOperation, error) {
	err := os.MkdirAll(profileDir, 0755)
	if err != nil {
		return nil, fmt.Errorf("creating profile directory: %w", err)
	}
	
	return &ProfiledOperation{
		name:       name,
		profileDir: profileDir,
		ctx:        ctx,
		memProfile: filepath.Join(profileDir, fmt.Sprintf("%s_mem.prof", name)),
	}, nil
}

// Start begins profiling
func (po *ProfiledOperation) Start() error {
	// Start CPU profiling
	cpuFile := filepath.Join(po.profileDir, fmt.Sprintf("%s_cpu.prof", po.name))
	var err error
	po.cpuProfile, err = os.Create(cpuFile)
	if err != nil {
		return fmt.Errorf("creating CPU profile file: %w", err)
	}
	
	if err := pprof.StartCPUProfile(po.cpuProfile); err != nil {
		po.cpuProfile.Close()
		return fmt.Errorf("starting CPU profile: %w", err)
	}
	
	po.startTime = time.Now()
	slog.InfoContext(po.ctx, "started profiling", "operation", po.name, "profile_dir", po.profileDir)
	
	return nil
}

// Stop ends profiling and writes memory profile
func (po *ProfiledOperation) Stop() error {
	duration := time.Since(po.startTime)
	
	// Stop CPU profiling
	pprof.StopCPUProfile()
	if po.cpuProfile != nil {
		po.cpuProfile.Close()
	}
	
	// Write memory profile
	runtime.GC() // Force GC before memory profile
	memFile, err := os.Create(po.memProfile)
	if err != nil {
		return fmt.Errorf("creating memory profile file: %w", err)
	}
	defer memFile.Close()
	
	if err := pprof.WriteHeapProfile(memFile); err != nil {
		return fmt.Errorf("writing memory profile: %w", err)
	}
	
	slog.InfoContext(po.ctx, "profiling completed", 
		"operation", po.name, 
		"duration", duration,
		"cpu_profile", filepath.Join(po.profileDir, fmt.Sprintf("%s_cpu.prof", po.name)),
		"mem_profile", po.memProfile)
	
	return nil
}

// ProfiledReader wraps an io.Reader with automatic profiling
type ProfiledReader struct {
	reader    io.Reader
	operation *ProfiledOperation
	started   bool
}

// NewProfiledReader creates a reader that automatically profiles when first read
func NewProfiledReader(ctx context.Context, reader io.Reader, name string, profileDir string) (*ProfiledReader, error) {
	op, err := NewProfiledOperation(ctx, name, profileDir)
	if err != nil {
		return nil, err
	}
	
	return &ProfiledReader{
		reader:    reader,
		operation: op,
	}, nil
}

func (pr *ProfiledReader) Read(p []byte) (n int, err error) {
	if !pr.started {
		if err := pr.operation.Start(); err != nil {
			return 0, fmt.Errorf("starting profiling: %w", err)
		}
		pr.started = true
	}
	
	return pr.reader.Read(p)
}

func (pr *ProfiledReader) Close() error {
	if pr.started {
		if err := pr.operation.Stop(); err != nil {
			slog.WarnContext(pr.operation.ctx, "error stopping profiling", "error", err)
		}
	}
	
	if closer, ok := pr.reader.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

// StreamProfileSuite runs a complete profiling suite for stream operations
type StreamProfileSuite struct {
	baseDir    string
	operations map[string]func(context.Context, io.Reader) (io.ReadCloser, error)
	dataGen    func() io.Reader
	ctx        context.Context
}

// NewStreamProfileSuite creates a new profiling suite
func NewStreamProfileSuite(ctx context.Context, baseDir string) *StreamProfileSuite {
	return &StreamProfileSuite{
		baseDir:    baseDir,
		operations: make(map[string]func(context.Context, io.Reader) (io.ReadCloser, error)),
		ctx:        ctx,
	}
}

// AddOperation adds an operation to profile
func (sps *StreamProfileSuite) AddOperation(name string, op func(context.Context, io.Reader) (io.ReadCloser, error)) {
	sps.operations[name] = op
}

// WithDataGenerator sets the data generator
func (sps *StreamProfileSuite) WithDataGenerator(gen func() io.Reader) *StreamProfileSuite {
	sps.dataGen = gen
	return sps
}

// RunAll runs all operations with profiling
func (sps *StreamProfileSuite) RunAll() error {
	for name, op := range sps.operations {
		if err := sps.runSingle(name, op); err != nil {
			return fmt.Errorf("profiling operation %s: %w", name, err)
		}
	}
	
	slog.InfoContext(sps.ctx, "profiling suite completed", 
		"operations", len(sps.operations), 
		"profile_dir", sps.baseDir)
	
	return nil
}

func (sps *StreamProfileSuite) runSingle(name string, op func(context.Context, io.Reader) (io.ReadCloser, error)) error {
	profileDir := filepath.Join(sps.baseDir, name)
	
	data := sps.dataGen()
	profiledData, err := NewProfiledReader(sps.ctx, data, name, profileDir)
	if err != nil {
		return err
	}
	defer profiledData.Close()
	
	result, err := op(sps.ctx, profiledData)
	if err != nil {
		return fmt.Errorf("operation failed: %w", err)
	}
	defer result.Close()
	
	// Consume the result to complete the operation
	_, err = io.Copy(io.Discard, result)
	return err
} 