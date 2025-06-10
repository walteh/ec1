package virtio

import (
	"context"
	"fmt"
	"log/slog"

	"gitlab.com/tozd/go/errors"
)

func ApplyDevices(ctx context.Context, applier DeviceApplier, devices []VirtioDevice) error {

	for _, dev := range devices {
		slog.InfoContext(ctx, "applying virtio device", "device", fmt.Sprintf("%T", dev))
		switch dev := dev.(type) {
		case *VirtioNet:
			if err := applier.ApplyVirtioNet(ctx, dev); err != nil {
				return errors.Errorf("applying virtio net: %w", err)
			}
		case *VirtioInput:
			if err := applier.ApplyVirtioInput(ctx, dev); err != nil {
				return errors.Errorf("applying virtio input: %w", err)
			}
		case *VirtioGPU:
			if err := applier.ApplyVirtioGPU(ctx, dev); err != nil {
				return errors.Errorf("applying virtio gpu: %w", err)
			}
		case *VirtioVsock:
			if err := applier.ApplyVirtioVsock(ctx, dev); err != nil {
				return errors.Errorf("applying virtio vsock: %w", err)
			}
		case *VirtioBlk:
			if err := applier.ApplyVirtioBlk(ctx, dev); err != nil {
				return errors.Errorf("applying virtio blk: %w", err)
			}
		case *VirtioFs:
			if err := applier.ApplyVirtioFs(ctx, dev); err != nil {
				return errors.Errorf("applying virtio fs: %w", err)
			}
		case *VirtioRng:
			if err := applier.ApplyVirtioRng(ctx, dev); err != nil {
				return errors.Errorf("applying virtio rng: %w", err)
			}
		case *VirtioSerialFifo:
			if err := applier.ApplyVirtioSerialFifo(ctx, dev); err != nil {
				return errors.Errorf("applying virtio serial fifo: %w", err)
			}
		case *VirtioSerialStdio:
			if err := applier.ApplyVirtioSerialStdio(ctx, dev); err != nil {
				return errors.Errorf("applying virtio serial stdio: %w", err)
			}
		case *VirtioSerialStdioPipes:
			if err := applier.ApplyVirtioSerialStdioPipes(ctx, dev); err != nil {
				return errors.Errorf("applying virtio serial stdio pipes: %w", err)
			}
		case *VirtioSerialPty:
			if err := applier.ApplyVirtioSerialPty(ctx, dev); err != nil {
				return errors.Errorf("applying virtio serial pty: %w", err)
			}
		case *VirtioSerialLogFile:
			if err := applier.ApplyVirtioSerialLogFile(ctx, dev); err != nil {
				return errors.Errorf("applying virtio serial log file: %w", err)
			}
		case *VirtioBalloon:
			if err := applier.ApplyVirtioBalloon(ctx, dev); err != nil {
				return errors.Errorf("applying virtio balloon: %w", err)
			}
		case *NetworkBlockDevice:
			if err := applier.ApplyVirtioNetworkBlockDevice(ctx, dev); err != nil {
				return errors.Errorf("applying virtio network block device: %w", err)
			}
		case *NVMExpressController:
			if err := applier.ApplyVirtioNVMExpressController(ctx, dev); err != nil {
				return errors.Errorf("applying virtio nvme express controller: %w", err)
			}
		case *RosettaShare:
			if err := applier.ApplyVirtioRosettaShare(ctx, dev); err != nil {
				return errors.Errorf("applying virtio rosetta share: %w", err)
			}
		case *USBMassStorage:
			if err := applier.ApplyVirtioUsbMassStorage(ctx, dev); err != nil {
				return errors.Errorf("applying virtio usb mass storage: %w", err)
			}
		case *VirtioSerialFifoFile:
			if err := applier.ApplyVirtioSerialFifoFile(ctx, dev); err != nil {
				return errors.Errorf("applying virtio serial fifo file: %w", err)
			}
		case *VirtioSerialFDPipes:
			if err := applier.ApplyVirtioSerialFDPipes(ctx, dev); err != nil {
				return errors.Errorf("applying virtio serial fd pipes: %w", err)
			}
		default:
			return errors.Errorf("unsupported device type: %T", dev)
		}
	}

	if err := applier.Finalize(ctx); err != nil {
		return errors.Errorf("finalizing virtual machine configuration: %w", err)
	}

	return nil
}

type DeviceApplier interface {
	Finalize(ctx context.Context) error
	ApplyVirtioNet(ctx context.Context, vmConfig *VirtioNet) error
	ApplyVirtioInput(ctx context.Context, vmConfig *VirtioInput) error
	ApplyVirtioGPU(ctx context.Context, vmConfig *VirtioGPU) error
	ApplyVirtioVsock(ctx context.Context, vmConfig *VirtioVsock) error
	ApplyVirtioBlk(ctx context.Context, vmConfig *VirtioBlk) error
	ApplyVirtioFs(ctx context.Context, vmConfig *VirtioFs) error
	ApplyVirtioRng(ctx context.Context, vmConfig *VirtioRng) error
	ApplyVirtioBalloon(ctx context.Context, vmConfig *VirtioBalloon) error
	ApplyVirtioNetworkBlockDevice(ctx context.Context, vmConfig *NetworkBlockDevice) error
	ApplyVirtioNVMExpressController(ctx context.Context, vmConfig *NVMExpressController) error
	ApplyVirtioRosettaShare(ctx context.Context, vmConfig *RosettaShare) error
	ApplyVirtioUsbMassStorage(ctx context.Context, vmConfig *USBMassStorage) error
	ApplyVirtioSerialFifo(ctx context.Context, vmConfig *VirtioSerialFifo) error
	ApplyVirtioSerialStdio(ctx context.Context, vmConfig *VirtioSerialStdio) error
	ApplyVirtioSerialPty(ctx context.Context, vmConfig *VirtioSerialPty) error
	ApplyVirtioSerialLogFile(ctx context.Context, vmConfig *VirtioSerialLogFile) error
	ApplyVirtioSerialFifoFile(ctx context.Context, vmConfig *VirtioSerialFifoFile) error
	ApplyVirtioSerialStdioPipes(ctx context.Context, vmConfig *VirtioSerialStdioPipes) error
	ApplyVirtioSerialFDPipes(ctx context.Context, vmConfig *VirtioSerialFDPipes) error
	// ApplyVirtioDiskStorage(ctx context.Context, vmConfig *DiskStorageConfig) error
	// ApplyVirtioNetworkBlockStorage(ctx context.Context, vmConfig *NetworkBlockStorageConfig) error
	// ApplyVirtioStorage(ctx context.Context, vmConfig *StorageConfig) error
}
