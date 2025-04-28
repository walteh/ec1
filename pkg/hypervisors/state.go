package hypervisors

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/containers/common/pkg/strongunits"
	"github.com/walteh/ec1/pkg/machines/virtio"
)

type NewVMOptions struct {
	Vcpus   uint
	Memory  strongunits.B
	Devices []virtio.VirtioDevice
}

type Hypervisor interface {
	NewVirtualMachine(ctx context.Context, opts NewVMOptions) (VirtualMachine, error)
	ListenNetworkBlockDevices(ctx context.Context, vm VirtualMachine) error
}

type VirtualMachineStateType string

const (
	VirtualMachineStateTypeUnknown VirtualMachineStateType = "unknown"
	VirtualMachineStateTypeRunning VirtualMachineStateType = "running"
	VirtualMachineStateTypeStopped VirtualMachineStateType = "stopped"
	VirtualMachineStateTypeError   VirtualMachineStateType = "error"
)

type VirtualMachineStateChange struct {
	StateType VirtualMachineStateType
	Metadata  map[string]string
}

func WaitForVMState(ctx context.Context, vm VirtualMachine, state VirtualMachineStateType, timeout <-chan time.Time) error {
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGPIPE)

	slog.DebugContext(ctx, "waiting for VM state", "state", state, "current state", vm.CurrentState())

	for {
		select {
		case s := <-signalCh:
			slog.DebugContext(ctx, "ignoring signal", "signal", s)
		case newState := <-vm.StateChangeNotify():

			slog.DebugContext(ctx, "VM state changed", "state", newState)

			if newState.StateType == state {
				return nil
			}
			if newState.StateType == VirtualMachineStateTypeError {
				return fmt.Errorf("hypervisor virtualization error")
			}
		case <-timeout:
			return fmt.Errorf("timeout waiting for VM state")
		}
	}
}
