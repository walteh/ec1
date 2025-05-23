package vf_test

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Code-Hex/vz/v3"
	"github.com/containers/common/pkg/strongunits"
	"github.com/fatih/color"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	slogctx "github.com/veqryn/slog-context"

	"github.com/walteh/ec1/pkg/host"
	"github.com/walteh/ec1/pkg/images/coreos"
	"github.com/walteh/ec1/pkg/images/puipui"
	"github.com/walteh/ec1/pkg/testing/tlog"
	"github.com/walteh/ec1/pkg/vmm"
	"github.com/walteh/ec1/pkg/vmm/vf"
)

// MockObjcRuntime allows mocking of objc interactions

// Create a real VM for testing
func setupVM(t *testing.T, ctx context.Context, memory strongunits.MiB, provider vmm.VMIProvider) (*vmm.RunningVM[*vf.VirtualMachine], vmm.VMIProvider) {
	hv := vf.NewHypervisor()

	ctx = slogctx.WithGroup(ctx, "test-vm-setup")

	slog.DebugContext(ctx, "running vm", "memory", memory, "memory.ToBytes()", memory.ToBytes())

	rvm, err := vmm.RunVirtualMachine(ctx, hv, provider, 2, memory.ToBytes())
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

	return rvm, provider
}

func setupPuipuiVM(t *testing.T, ctx context.Context, memory strongunits.MiB) (*vmm.RunningVM[*vf.VirtualMachine], vmm.VMIProvider) {
	return setupVM(t, ctx, memory, puipui.NewPuipuiProvider())
}

func setupCoreOSVM(t *testing.T, ctx context.Context, memory strongunits.MiB) (*vmm.RunningVM[*vf.VirtualMachine], vmm.VMIProvider) {
	return setupVM(t, ctx, memory, coreos.NewProvider())
}

func buildDiffReport(t *testing.T, title string, header1 string, header2 string, diffContent string) string {
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

func catConsoleFile(t *testing.T, ctx context.Context, rvm vmm.VirtualMachine) {
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

	t.Log(buildDiffReport(t, "console.log", "", "", string(content)))
}

// Mock bootloader for testing
type mockBootloader struct{}

func (m *mockBootloader) GetKernel() ([]byte, error) {
	return []byte{}, nil
}

func (m *mockBootloader) GetInitRD() ([]byte, error) {
	return []byte{}, nil
}

func (m *mockBootloader) GetCmdLine() (string, error) {
	return "console=hvc0", nil
}

func TestMemoryBalloonDevices(t *testing.T) {
	ctx := tlog.SetupSlogForTest(t)

	// // Skip on non-macOS platforms
	// if virtualizationFramework == 0 {
	// 	t.Skip("Skipping test as Virtualization framework is not available")
	// }

	// Create a real VM for testing
	rvm, pp := setupPuipuiVM(t, ctx, 1024)
	require.NotNil(t, rvm)
	require.NotNil(t, pp)

	if rvm == nil {
		t.Skip("Could not create test VM")
		return
	}

	slog.DebugContext(ctx, "waiting for test VM to be running")

	if err := vmm.WaitForVMState(ctx, rvm.VM(), vmm.VirtualMachineStateTypeRunning, nil); err != nil {
		t.Fatalf("virtualization error: %v", err)
	}

	// Now we can call the actual method
	devices := rvm.VM().VZ().MemoryBalloonDevices()

	// Just check that the call completes - results will depend on the actual environment
	if len(devices) == 0 {
		t.Fatalf("No memory balloon devices found")
	} else {
		t.Logf("Found %d memory balloon devices", len(devices))
	}
}

func TestSetTargetVirtualMachineMemorySize(t *testing.T) {
	ctx := tlog.SetupSlogForTest(t)

	// Skip on non-macOS platforms
	// if virtualizationFramework == 0 {
	// 	t.Skip("Skipping test as Virtualization framework is not available")
	// }

	startingMemory := strongunits.MiB(512)
	targetMemory := strongunits.MiB(300)

	// Create a real VM for testing
	rvm, pp := setupPuipuiVM(t, ctx, startingMemory)
	require.NotNil(t, rvm)
	require.NotNil(t, pp)
	// Get devices
	devices := rvm.VM().VZ().MemoryBalloonDevices()

	require.NotNil(t, devices)
	require.Equal(t, len(devices), 1)

	// Try to set memory size on the first device
	device := devices[0]

	trad, ok := device.(*vz.VirtioTraditionalMemoryBalloonDevice)
	require.True(t, ok)

	// Get the current target memory size
	sizeBefore := trad.GetTargetVirtualMachineMemorySize()
	slog.DebugContext(ctx, "sizeBefore", "sizeBefore", sizeBefore, "startingMemory", startingMemory)
	require.Equal(t, sizeBefore, uint64(startingMemory.ToBytes()))

	trad.SetTargetVirtualMachineMemorySize(uint64(targetMemory.ToBytes())) // 100 MB

	sizeAfter := trad.GetTargetVirtualMachineMemorySize()

	require.Equal(t, sizeAfter, uint64(targetMemory.ToBytes()))

}
