package vf_test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/Code-Hex/vz/v3"
	"github.com/containers/common/pkg/strongunits"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/walteh/ec1/pkg/images/puipui"
	"github.com/walteh/ec1/pkg/testing/tlog"
	"github.com/walteh/ec1/pkg/vmm"
	"github.com/walteh/ec1/pkg/vmm/vf"
)

// MockObjcRuntime allows mocking of objc interactions

// Create a real VM for testing
func setupPuipuiVM(t *testing.T, ctx context.Context, memory strongunits.MiB) (*vmm.RunningVM[*vf.VirtualMachine], vmm.VMIProvider) {
	hv := vf.NewHypervisor()
	pp := puipui.NewPuipuiProvider()

	slog.DebugContext(ctx, "running vm", "memory", memory, "memory.ToBytes()", memory.ToBytes())

	rvm, err := vmm.RunVirtualMachine(ctx, hv, pp, 2, memory.ToBytes())
	require.NoError(t, err)

	go func() {
		t.Logf("vm running,waiting for vm to stop")
		err := rvm.Wait()
		assert.NoError(t, err)
	}()

	return rvm, pp

	// timeout := time.After(30 * time.Second)
	// slog.DebugContext(ctx, "waiting for test VM")
	// select {
	// case <-timeout:
	// 	t.Fatalf("Timed out waiting for test VM")
	// 	return nil
	// case vm := <-hv.notify:
	// 	return rvm.VM()
	// case err := <-problemch:
	// 	t.Fatalf("problem running vm: %v", err)
	// 	return nil
	// }
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
