package vf

import (
	"context"
	"io"
	"log/slog"
	"sync"

	"github.com/Code-Hex/vz/v3"
	"github.com/mholt/archives"
	"gitlab.com/tozd/go/errors"

	"github.com/walteh/ec1/pkg/bootloader"
	"github.com/walteh/ec1/pkg/ext/archivesx"
	"github.com/walteh/ec1/pkg/magic"
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

func (hpv *Hypervisor) EncodeLinuxInitramfs(ctx context.Context, initramfs io.ReadCloser) (io.ReadCloser, error) {

	// convert to gzip
	arc, err := archivesx.CreateCompressorPipeline(ctx, &archives.Gz{}, initramfs)
	if err != nil {
		return nil, errors.Errorf("creating compressor pipeline: %w", err)
	}

	// counter := iox.NewReadCounter(arc)
	// counter.SetDebug(true)

	return arc, nil
}

func (hpv *Hypervisor) EncodeLinuxKernel(ctx context.Context, kernel io.ReadCloser) (io.ReadCloser, error) {
	// ensure kernel is valid
	validationReader, err := magic.ARM64LinuxKernelValidationReader(kernel)
	if err != nil {
		return nil, errors.Errorf("checking linux kernel: %w", err)
	}
	return validationReader, nil
}

func (hpv *Hypervisor) EncodeLinuxRootfs(ctx context.Context, rootfs io.ReadCloser) (io.ReadCloser, error) {
	return rootfs, nil
}

func (hpv *Hypervisor) InitramfsCompression() archives.Compression {
	return &archives.Gz{}
}
