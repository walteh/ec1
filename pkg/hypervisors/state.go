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
	Vcpus        uint
	Memory       strongunits.B
	Devices      []virtio.VirtioDevice
	Provisioners []Provisioner
}

func ProvisionerForType[T Provisioner](opts *NewVMOptions) (T, bool) {
	for _, provisioner := range opts.Provisioners {
		if p, ok := provisioner.(T); ok {
			return p, true
		}
	}
	return *new(T), false
}

type VirtualMachineStateType string

const (
	VirtualMachineStateTypeUnknown  VirtualMachineStateType = "unknown"
	VirtualMachineStateTypeRunning  VirtualMachineStateType = "running"
	VirtualMachineStateTypeStarting VirtualMachineStateType = "starting"
	VirtualMachineStateTypeStopping VirtualMachineStateType = "stopping"
	VirtualMachineStateTypeStopped  VirtualMachineStateType = "stopped"
	VirtualMachineStateTypePaused   VirtualMachineStateType = "paused"
	VirtualMachineStateTypeError    VirtualMachineStateType = "error"
)

type VirtualMachineStateChange struct {
	StateType VirtualMachineStateType
	Metadata  map[string]string
}

func WaitForVMState(ctx context.Context, vm VirtualMachine, state VirtualMachineStateType, timeout <-chan time.Time) error {
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGPIPE)

	slog.DebugContext(ctx, "waiting for VM state", "state", state, "current state", vm.CurrentState())

	notifier := vm.StateChangeNotify(ctx)

	for {
		select {
		case s := <-signalCh:
			slog.DebugContext(ctx, "ignoring signal", "signal", s)
		case newState := <-notifier:

			slog.DebugContext(ctx, "VM state changed", "state", newState.StateType, "metadata", newState.Metadata)

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
