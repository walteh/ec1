package vmm

import (
	"context"
	"io"
	"log/slog"
	"time"

	"github.com/mholt/archives"
	"gitlab.com/tozd/go/errors"

	"github.com/walteh/ec1/pkg/bootloader"
	"github.com/walteh/ec1/pkg/streamexec"
	"github.com/walteh/ec1/pkg/streamexec/protocol"
	"github.com/walteh/ec1/pkg/streamexec/transport"
)

const (
	ExecVSockPort = 2019
)

type Hypervisor[VM VirtualMachine] interface {
	NewVirtualMachine(ctx context.Context, id string, opts NewVMOptions, bl bootloader.Bootloader) (VM, error)
	OnCreate() <-chan VM
	EncodeLinuxInitramfs(ctx context.Context, initramfs io.ReadCloser) (io.ReadCloser, error)
	EncodeLinuxKernel(ctx context.Context, kernel io.ReadCloser) (io.ReadCloser, error)
	EncodeLinuxRootfs(ctx context.Context, rootfs io.ReadCloser) (io.ReadCloser, error)
	InitramfsCompression() archives.Compression
}

type RunningVM[VM VirtualMachine] struct {
	// streamExecReady bool
	manager      *VSockManager
	streamexec   *streamexec.Client
	portOnHostIP uint16
	wait         <-chan error
	vm           VM
	// connStatus      <-chan VSockManagerState
	start time.Time
}

func NewRunningVM[VM VirtualMachine](ctx context.Context, vm VM, portOnHostIP uint16, start time.Time, wait <-chan error) *RunningVM[VM] {

	transporz := NewVSockManager(func(ctx context.Context) (io.ReadWriteCloser, error) {
		return vm.VSockConnect(ctx, uint32(ExecVSockPort))
	})

	tfunc := transport.NewFunctionTransport(func() (io.ReadWriteCloser, error) {
		slog.Info("dialing vm transport")
		conn, err := transporz.Dial(ctx)
		if err != nil {
			slog.Error("failed to dial vm transport", "error", err)
			return nil, err
		}
		slog.Info("dialed vm transport")
		return conn, nil
	}, nil)

	client := streamexec.NewClient(tfunc, func(conn io.ReadWriter) protocol.Protocol {
		return protocol.NewFramedProtocol(conn)
	})

	go func() {
		slog.Info("dialing vm")
		err := client.Connect(ctx)
		if err != nil {
			slog.Error("failed to connect to vm", "error", err)
		} else {
			slog.Info("connected to vm")
		}

	}()

	// connStatus := transporz.AddStateNotifier()

	return &RunningVM[VM]{
		start:   start,
		vm:      vm,
		manager: transporz,
		// connStatus:      connStatus,
		portOnHostIP: portOnHostIP,
		wait:         wait,
		// streamExecReady: false,
		streamexec: client,
	}
}

func (r *RunningVM[VM]) WaitOnVmStopped() error {
	return <-r.wait
}

func (r *RunningVM[VM]) WaitOnVMReadyToExec() <-chan struct{} {
	ch := make(chan struct{})

	if r.manager.State() == StateConnected {
		close(ch)
		return ch
	}
	check := r.manager.AddStateNotifier()
	go func() {
		defer close(check)
		for {
			select {
			case <-check:
				if r.manager.State() == StateConnected {
					close(ch)
				}
			}
		}
	}()
	return ch
}

func (r *RunningVM[VM]) VM() VM {
	return r.vm
}

func (r *RunningVM[VM]) PortOnHostIP() uint16 {
	return r.portOnHostIP
}

func (r *RunningVM[VM]) Exec(ctx context.Context, command string) (stdout []byte, stderr []byte, errorcode []byte, err error) {
	if r.manager.State() != StateConnected {
		return nil, nil, nil, errors.New("stream exec not ready")
	}
	return r.streamexec.ExecuteCommand(ctx, command)
}
