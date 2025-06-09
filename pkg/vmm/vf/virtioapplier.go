package vf

import (
	"context"
	"log/slog"
	"os"

	"github.com/pkg/term/termios"

	"github.com/Code-Hex/vz/v3"
	"github.com/crc-org/vfkit/pkg/util"
	"gitlab.com/tozd/go/errors"

	"github.com/walteh/ec1/pkg/virtio"
	"github.com/walteh/ec1/pkg/vmm"
)

var _ virtio.DeviceApplier = &vzVirtioDeviceApplier{}

type vzVirtioDeviceApplier struct {
	storageDevicesToSet          []vz.StorageDeviceConfiguration
	directorySharingDevicesToSet []vz.DirectorySharingDeviceConfiguration
	keyboardToSet                []vz.KeyboardConfiguration
	pointingDevicesToSet         []vz.PointingDeviceConfiguration
	graphicsDevicesToSet         []vz.GraphicsDeviceConfiguration
	networkDevicesToSet          []*vz.VirtioNetworkDeviceConfiguration
	entropyDevicesToSet          []*vz.VirtioEntropyDeviceConfiguration
	serialPortsToSet             []*vz.VirtioConsoleDeviceSerialPortConfiguration
	// socketDevicesToSet           []vz.SocketDeviceConfiguration
	consolePortsToSet []*vz.VirtioConsolePortConfiguration
	// memoryBalloonDevicesConfiguration    []vz.MemoryBalloonDeviceConfiguration

	addSocketDevice        bool
	addMemoryBalloonDevice bool

	*vz.VirtualMachineConfiguration
	bootLoader vmm.Bootloader
}

func NewVzVirtioDeviceApplier(ctx context.Context, cfg *vz.VirtualMachineConfiguration, bootLoader vmm.Bootloader) (*vzVirtioDeviceApplier, error) {

	wrapper := &vzVirtioDeviceApplier{
		storageDevicesToSet:          make([]vz.StorageDeviceConfiguration, 0),
		directorySharingDevicesToSet: make([]vz.DirectorySharingDeviceConfiguration, 0),
		pointingDevicesToSet:         make([]vz.PointingDeviceConfiguration, 0),
		keyboardToSet:                make([]vz.KeyboardConfiguration, 0),
		graphicsDevicesToSet:         make([]vz.GraphicsDeviceConfiguration, 0),
		networkDevicesToSet:          make([]*vz.VirtioNetworkDeviceConfiguration, 0),
		entropyDevicesToSet:          make([]*vz.VirtioEntropyDeviceConfiguration, 0),
		serialPortsToSet:             make([]*vz.VirtioConsoleDeviceSerialPortConfiguration, 0),
		consolePortsToSet:            make([]*vz.VirtioConsolePortConfiguration, 0),
		addSocketDevice:              false,
		addMemoryBalloonDevice:       false,
		bootLoader:                   bootLoader,
		VirtualMachineConfiguration:  cfg,
	}

	return wrapper, nil
}

func (v *vzVirtioDeviceApplier) Finalize(ctx context.Context) error {

	v.SetStorageDevicesVirtualMachineConfiguration(v.storageDevicesToSet)
	v.SetDirectorySharingDevicesVirtualMachineConfiguration(v.directorySharingDevicesToSet)
	v.SetPointingDevicesVirtualMachineConfiguration(v.pointingDevicesToSet)
	v.SetKeyboardsVirtualMachineConfiguration(v.keyboardToSet)
	v.SetGraphicsDevicesVirtualMachineConfiguration(v.graphicsDevicesToSet)
	v.SetNetworkDevicesVirtualMachineConfiguration(v.networkDevicesToSet)
	v.SetEntropyDevicesVirtualMachineConfiguration(v.entropyDevicesToSet)
	v.SetSerialPortsVirtualMachineConfiguration(v.serialPortsToSet)

	if v.addMemoryBalloonDevice {
		bal, err := vz.NewVirtioTraditionalMemoryBalloonDeviceConfiguration()
		if err != nil {
			return errors.Errorf("creating memory balloon device configuration: %w", err)
		}

		v.SetMemoryBalloonDevicesVirtualMachineConfiguration([]vz.MemoryBalloonDeviceConfiguration{bal})
	}

	if v.addSocketDevice {
		vzdev, err := vz.NewVirtioSocketDeviceConfiguration()
		if err != nil {
			return errors.Errorf("creating vsock device configuration: %w", err)
		}
		v.SetSocketDevicesVirtualMachineConfiguration([]vz.SocketDeviceConfiguration{vzdev})
	}

	platformConfig, err := toVzPlatformConfiguration(v.bootLoader)
	if err != nil {
		return errors.Errorf("converting platform configuration to vz platform configuration: %w", err)
	}

	v.SetPlatformVirtualMachineConfiguration(platformConfig)

	valid, err := v.Validate()
	if err != nil {
		return errors.Errorf("validating virtual machine configuration: %w", err)
	}
	if !valid {
		return errors.New("invalid virtual machine configuration")
	} else {
		slog.InfoContext(ctx, "virtual machine configuration is valid")
	}

	return nil
}

// ApplyVirtioBalloon implements virtio.DeviceApplier.
func (v *vzVirtioDeviceApplier) ApplyVirtioBalloon(ctx context.Context, dev *virtio.VirtioBalloon) error {
	v.addMemoryBalloonDevice = true
	return nil
}

// ApplyVirtioBlk implements virtio.DeviceApplier.
func (v *vzVirtioDeviceApplier) ApplyVirtioBlk(ctx context.Context, dev *virtio.VirtioBlk) error {
	return v.applyVirtioBlk(dev)
}

// ApplyVirtioFs implements virtio.DeviceApplier.
func (v *vzVirtioDeviceApplier) ApplyVirtioFs(ctx context.Context, dev *virtio.VirtioFs) error {
	return v.applyVirtioFs(dev)
}

// ApplyVirtioGPU implements virtio.DeviceApplier.
func (v *vzVirtioDeviceApplier) ApplyVirtioGPU(ctx context.Context, dev *virtio.VirtioGPU) error {
	return v.applyVirtioGPU(dev)
}

// ApplyVirtioInput implements virtio.DeviceApplier.
func (v *vzVirtioDeviceApplier) ApplyVirtioInput(ctx context.Context, dev *virtio.VirtioInput) error {
	return v.applyVirtioInput(dev)
}

// ApplyVirtioNVMExpressController implements virtio.DeviceApplier.
func (v *vzVirtioDeviceApplier) ApplyVirtioNVMExpressController(ctx context.Context, dev *virtio.NVMExpressController) error {
	return v.applyNVMExpressController(dev)
}

// ApplyVirtioNetworkBlockDevice implements virtio.DeviceApplier.
func (v *vzVirtioDeviceApplier) ApplyVirtioNetworkBlockDevice(ctx context.Context, dev *virtio.NetworkBlockDevice) error {
	return v.applyNetworkBlockDevice(dev)
}

// ApplyVirtioRng implements virtio.DeviceApplier.
func (v *vzVirtioDeviceApplier) ApplyVirtioRng(ctx context.Context, dev *virtio.VirtioRng) error {
	return v.applyVirtioRng(dev)
}

// ApplyVirtioRosettaShare implements virtio.DeviceApplier.
func (v *vzVirtioDeviceApplier) ApplyVirtioRosettaShare(ctx context.Context, dev *virtio.RosettaShare) error {
	return v.applyRosettaShare(dev)
}

// ApplyVirtioSerialFifoFile implements virtio.DeviceApplier.
func (v *vzVirtioDeviceApplier) ApplyVirtioSerialFifoFile(ctx context.Context, dev *virtio.VirtioSerialFifoFile) error {
	serialPortAttachment, err := vz.NewFileSerialPortAttachment(dev.Path, false)
	if err != nil {
		return errors.Errorf("creating file serial port attachment: %w", err)
	}

	serialPort, err := vz.NewVirtioConsoleDeviceSerialPortConfiguration(serialPortAttachment)
	if err != nil {
		return errors.Errorf("creating virtio console device serial port configuration: %w", err)
	}

	v.serialPortsToSet = append(v.serialPortsToSet, serialPort)
	return nil
}

// ApplyVirtioSerialFifo implements virtio.DeviceApplier.
func (v *vzVirtioDeviceApplier) ApplyVirtioSerialFifo(ctx context.Context, dev *virtio.VirtioSerialFifo) error {
	fifoRead := os.NewFile(uintptr(dev.FD), "fifo-read")
	fifoWrite := os.NewFile(uintptr(dev.FD), "fifo-write")

	serialPortAttachment, err := vz.NewFileHandleSerialPortAttachment(fifoRead, fifoWrite)
	if err != nil {
		return errors.Errorf("creating file handle serial port attachment: %w", err)
	}

	serialPort, err := vz.NewVirtioConsoleDeviceSerialPortConfiguration(serialPortAttachment)
	if err != nil {
		return errors.Errorf("creating virtio console device serial port configuration: %w", err)
	}

	v.serialPortsToSet = append(v.serialPortsToSet, serialPort)
	return nil
}

// ApplyVirtioSerialStdio implements virtio.DeviceApplier.
func (v *vzVirtioDeviceApplier) ApplyVirtioSerialStdio(ctx context.Context, dev *virtio.VirtioSerialStdio) error {
	stdin := dev.Stdin
	stdout := dev.Stdout
	if err := setRawMode(stdin); err != nil {
		return errors.Errorf("setting raw mode for stdin: %w", err)
	}
	serialPortAttachment, err := vz.NewFileHandleSerialPortAttachment(stdin, stdout)
	if err != nil {
		return errors.Errorf("creating file handle serial port attachment: %w", err)
	}
	serialPort, err := vz.NewVirtioConsoleDeviceSerialPortConfiguration(serialPortAttachment)
	if err != nil {
		return errors.Errorf("creating virtio console device serial port configuration: %w", err)
	}
	v.serialPortsToSet = append(v.serialPortsToSet, serialPort)
	return nil
}

// ApplyVirtioSerialPty implements virtio.DeviceApplier.
func (v *vzVirtioDeviceApplier) ApplyVirtioSerialPty(ctx context.Context, dev *virtio.VirtioSerialPty) error {
	master, slave, err := termios.Pty()
	if err != nil {
		return errors.Errorf("creating pty: %w", err)
	}

	// the master fd and slave fd must stay open for vfkit's lifetime
	util.RegisterExitHandler(func() {
		_ = master.Close()
		_ = slave.Close()
	})

	dev.InternalManagedName = slave.Name()

	if err := setRawMode(master); err != nil {
		return errors.Errorf("setting raw mode for master: %w", err)
	}

	serialPortAttachment, err := vz.NewFileHandleSerialPortAttachment(master, master)
	if err != nil {
		return errors.Errorf("creating file handle serial port attachment: %w", err)
	}

	serialPort, err := vz.NewVirtioConsolePortConfiguration(
		vz.WithVirtioConsolePortConfigurationAttachment(serialPortAttachment),
		vz.WithVirtioConsolePortConfigurationIsConsole(dev.IsSystemConsole),
	)
	if err != nil {
		return errors.Errorf("creating virtio console device serial port configuration: %w", err)
	}

	v.consolePortsToSet = append(v.consolePortsToSet, serialPort)
	return nil
}

// ApplyVirtioSerialLogFile implements virtio.DeviceApplier.
func (v *vzVirtioDeviceApplier) ApplyVirtioSerialLogFile(ctx context.Context, dev *virtio.VirtioSerialLogFile) error {
	serialPortAttachment, err := vz.NewFileSerialPortAttachment(dev.Path, dev.Append)
	if err != nil {
		return errors.Errorf("creating file serial port attachment: %w", err)
	}

	serialPort, err := vz.NewVirtioConsoleDeviceSerialPortConfiguration(serialPortAttachment)
	if err != nil {
		return errors.Errorf("creating virtio console device serial port configuration: %w", err)
	}

	v.serialPortsToSet = append(v.serialPortsToSet, serialPort)

	return nil
}

// ApplyVirtioStorage implements virtio.DeviceApplier.
func (v *vzVirtioDeviceApplier) ApplyVirtioStorage(ctx context.Context, dev *virtio.StorageConfig) error {
	return nil // not sure if this is directly used
}

// ApplyVirtioUsbMassStorage implements virtio.DeviceApplier.
func (v *vzVirtioDeviceApplier) ApplyVirtioUsbMassStorage(ctx context.Context, dev *virtio.USBMassStorage) error {
	return v.applyUSBMassStorage(dev)
}

// ApplyVirtioVirtioBlk implements virtio.DeviceApplier.
func (v *vzVirtioDeviceApplier) ApplyVirtioVirtioBlk(ctx context.Context, dev *virtio.VirtioBlk) error {
	return v.applyVirtioBlk(dev)
}

// ApplyVirtioVirtioFs implements virtio.DeviceApplier.
func (v *vzVirtioDeviceApplier) ApplyVirtioVirtioFs(ctx context.Context, dev *virtio.VirtioFs) error {
	return v.applyVirtioFs(dev)
}

// ApplyVirtioVsock implements virtio.DeviceApplier.
func (v *vzVirtioDeviceApplier) ApplyVirtioVsock(ctx context.Context, dev *virtio.VirtioVsock) error {
	v.addSocketDevice = true
	return nil // not sure if this is directly used
}

// type VirtualMachineConfiguration struct {
// 	id        string
// 	bl        vmm.Bootloader
// 	newVMOpts *vmm.NewVMOptions // go-friendly virtual machine configuration definition
// 	wrapper   *vzVirtioDeviceApplier
// 	internal  *vz.VirtualMachineConfiguration
// 	platform  string
// }

// func NewVirtualMachineConfiguration(ctx context.Context, id string, opts *vmm.NewVMOptions, bootLoader vmm.Bootloader) (*VirtualMachineConfiguration, error) {

// 	wrapper := &vzVirtioConverter{
// 		storageDevicesToSet:          make([]vz.StorageDeviceConfiguration, 0),
// 		directorySharingDevicesToSet: make([]vz.DirectorySharingDeviceConfiguration, 0),
// 		keyboardToSet:                make([]vz.KeyboardConfiguration, 0),
// 		pointingDevicesToSet:         make([]vz.PointingDeviceConfiguration, 0),
// 		graphicsDevicesToSet:         make([]vz.GraphicsDeviceConfiguration, 0),
// 		networkDevicesToSet:          make([]*vz.VirtioNetworkDeviceConfiguration, 0),
// 		entropyDevicesToSet:          make([]*vz.VirtioEntropyDeviceConfiguration, 0),
// 		serialPortsToSet:             make([]*vz.VirtioConsoleDeviceSerialPortConfiguration, 0),
// 		consolePortsToSet:            make([]*vz.VirtioConsolePortConfiguration, 0),
// 	}

// 	// vzVMConfig.SetSocketDevicesVirtualMachineConfiguration(wrapper.socketDevicesConfiguration)
// 	// vzVMConfig.SetMemoryBalloonDevicesVirtualMachineConfiguration(wrapper.memoryBalloonDevicesConfiguration)

// 	if len(wrapper.consolePortsConfiguration) > 0 {
// 		slog.DebugContext(ctx, "Adding console devices to virtual machine configuration", "count", len(wrapper.consolePortsConfiguration))

// 		consoleDeviceConfiguration, err := vz.NewVirtioConsoleDeviceConfiguration()
// 		if err != nil {
// 			return nil, errors.Errorf("creating console device configuration: %w", err)
// 		}
// 		for i, portCfg := range wrapper.consolePortsConfiguration {
// 			consoleDeviceConfiguration.SetVirtioConsolePortConfiguration(i, portCfg)
// 		}
// 		vzVMConfig.SetConsoleDevicesVirtualMachineConfiguration([]vz.ConsoleDeviceConfiguration{consoleDeviceConfiguration})
// 	}

// 	// always add a memory balloon device
// 	bal, err := vz.NewVirtioTraditionalMemoryBalloonDeviceConfiguration()
// 	if err != nil {
// 		return nil, errors.Errorf("creating memory balloon device configuration: %w", err)
// 	}
// 	vzVMConfig.SetMemoryBalloonDevicesVirtualMachineConfiguration([]vz.MemoryBalloonDeviceConfiguration{bal})

// 	slog.DebugContext(ctx, "Validating virtual machine configuration")

// 	valid, err := vzVMConfig.Validate()
// 	if err != nil {
// 		return nil, errors.Errorf("validating virtual machine configuration: %w", err)
// 	}
// 	if !valid {
// 		return nil, errors.New("invalid virtual machine configuration")
// 	}

// 	return &VirtualMachineConfiguration{
// 		id:        id,
// 		bl:        bootLoader,
// 		newVMOpts: opts,
// 		wrapper:   wrapper,
// 		internal:  vzVMConfig,
// 		platform:  platformType,
// 	}, nil
// }
