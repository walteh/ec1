package vf

import (
	"context"
	"io"
	"log/slog"
	"sync"

	"github.com/Code-Hex/vz/v3"
	"github.com/mholt/archives"
	"gitlab.com/tozd/go/errors"

	"github.com/walteh/ec1/pkg/ext/archivesx"
	"github.com/walteh/ec1/pkg/magic"
	"github.com/walteh/ec1/pkg/virtio"
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

func (hpv *Hypervisor) NewVirtualMachine(ctx context.Context, id string, opts *vmm.NewVMOptions, bl vmm.Bootloader) (*VirtualMachine, error) {
	cfg, vzbl, err := hpv.buildConfig(ctx, opts, bl)
	if err != nil {
		return nil, err
	}

	slog.InfoContext(ctx, "validating vz virtual machine configuration")

	applier, err := NewVzVirtioDeviceApplier(ctx, cfg, bl)
	if err != nil {
		return nil, errors.Errorf("creating vz virtio device applier: %w", err)
	}

	if err := virtio.ApplyDevices(ctx, applier, opts.Devices); err != nil {
		return nil, errors.Errorf("applying virtio devices: %w", err)
	}

	slog.InfoContext(ctx, "creating vz virtual machine")

	vzVM, err := vz.NewVirtualMachine(cfg)
	if err != nil {
		return nil, errors.Errorf("creating vz virtual machine: %w", err)
	}

	vm := &VirtualMachine{
		id:            id,
		bootLoader:    vzbl,
		configuration: cfg,
		vzvm:          vzVM,
		opts:          opts,
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

func (hpv *Hypervisor) buildConfig(ctx context.Context, opts *vmm.NewVMOptions, bl vmm.Bootloader) (*vz.VirtualMachineConfiguration, vz.BootLoader, error) {
	slog.DebugContext(ctx, "Creating virtual machine configuration")
	vzBootloader, err := toVzBootloader(bl)
	if err != nil {
		return nil, nil, errors.Errorf("converting bootloader to vz bootloader: %w", err)
	}

	vzVMConfig, err := vz.NewVirtualMachineConfiguration(vzBootloader, uint(opts.Vcpus), uint64(opts.Memory.ToBytes()))
	if err != nil {
		return nil, nil, errors.Errorf("creating vz virtual machine configuration: %w", err)
	}

	return vzVMConfig, vzBootloader, nil
}

func (hpv *Hypervisor) EncodeLinuxInitramfs(ctx context.Context, initramfs io.Reader) (io.ReadCloser, error) {
	// Use optimized gzip settings for speed over compression ratio
	arc, err := archivesx.CreateCompressorPipeline(ctx, &archives.Gz{
		CompressionLevel:   1,    // Fastest compression level
		Multithreaded:      true, // Enable multithreaded compression
		DisableMultistream: false,
	}, initramfs)
	if err != nil {
		return nil, errors.Errorf("creating compressor pipeline: %w", err)
	}

	// counter := iox.NewReadCounter(arc)
	// counter.SetDebug(true)

	return arc, nil
}

func (hpv *Hypervisor) EncodeLinuxKernel(ctx context.Context, kernel io.Reader) (io.ReadCloser, error) {
	// ensure kernel is valid
	validationReader, err := magic.ARM64LinuxKernelValidationReader(kernel)
	if err != nil {
		return nil, errors.Errorf("checking linux kernel: %w", err)
	}
	return validationReader, nil
}

func (hpv *Hypervisor) EncodeLinuxRootfs(ctx context.Context, rootfs io.Reader) (io.ReadCloser, error) {
	return io.NopCloser(rootfs), nil
}

func (hpv *Hypervisor) InitramfsCompression() archives.Compression {
	return &archives.Gz{}
}
