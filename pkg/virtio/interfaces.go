package virtio

import (
	"context"
	"fmt"

	"gitlab.com/tozd/go/errors"
)

func ApplyDevices(ctx context.Context, applier DeviceApplier, devices []VirtioDevice) error {
	for _, dev := range devices {
		switch dev := dev.(type) {
		case *VirtioNet:
			return applier.ApplyVirtioNet(ctx, dev)
		case *VirtioInput:
			return applier.ApplyVirtioInput(ctx, dev)
		case *VirtioGPU:
			return applier.ApplyVirtioGPU(ctx, dev)
		case *VirtioVsock:
			return applier.ApplyVirtioVsock(ctx, dev)
		case *VirtioBlk:
			return applier.ApplyVirtioBlk(ctx, dev)
		case *VirtioFs:
			return applier.ApplyVirtioFs(ctx, dev)
		case *VirtioRng:
			return applier.ApplyVirtioRng(ctx, dev)
		case *VirtioSerial:
			return applier.ApplyVirtioSerial(ctx, dev)
		case *VirtioBalloon:
			return applier.ApplyVirtioBalloon(ctx, dev)
		case *NetworkBlockDevice:
			return applier.ApplyVirtioNetworkBlockDevice(ctx, dev)
		case *NVMExpressController:
			return applier.ApplyVirtioNVMExpressController(ctx, dev)
		case *RosettaShare:
			return applier.ApplyVirtioRosettaShare(ctx, dev)
		case *USBMassStorage:
			return applier.ApplyVirtioUsbMassStorage(ctx, dev)
		default:
			return fmt.Errorf("unsupported device type: %T", dev)
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
	ApplyVirtioSerial(ctx context.Context, vmConfig *VirtioSerial) error
	ApplyVirtioBalloon(ctx context.Context, vmConfig *VirtioBalloon) error
	ApplyVirtioNetworkBlockDevice(ctx context.Context, vmConfig *NetworkBlockDevice) error
	ApplyVirtioNVMExpressController(ctx context.Context, vmConfig *NVMExpressController) error
	ApplyVirtioRosettaShare(ctx context.Context, vmConfig *RosettaShare) error
	ApplyVirtioUsbMassStorage(ctx context.Context, vmConfig *USBMassStorage) error
	// ApplyVirtioDiskStorage(ctx context.Context, vmConfig *DiskStorageConfig) error
	// ApplyVirtioNetworkBlockStorage(ctx context.Context, vmConfig *NetworkBlockStorageConfig) error
	// ApplyVirtioStorage(ctx context.Context, vmConfig *StorageConfig) error
}
