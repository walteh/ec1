package hypervisors

import (
	"context"
	"net"

	"github.com/walteh/ec1/pkg/machines/virtio"
)

type VirtualMachine interface {
	ID() string
	// StartVsock(ctx context.Context) error
	VSockConnect(ctx context.Context, port uint32) (net.Conn, error)
	VSockListen(ctx context.Context, port uint32) (net.Listener, error)
	CurrentState() VirtualMachineStateType
	StateChangeNotify() <-chan VirtualMachineStateChange
	Start(ctx context.Context) error
	Devices() []virtio.VirtioDevice
	StartGraphicApplication(width float64, height float64) error
}
