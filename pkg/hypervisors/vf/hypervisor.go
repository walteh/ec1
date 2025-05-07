package vf

import (
	"context"
	"log/slog"
	"sync"

	"github.com/Code-Hex/vz/v3"
	"github.com/walteh/ec1/pkg/hypervisors"
	"github.com/walteh/ec1/pkg/machines/bootloader"
	"gitlab.com/tozd/go/errors"
)

func NewHypervisor() *Hypervisor {
	return &Hypervisor{
		vms:    make(map[string]*VirtualMachine),
		notify: make(chan hypervisors.VirtualMachine),
	}
}

var _ hypervisors.Hypervisor = &Hypervisor{}

type Hypervisor struct {
	vms    map[string]*VirtualMachine
	mu     sync.Mutex
	notify chan hypervisors.VirtualMachine
}

func (hpv *Hypervisor) NewVirtualMachine(ctx context.Context, id string, opts hypervisors.NewVMOptions, bl bootloader.Bootloader) (hypervisors.VirtualMachine, error) {
	vfConfig, err := NewVirtualMachineConfiguration(ctx, id, &opts, bl)
	if err != nil {
		return nil, err
	}

	// pp.Println(vfConfig)

	valid, err := vfConfig.internal.Validate()
	if err != nil {
		return nil, errors.Errorf("validating vz virtual machine configuration: %w", err)
	}
	if !valid {
		return nil, errors.New("invalid vz virtual machine configuration")
	}

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

func (hpv *Hypervisor) OnCreate() <-chan hypervisors.VirtualMachine {
	return hpv.notify
}

// func (hpv *Hypervisor) ListenNetworkBlockDevices(ctx context.Context, vm hypervisors.VirtualMachine) error {
// 	panic("not implemented")
// }
