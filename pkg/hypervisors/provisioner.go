package hypervisors

import (
	"context"

	"github.com/walteh/ec1/pkg/machines/virtio"
)

type Provisioner interface {
	VirtioDevices(ctx context.Context) ([]virtio.VirtioDevice, error)
}

type BootProvisioner interface {
	Provisioner
	RunDuringBoot(ctx context.Context, vm VirtualMachine) error
}

type RuntimeProvisioner interface {
	Provisioner
	RunDuringRuntime(ctx context.Context, vm VirtualMachine) error
}
