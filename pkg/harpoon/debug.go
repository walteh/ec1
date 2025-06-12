package harpoon

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"runtime/pprof"
	"slices"
	"time"

	"github.com/shirou/gopsutil/v3/process"

	"github.com/walteh/ec1/pkg/ec1init"
)

type MemoryConsumer struct {
	Name       string
	Process    *process.Process
	MemoryInfo *process.MemoryInfoStat
}

func TopMemoryConsumers(count int) ([]MemoryConsumer, error) {
	procs, err := process.Processes()
	if err != nil {
		return nil, err
	}
	var list []MemoryConsumer
	for _, p := range procs {
		mi, err := p.MemoryInfo()
		if err == nil {
			name, _ := p.Name()
			list = append(list, MemoryConsumer{Name: name, Process: p, MemoryInfo: mi})
		}
	}
	slices.SortFunc(list, func(a, b MemoryConsumer) int { return int(b.MemoryInfo.RSS - a.MemoryInfo.RSS) })
	return list[:count], nil
}

func DumpHeapProfileAfter(ctx context.Context, d time.Duration) {

	slog.InfoContext(ctx, "waiting to dump heap profile", "duration", d)
	// wait however long you like
	time.Sleep(d)

	slog.InfoContext(ctx, "creating heap profile file")

	// dump heap profile to the shared directory
	f, err := os.Create(filepath.Join(ec1init.Ec1AbsPath, "memory.trace"))
	if err != nil {
		slog.ErrorContext(ctx, "ERROR creating memory.trace", "error", err)
		return
	}
	defer f.Close()

	slog.InfoContext(ctx, "writing heap profile")

	if err := pprof.WriteHeapProfile(f); err != nil {
		slog.ErrorContext(ctx, "ERROR writing heap profile", "error", err)
	} else {
		slog.InfoContext(ctx, "WROTE heap profile to /ec1/memory.trace")
	}
}
