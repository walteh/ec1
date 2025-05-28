package vf_test

import (
	"context"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/containers/common/pkg/strongunits"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	slogctx "github.com/veqryn/slog-context"

	"github.com/walteh/ec1/pkg/testing/tctx"
	"github.com/walteh/ec1/pkg/testing/tlog"
	"github.com/walteh/ec1/pkg/vmm"
	"github.com/walteh/ec1/pkg/vmm/vf"
)

func runHarpoonVM(t *testing.T, ctx context.Context, cfg vmm.ConatinerImageConfig) *vmm.RunningVM[*vf.VirtualMachine] {
	hv := vf.NewHypervisor()

	ctx = slogctx.WithGroup(ctx, "test-vm-setup")

	slog.DebugContext(ctx, "running vm", "memory", cfg.Memory, "memory.ToBytes()", cfg.Memory.ToBytes())

	rvm, err := vmm.NewContainerizedVirtualMachine(ctx, hv, cfg)
	require.NoError(t, err)

	go func() {
		slog.DebugContext(ctx, "vm running, waiting for vm to stop")
		err := rvm.WaitOnVmStopped()
		assert.NoError(t, err)
	}()

	t.Cleanup(func() {
		slog.DebugContext(ctx, "vm stopped")
		catConsoleFile(t, ctx, rvm.VM())
		rvm.VM().VZ().Stop()
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
func TestHarpoon(t *testing.T) {
	ctx := tlog.SetupSlogForTest(t)
	ctx = tctx.WithContext(ctx, t)

	// Create a real VM for testing
	rvm := runHarpoonVM(t, ctx, vmm.ConatinerImageConfig{
		ImageRef: "docker.io/oven/bun:debian",
		Cmdline:  []string{},
		Arch:     "arm64",
		OS:       "linux",
		Memory:   strongunits.MiB(1024).ToBytes(),
		VCPUs:    1,
	})

	slog.DebugContext(ctx, "waiting for test VM to be running")

	t.Logf("ready to exec")

	var errres error
	var stdout string
	var stderr string
	var exitCode string
	var errchan = make(chan error, 1)

	go func() {
		start := time.Now()
		// Verify the OCI container filesystem is properly mounted and accessible
		// Focus on filesystem verification rather than binary execution due to library dependencies
		stdout, stderr, exitCode, errres = vmm.Exec(ctx, rvm, "/usr/local/bin/bun --version")
		slog.InfoContext(ctx, "bun --version", "duration", time.Since(start))
		errchan <- errres
	}()

	select {
	case <-errchan:
	case <-time.After(3 * time.Second):
		t.Fatalf("timeout waiting for command execution")
	}

	fmt.Println("stdout", stdout)
	fmt.Println("stderr", stderr)
	fmt.Println("exitCode", exitCode)

	require.NoError(t, errres, "Failed to execute commands")
	require.Contains(t, stdout, "1.2.14", "Should find bun binary")

	// Test passed - OCI container filesystem is properly mounted and accessible!

}
