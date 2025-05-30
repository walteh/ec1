package tvmm

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/fatih/color"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	slogctx "github.com/veqryn/slog-context"

	"github.com/walteh/ec1/pkg/host"
	"github.com/walteh/ec1/pkg/oci"
	"github.com/walteh/ec1/pkg/vmm"
)

func RunHarpoonVM[VM vmm.VirtualMachine](t *testing.T, ctx context.Context, hv vmm.Hypervisor[VM], cfg vmm.ConatinerImageConfig, cache *oci.SimpleCache) *vmm.RunningVM[VM] {

	ctx = slogctx.WithGroup(ctx, "test-vm-setup")

	slog.DebugContext(ctx, "running vm", "memory", humanize.Bytes(uint64(cfg.Memory.ToBytes())))

	rvm, err := vmm.NewContainerizedVirtualMachine(ctx, hv, cache, cfg)
	require.NoError(t, err)

	go func() {
		slog.DebugContext(ctx, "vm running, waiting for vm to stop")
		err := rvm.WaitOnVmStopped()
		assert.NoError(t, err)
	}()

	t.Cleanup(func() {
		slog.DebugContext(ctx, "stopping vm")
		CatConsoleFile(t, ctx, rvm.VM())
		rvm.VM().HardStop(ctx)
	})

	err = vmm.WaitForVMState(ctx, rvm.VM(), vmm.VirtualMachineStateTypeRunning, time.After(30*time.Second))
	require.NoError(t, err, "timeout waiting for vm to be running: %v", err)

	select {
	case <-rvm.WaitOnVMReadyToExec():
	case <-time.After(3 * time.Second):
		t.Fatalf("timeout waiting for vm to be ready to exec")
	}
	return rvm
}

func BuildDiffReport(t *testing.T, title string, header1 string, header2 string, diffContent string) string {
	var result strings.Builder

	// Add report header
	result.WriteString(color.New(color.FgHiYellow, color.Faint).Sprintf("\n\n============= %s START =============\n\n", title))
	result.WriteString(fmt.Sprintf("%s\n\n", color.YellowString("%s", t.Name())))

	// Add type/value information headers if provided
	if header1 != "" {
		result.WriteString(header1 + "\n")
	}
	if header2 != "" {
		result.WriteString(header2 + "\n\n\n")
	}

	// Add diff content
	result.WriteString(diffContent + "\n\n")

	// Add report footer
	result.WriteString(color.New(color.FgHiYellow, color.Faint).Sprintf("============= %s END ===============\n\n", title))

	return result.String()
}

func CatConsoleFile(t *testing.T, ctx context.Context, rvm vmm.VirtualMachine) {
	cd, err := host.EmphiricalVMCacheDir(ctx, rvm.ID())
	if err != nil {
		t.Logf("Failed to get vm cache dir: %v", err)
		return
	}
	fullPath := filepath.Join(cd, "console.log")

	content, err := os.ReadFile(fullPath)
	if err != nil {
		t.Logf("Failed to read console.log: %v", err)
		return
	}

	t.Log(BuildDiffReport(t, "console.log", "", "", string(content)))
}
