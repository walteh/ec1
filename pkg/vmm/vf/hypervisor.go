package vf

import (
	"context"
	"log/slog"
	"sync"

	"github.com/Code-Hex/vz/v3"
	"gitlab.com/tozd/go/errors"

	"github.com/walteh/ec1/pkg/bootloader"
	"github.com/walteh/ec1/pkg/vmm"
)

func NewHypervisor() vmm.Hypervisor[*VirtualMachine] {
	return &Hypervisor{
		vms:    make(map[string]*VirtualMachine),
		notify: make(chan *VirtualMachine),
	}
}

var _ vmm.Hypervisor[*VirtualMachine] = &Hypervisor{}

type Hypervisor struct {
	vms    map[string]*VirtualMachine
	mu     sync.Mutex
	notify chan *VirtualMachine
}

func (hpv *Hypervisor) NewVirtualMachine(ctx context.Context, id string, opts vmm.NewVMOptions, bl bootloader.Bootloader) (*VirtualMachine, error) {
	vfConfig, err := NewVirtualMachineConfiguration(ctx, id, &opts, bl)
	if err != nil {
		return nil, err
	}

	slog.InfoContext(ctx, "validating vz virtual machine configuration")

	valid, err := vfConfig.internal.Validate()
	if err != nil {
		return nil, errors.Errorf("validating vz virtual machine configuration: %w", err)
	}
	if !valid {
		return nil, errors.New("invalid vz virtual machine configuration")
	}

	slog.InfoContext(ctx, "creating vz virtual machine")

	vzVM, err := vz.NewVirtualMachine(vfConfig.internal)
	if err != nil {
		return nil, errors.Errorf("creating vz virtual machine: %w", err)
	}


	vm := &VirtualMachine{
		configuration: vfConfig,
		vzvm:          vzVM,
	}

	hpv.mu.Lock()
	hpv.vms[id] = vm
	hpv.mu.Unlock()

	slog.DebugContext(ctx, "notifying hypervisor", "vm", vm)
	go func() {
		hpv.notify <- vm
	}()

	slog.DebugContext(ctx, "returning vm", "vm", vm)
	return vm, nil
}

func (hpv *Hypervisor) OnCreate() <-chan *VirtualMachine {
	return hpv.notify
}

// func (hpv *Hypervisor) ListenNetworkBlockDevices(ctx context.Context, vm vmm.VirtualMachine) error {
// 	panic("not implemented")
// }
