package virtio

import "context"

type DeviceApplier interface {
	ApplyVirtioInput(ctx context.Context, vmConfig *VirtioInput) error
	ApplyVirtioGPU(ctx context.Context, vmConfig *VirtioGPU) error
	ApplyVirtioGPUResolution(ctx context.Context, vmConfig *VirtioGPUResolution) error
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
	ApplyVirtioDiskStorage(ctx context.Context, vmConfig *DiskStorageConfig) error
	ApplyVirtioNetworkBlockStorage(ctx context.Context, vmConfig *NetworkBlockStorageConfig) error
	ApplyVirtioStorage(ctx context.Context, vmConfig *StorageConfig) error
	ApplyVirtioVirtioBlk(ctx context.Context, vmConfig *VirtioBlk) error
	ApplyVirtioVirtioFs(ctx context.Context, vmConfig *VirtioFs) error
}
