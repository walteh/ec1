package vf

import (
	"context"

	"github.com/Code-Hex/vz/v3"

	"github.com/walteh/ec1/pkg/virtio"
)

var _ virtio.DeviceApplier = &vzVirtioConverter{}

type vzVirtioConverter struct {
	useMacOSGPUGraphicsDevice            bool
	storageDevicesConfiguration          []vz.StorageDeviceConfiguration
	directorySharingDevicesConfiguration []vz.DirectorySharingDeviceConfiguration
	keyboardConfiguration                []vz.KeyboardConfiguration
	pointingDevicesConfiguration         []vz.PointingDeviceConfiguration
	graphicsDevicesConfiguration         []vz.GraphicsDeviceConfiguration
	networkDevicesConfiguration          []*vz.VirtioNetworkDeviceConfiguration
	entropyDevicesConfiguration          []*vz.VirtioEntropyDeviceConfiguration
	serialPortsConfiguration             []*vz.VirtioConsoleDeviceSerialPortConfiguration
	socketDevicesConfiguration           []vz.SocketDeviceConfiguration
	consolePortsConfiguration            []*vz.VirtioConsolePortConfiguration
	memoryBalloonDevicesConfiguration    []vz.MemoryBalloonDeviceConfiguration
}

// ApplyVirtioBalloon implements virtio.DeviceApplier.
func (v *vzVirtioConverter) ApplyVirtioBalloon(ctx context.Context, vmConfig *virtio.VirtioBalloon) error {
	panic("unimplemented")
}

// ApplyVirtioBlk implements virtio.DeviceApplier.
func (v *vzVirtioConverter) ApplyVirtioBlk(ctx context.Context, vmConfig *virtio.VirtioBlk) error {
	panic("unimplemented")
}

// ApplyVirtioDiskStorage implements virtio.DeviceApplier.
func (v *vzVirtioConverter) ApplyVirtioDiskStorage(ctx context.Context, vmConfig *virtio.DiskStorageConfig) error {
	panic("unimplemented")
}

// ApplyVirtioFs implements virtio.DeviceApplier.
func (v *vzVirtioConverter) ApplyVirtioFs(ctx context.Context, vmConfig *virtio.VirtioFs) error {
	panic("unimplemented")
}

// ApplyVirtioGPU implements virtio.DeviceApplier.
func (v *vzVirtioConverter) ApplyVirtioGPU(ctx context.Context, vmConfig *virtio.VirtioGPU) error {
	panic("unimplemented")
}

// ApplyVirtioGPUResolution implements virtio.DeviceApplier.
func (v *vzVirtioConverter) ApplyVirtioGPUResolution(ctx context.Context, vmConfig *virtio.VirtioGPUResolution) error {
	panic("unimplemented")
}

// ApplyVirtioInput implements virtio.DeviceApplier.
func (v *vzVirtioConverter) ApplyVirtioInput(ctx context.Context, vmConfig *virtio.VirtioInput) error {
	panic("unimplemented")
}

// ApplyVirtioNVMExpressController implements virtio.DeviceApplier.
func (v *vzVirtioConverter) ApplyVirtioNVMExpressController(ctx context.Context, vmConfig *virtio.NVMExpressController) error {
	panic("unimplemented")
}

// ApplyVirtioNetworkBlockDevice implements virtio.DeviceApplier.
func (v *vzVirtioConverter) ApplyVirtioNetworkBlockDevice(ctx context.Context, vmConfig *virtio.NetworkBlockDevice) error {
	panic("unimplemented")
}

// ApplyVirtioNetworkBlockStorage implements virtio.DeviceApplier.
func (v *vzVirtioConverter) ApplyVirtioNetworkBlockStorage(ctx context.Context, vmConfig *virtio.NetworkBlockStorageConfig) error {
	panic("unimplemented")
}

// ApplyVirtioRng implements virtio.DeviceApplier.
func (v *vzVirtioConverter) ApplyVirtioRng(ctx context.Context, vmConfig *virtio.VirtioRng) error {
	panic("unimplemented")
}

// ApplyVirtioRosettaShare implements virtio.DeviceApplier.
func (v *vzVirtioConverter) ApplyVirtioRosettaShare(ctx context.Context, vmConfig *virtio.RosettaShare) error {
	panic("unimplemented")
}

// ApplyVirtioSerial implements virtio.DeviceApplier.
func (v *vzVirtioConverter) ApplyVirtioSerial(ctx context.Context, vmConfig *virtio.VirtioSerial) error {
	panic("unimplemented")
}

// ApplyVirtioStorage implements virtio.DeviceApplier.
func (v *vzVirtioConverter) ApplyVirtioStorage(ctx context.Context, vmConfig *virtio.StorageConfig) error {
	panic("unimplemented")
}

// ApplyVirtioUsbMassStorage implements virtio.DeviceApplier.
func (v *vzVirtioConverter) ApplyVirtioUsbMassStorage(ctx context.Context, vmConfig *virtio.USBMassStorage) error {
	panic("unimplemented")
}

// ApplyVirtioVirtioBlk implements virtio.DeviceApplier.
func (v *vzVirtioConverter) ApplyVirtioVirtioBlk(ctx context.Context, vmConfig *virtio.VirtioBlk) error {
	panic("unimplemented")
}

// ApplyVirtioVirtioFs implements virtio.DeviceApplier.
func (v *vzVirtioConverter) ApplyVirtioVirtioFs(ctx context.Context, vmConfig *virtio.VirtioFs) error {
	panic("unimplemented")
}

// ApplyVirtioVsock implements virtio.DeviceApplier.
func (v *vzVirtioConverter) ApplyVirtioVsock(ctx context.Context, vmConfig *virtio.VirtioVsock) error {
	panic("unimplemented")
}
