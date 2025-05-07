package hypervisors

import (
	"context"

	"github.com/walteh/ec1/pkg/machines/bootloader"
)

type Hypervisor interface {
	NewVirtualMachine(ctx context.Context, id string, opts NewVMOptions, bl bootloader.Bootloader) (VirtualMachine, error)
	OnCreate() <-chan VirtualMachine
}
