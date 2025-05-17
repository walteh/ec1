package vf

import (
	"context"
	"fmt"
	"net"

	"github.com/Code-Hex/vz/v3"
	"github.com/containers/common/pkg/strongunits"
	"gitlab.com/tozd/go/errors"

	"github.com/walteh/ec1/pkg/machines/virtio"
	"github.com/walteh/ec1/pkg/vmm"
)

type MemoryBalloonDevice struct {
}

var _ vmm.VirtualMachine = &VirtualMachine{}

func vzStateToHypervisorState(state vz.VirtualMachineState) vmm.VirtualMachineStateType {
	switch state {
	case vz.VirtualMachineStateRunning:
		return vmm.VirtualMachineStateTypeRunning
	case vz.VirtualMachineStatePaused:
		return vmm.VirtualMachineStateTypePaused
	case vz.VirtualMachineStateStarting:
		return vmm.VirtualMachineStateTypeStarting
	case vz.VirtualMachineStateStopping:
		return vmm.VirtualMachineStateTypeStopping
	case vz.VirtualMachineStateStopped:
		return vmm.VirtualMachineStateTypeStopped
	case vz.VirtualMachineStateError:
		return vmm.VirtualMachineStateTypeError
	default:
		return vmm.VirtualMachineStateTypeUnknown
	}
}

type VirtualMachine struct {
	vzvm          *vz.VirtualMachine
	configuration *VirtualMachineConfiguration
}

// func (vm *VirtualMachine) objcPtr() uintptr {
// 	objcVM := reflect.ValueOf(vm.vzvm).Pointer()
// 	// objcVMp, ok := objcVM.(unsafe.Pointer)
// 	// if !ok {
// 	// 	panic("objcVM is not a pointer: " + fmt.Sprintf("%T", objcVM))
// 	// }

// 	return objcVM
// }

// func (vm *VirtualMachine) BalloonDevice() *vz.VirtioTraditionalMemoryBalloonDevice {
// 	devices := vm.vzvm.MemoryBalloonDevices()
// 	if len(devices) == 0 {
// 		return nil
// 	}
// 	return devices[0].(*vz.VirtioTraditionalMemoryBalloonDevice)
// }

// // SetMemoryBalloonTargetSize adjusts the size of memory available to the VM
// // by inflating or deflating the memory balloon.
// // targetBytes is the amount of memory the guest OS should have access to.
// // Note that the target memory should be less than the total VM memory.
// func (vm *VirtualMachine) SetMemoryBalloonTargetSize(targetBytes strongunits.B) error {
// 	if vm.CurrentState() != vmm.VirtualMachineStateTypeRunning {
// 		return fmt.Errorf("VM must be running to adjust memory balloon")
// 	}

// 	balloonDevice := vm.BalloonDevice()
// 	if balloonDevice == nil {
// 		return fmt.Errorf("no memory balloon device found in VM configuration")
// 	}

// 	// Calculate total VM memory from config
// 	totalMemory := strongunits.B(vm.configuration.memorySize)
// 	if targetBytes >= totalMemory {
// 		return fmt.Errorf("target memory size (%s) must be less than total VM memory (%s)", targetBytes, totalMemory)
// 	}

// 	// Set the target memory size
// 	balloonDevice.SetTargetVirtualMachineMemory(uint64(targetBytes.ToBytes()))

// 	return nil
// }

// MemoryUsage implements vmm.VirtualMachine.
// func (vm *VirtualMachine) MemoryUsage() strongunits.B {
// 	// For now, just return the configured memory size
// 	// In a real implementation, you would get this from the balloon device
// 	return strongunits.B(vm.configuration.memorySize)
// }

// // VCPUsUsage implements vmm.VirtualMachine.
// func (vm *VirtualMachine) VCPUsUsage() float64 {
// 	return vm.vzvm.VCPUsUsage()
// }

// CanHardStop implements vmm.VirtualMachine.
func (vm *VirtualMachine) CanHardStop(ctx context.Context) bool {
	return vm.vzvm.CanStop()
}

// CanRequestStop implements vmm.VirtualMachine.
func (vm *VirtualMachine) CanRequestStop(ctx context.Context) bool {
	return vm.vzvm.CanRequestStop()
}

// HardStop implements vmm.VirtualMachine.
func (vm *VirtualMachine) HardStop(ctx context.Context) error {
	return vm.Stop(ctx)
}

// CurrentState implements vmm.VirtualMachine.
func (vm *VirtualMachine) CurrentState() vmm.VirtualMachineStateType {
	return vzStateToHypervisorState(vm.vzvm.State())
}

// Devices implements vmm.VirtualMachine.
func (vm *VirtualMachine) Devices() []virtio.VirtioDevice {
	return vm.configuration.newVMOpts.Devices
}

// ID implements vmm.VirtualMachine.
func (vm *VirtualMachine) ID() string {
	return vm.configuration.id
}

func (vm *VirtualMachine) GetVSockDevice() (*vz.VirtioSocketDevice, error) {
	devices := vm.vzvm.SocketDevices()
	if len(devices) == 0 {
		return nil, fmt.Errorf("no socket device found")
	}
	return devices[0], nil
}

func (vm *VirtualMachine) GetConsoleDevice() (vz.ConsoleDevice, error) {
	devices := vm.vzvm.ConsoleDevices()
	if len(devices) == 0 {
		return nil, fmt.Errorf("no console device found")
	}
	return devices[0], nil
}

// StartGraphicApplication implements vmm.VirtualMachine.
// Subtle: this method shadows the method (*VirtualMachine).StartGraphicApplication of VirtualMachine.VirtualMachine.
func (vm *VirtualMachine) StartGraphicApplication(width float64, height float64) error {
	return vm.vzvm.StartGraphicApplication(width, height)
}

// StateChangeNotify implements vmm.VirtualMachine.
func (vm *VirtualMachine) StateChangeNotify(ctx context.Context) <-chan vmm.VirtualMachineStateChange {
	stateChangeNotify := make(chan vmm.VirtualMachineStateChange)
	go func() {
		for {
			select {
			case <-ctx.Done():
				// slog.DebugContext(ctx, "state change notify context done")
				return
			case yep := <-vm.vzvm.StateChangedNotify():
				// slog.DebugContext(ctx, "state change notify start", "state", yep)
				stateChangeNotify <- vmm.VirtualMachineStateChange{
					StateType: vzStateToHypervisorState(yep),
					Metadata: map[string]string{
						"raw_state": yep.String(),
					},
				}
				// slog.DebugContext(ctx, "state change notify end", "state", yep)
			}
		}
	}()
	return stateChangeNotify
}

// VSockConnect implements vmm.VirtualMachine.
func (vm *VirtualMachine) VSockConnect(ctx context.Context, port uint32) (net.Conn, error) {
	vsockDev, err := vm.GetVSockDevice()
	if err != nil {
		return nil, errors.Errorf("getting vsock device: %w", err)
	}
	return vsockDev.Connect(port)
}

// VSockListen implements vmm.VirtualMachine.
func (vm *VirtualMachine) VSockListen(ctx context.Context, port uint32) (net.Listener, error) {
	vsockDev, err := vm.GetVSockDevice()
	if err != nil {
		return nil, errors.Errorf("getting vsock device: %w", err)
	}
	return vsockDev.Listen(port)
}

func (vm *VirtualMachine) CanStart(_ context.Context) bool {
	return vm.vzvm.CanStart()
}

func (vm *VirtualMachine) CanStop(_ context.Context) bool {
	return vm.vzvm.CanStop()
}

func (vm *VirtualMachine) CanPause(_ context.Context) bool {
	return vm.vzvm.CanPause()
}

func (vm *VirtualMachine) CanResume(_ context.Context) bool {
	return vm.vzvm.CanResume()
}

func (vm *VirtualMachine) Pause(_ context.Context) error {
	return vm.vzvm.Pause()
}

func (vm *VirtualMachine) Resume(_ context.Context) error {
	return vm.vzvm.Resume()
}

func (vm *VirtualMachine) Stop(_ context.Context) error {
	return vm.vzvm.Stop()
}

func (vm *VirtualMachine) RequestStop(_ context.Context) (bool, error) {
	return vm.vzvm.RequestStop()
}

func (vm *VirtualMachine) GetMemoryBalloonTargetSize(_ context.Context) (strongunits.B, error) {
	balloonDevices := vm.vzvm.MemoryBalloonDevices()
	if len(balloonDevices) == 0 {
		return 0, errors.New("no memory balloon devices found")
	}

	trad, ok := balloonDevices[0].(*vz.VirtioTraditionalMemoryBalloonDevice)
	if !ok {
		return 0, errors.New("memory balloon device is not a VirtioTraditionalMemoryBalloonDevice")
	}

	size := trad.GetTargetVirtualMachineMemorySize()

	return strongunits.B(size), nil
}

func (vm *VirtualMachine) SetMemoryBalloonTargetSize(ctx context.Context, targetBytes strongunits.B) error {
	balloonDevices := vm.vzvm.MemoryBalloonDevices()
	if len(balloonDevices) == 0 {
		return errors.New("no memory balloon devices found")
	}

	trad, ok := balloonDevices[0].(*vz.VirtioTraditionalMemoryBalloonDevice)
	if !ok {
		return errors.New("memory balloon device is not a VirtioTraditionalMemoryBalloonDevice")
	}

	trad.SetTargetVirtualMachineMemorySize(uint64(targetBytes.ToBytes()))

	getSize, err := vm.GetMemoryBalloonTargetSize(ctx)
	if err != nil {
		return errors.Errorf("getting memory balloon target size: %w", err)
	}

	if getSize != targetBytes {
		return errors.Errorf("validating memory balloon target size update: expected %d, got %d", uint64(targetBytes), uint64(getSize))
	}

	return nil
}

func (vm *VirtualMachine) Opts() *vmm.NewVMOptions {
	return vm.configuration.newVMOpts
}

// func (vm *VirtualMachine) RootVSockAddress() string {
// 	vsockDev, err := vm.GetVSockDevice()
// 	if err != nil {
// 		return ""
// 	}

// 	return vsockDev.
// }
