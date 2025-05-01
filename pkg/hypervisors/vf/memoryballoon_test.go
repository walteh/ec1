package vf

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/Code-Hex/vz/v3"
	"github.com/containers/common/pkg/strongunits"
	"github.com/stretchr/testify/require"
	"github.com/walteh/ec1/pkg/hypervisors"
	"github.com/walteh/ec1/pkg/machines/images/puipui"
	"github.com/walteh/ec1/pkg/testutils"
)

// MockObjcRuntime allows mocking of objc interactions

// Create a real VM for testing
func createTestVMWithMemory(t *testing.T, ctx context.Context, memory strongunits.MiB) *VirtualMachine {
	hv := NewHypervisor()
	pp := puipui.NewPuipuiProvider()

	problemch := make(chan error)

	slog.DebugContext(ctx, "running vm", "memory", memory, "memory.ToBytes()", memory.ToBytes())

	go func() {
		err := hypervisors.RunVirtualMachine(ctx, hv, pp, 2, memory.ToBytes())
		if err != nil {
			problemch <- err
			return
		}
	}()

	timeout := time.After(30 * time.Second)
	slog.DebugContext(ctx, "waiting for test VM")
	select {
	case <-timeout:
		t.Fatalf("Timed out waiting for test VM")
		return nil
	case vm := <-hv.notify:
		return vm
	case err := <-problemch:
		t.Fatalf("problem running vm: %v", err)
		return nil
	}
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
	ctx := t.Context()
	ctx = testutils.SetupSlog(t, ctx)

	// // Skip on non-macOS platforms
	// if virtualizationFramework == 0 {
	// 	t.Skip("Skipping test as Virtualization framework is not available")
	// }

	// Create a real VM for testing
	vm := createTestVMWithMemory(t, ctx, 1024)
	if vm == nil {
		t.Skip("Could not create test VM")
		return
	}

	slog.DebugContext(ctx, "waiting for test VM to be running")

	if err := hypervisors.WaitForVMState(ctx, vm, hypervisors.VirtualMachineStateTypeRunning, nil); err != nil {
		t.Fatalf("virtualization error: %v", err)
	}

	// Now we can call the actual method
	devices := vm.vzvm.MemoryBalloonDevices()

	// Just check that the call completes - results will depend on the actual environment
	if len(devices) == 0 {
		t.Fatalf("No memory balloon devices found")
	} else {
		t.Logf("Found %d memory balloon devices", len(devices))
	}
}

func TestSetTargetVirtualMachineMemorySize(t *testing.T) {
	ctx := t.Context()
	ctx = testutils.SetupSlog(t, ctx)

	// Skip on non-macOS platforms
	// if virtualizationFramework == 0 {
	// 	t.Skip("Skipping test as Virtualization framework is not available")
	// }

	startingMemory := strongunits.MiB(512)
	targetMemory := strongunits.MiB(300)

	// Create a real VM for testing
	vm := createTestVMWithMemory(t, ctx, startingMemory)
	require.NotNil(t, vm)
	// Get devices
	devices := vm.vzvm.MemoryBalloonDevices()

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
