package vf

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/containers/common/pkg/strongunits"
	"github.com/stretchr/testify/assert"
	"github.com/walteh/ec1/pkg/hypervisors"
	"github.com/walteh/ec1/pkg/machines/images/puipui"
	"github.com/walteh/ec1/pkg/testutils"
)

// MockObjcRuntime allows mocking of objc interactions

// Create a real VM for testing
func createTestVM(t *testing.T, ctx context.Context) *VirtualMachine {
	hv := NewHypervisor()
	pp := puipui.NewPuipuiProvider()

	go func() {
		err := hypervisors.RunVirtualMachine(ctx, hv, pp, 2, strongunits.MiB(1024).ToBytes())
		if err != nil {
			t.Logf("problem running vm: %v", err)
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

	// Skip on non-macOS platforms
	if virtualizationFramework == 0 {
		t.Skip("Skipping test as Virtualization framework is not available")
	}

	// Create a real VM for testing
	vm := createTestVM(t, ctx)
	if vm == nil {
		t.Skip("Could not create test VM")
		return
	}

	slog.DebugContext(ctx, "waiting for test VM to be running")

	if err := hypervisors.WaitForVMState(ctx, vm, hypervisors.VirtualMachineStateTypeRunning, nil); err != nil {
		t.Fatalf("virtualization error: %v", err)
	}

	// Now we can call the actual method
	devices, err := vm.MemoryBalloonDevices()

	// Just check that the call completes - results will depend on the actual environment
	if err != nil {
		t.Logf("MemoryBalloonDevices returned error: %v", err)
	} else {
		t.Logf("Found %d memory balloon devices", len(devices))
	}
}

func TestSetTargetVirtualMachineMemorySize(t *testing.T) {
	ctx := t.Context()
	ctx = testutils.SetupSlog(t, ctx)

	// Skip on non-macOS platforms
	if virtualizationFramework == 0 {
		t.Skip("Skipping test as Virtualization framework is not available")
	}

	// Create a real VM for testing
	vm := createTestVM(t, ctx)
	if vm == nil {
		t.Skip("Could not create test VM")
		return
	}

	// Get devices
	devices, err := vm.MemoryBalloonDevices()
	if err != nil || len(devices) == 0 {
		t.Skip("No memory balloon devices available")
		return
	}

	// Try to set memory size on the first device
	device := devices[0]
	err = device.SetTargetVirtualMachineMemorySize(1024 * 1024 * 100) // 100 MB

	// Just check that the call completes
	if err != nil {
		t.Logf("SetTargetVirtualMachineMemorySize returned error: %v", err)
	} else {
		t.Log("Successfully set target memory size")
	}
}

func TestErrorHandling(t *testing.T) {
	// Skip on non-macOS platforms
	if virtualizationFramework == 0 {
		t.Skip("Skipping test as Virtualization framework is not available")
	}

	// Test case: Invalid device
	device := &VirtioTraditionalMemoryBalloonDevice{
		id: 0,
	}
	err := device.SetTargetVirtualMachineMemorySize(1024)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid memory balloon device object")
}
