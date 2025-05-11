package hypervisors

import (
	"context"
	"time"

	"github.com/walteh/ec1/pkg/machines/bootloader"
)

type Hypervisor[VM VirtualMachine] interface {
	NewVirtualMachine(ctx context.Context, id string, opts NewVMOptions, bl bootloader.Bootloader) (VM, error)
	OnCreate() <-chan VM
}

type RunningVM[VM VirtualMachine] struct {
	portOnHostIP uint16
	wait         <-chan error
	vm           VM
	start        time.Time
}

func (r *RunningVM[VM]) Wait() error {
	return <-r.wait
}

func (r *RunningVM[VM]) VM() VM {
	return r.vm
}

func (r *RunningVM[VM]) PortOnHostIP() uint16 {
	return r.portOnHostIP
}
