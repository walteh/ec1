package vf

// // MockObjcRuntime allows mocking of objc interactions

// // Create a real VM for testing
// func createTestVMWithMemory(t *testing.T, ctx context.Context, memoryMiB uint64) *VirtualMachine {
// 	hv := NewHypervisor()
// 	pp := puipui.NewPuipuiProvider()

// 	go func() {
// 		err := hypervisors.RunVirtualMachine(ctx, hv, pp, 2, strongunits.MiB(memoryMiB).ToBytes())
// 		if err != nil {
// 			t.Logf("problem running vm: %v", err)
// 			return
// 		}
// 	}()

// 	timeout := time.After(30 * time.Second)
// 	slog.DebugContext(ctx, "waiting for test VM")
// 	select {
// 	case <-timeout:
// 		t.Fatalf("Timed out waiting for test VM")
// 		return nil
// 	case vm := <-hv.notify:
// 		return vm
// 	}
// }

// // Mock bootloader for testing
// type mockBootloader struct{}

// func (m *mockBootloader) GetKernel() ([]byte, error) {
// 	return []byte{}, nil
// }

// func (m *mockBootloader) GetInitRD() ([]byte, error) {
// 	return []byte{}, nil
// }

// func (m *mockBootloader) GetCmdLine() (string, error) {
// 	return "console=hvc0", nil
// }

// func TestMemoryBalloonDevices(t *testing.T) {
// 	ctx := t.Context()
// 	ctx = testutils.SetupSlog(t, ctx)

// 	// // Skip on non-macOS platforms
// 	// if virtualizationFramework == 0 {
// 	// 	t.Skip("Skipping test as Virtualization framework is not available")
// 	// }

// 	// Create a real VM for testing
// 	vm := createTestVMWithMemory(t, ctx, 1024)
// 	if vm == nil {
// 		t.Skip("Could not create test VM")
// 		return
// 	}

// 	slog.DebugContext(ctx, "waiting for test VM to be running")

// 	if err := hypervisors.WaitForVMState(ctx, vm, hypervisors.VirtualMachineStateTypeRunning, nil); err != nil {
// 		t.Fatalf("virtualization error: %v", err)
// 	}

// 	// Now we can call the actual method
// 	devices, err := vm.MemoryBalloonDevices()

// 	// Just check that the call completes - results will depend on the actual environment
// 	if err != nil {
// 		t.Logf("MemoryBalloonDevices returned error: %v", err)
// 	} else {
// 		t.Logf("Found %d memory balloon devices", len(devices))
// 	}
// }

// func TestSetTargetVirtualMachineMemorySize(t *testing.T) {
// 	ctx := t.Context()
// 	ctx = testutils.SetupSlog(t, ctx)

// 	// Skip on non-macOS platforms
// 	// if virtualizationFramework == 0 {
// 	// 	t.Skip("Skipping test as Virtualization framework is not available")
// 	// }

// 	initialMemoryMiB := uint64(1024)
// 	targetMemoryMiB := uint64(2048)

// 	// Create a real VM for testing
// 	vm := createTestVMWithMemory(t, ctx, initialMemoryMiB)
// 	require.NotNil(t, vm)
// 	// Get devices
// 	devices, err := vm.MemoryBalloonDevices()
// 	require.NoError(t, err)
// 	require.NotNil(t, devices)
// 	require.Len(t, devices, 1)

// 	// Try to set memory size on the first device
// 	device := devices[0]

// 	// Get the current target memory size
// 	sizeBefore, err := device.GetTargetVirtualMachineMemorySize()
// 	require.NoError(t, err)
// 	slog.DebugContext(ctx, "sizeBefore", "size", sizeBefore, "initialMemoryMiB", initialMemoryMiB)
// 	require.Equal(t, sizeBefore, initialMemoryMiB)

// 	err = device.SetTargetVirtualMachineMemorySize(targetMemoryMiB) // 100 MB
// 	require.NoError(t, err)

// 	sizeAfter, err := device.GetTargetVirtualMachineMemorySize()
// 	require.NoError(t, err)

// 	require.Equal(t, sizeAfter, targetMemoryMiB)

// }

// func TestErrorHandling(t *testing.T) {
// 	// // Skip on non-macOS platforms
// 	// if virtualizationFramework == 0 {
// 	// 	t.Skip("Skipping test as Virtualization framework is not available")
// 	// }

// 	// Test case: Invalid device
// 	device := &VirtioTraditionalMemoryBalloonDevice{
// 		id: 0,
// 	}
// 	err := device.SetTargetVirtualMachineMemorySize(1024)
// 	assert.Error(t, err)
// 	assert.Contains(t, err.Error(), "invalid memory balloon device object")
// }
