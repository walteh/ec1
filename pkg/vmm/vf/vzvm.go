// Package vf converts a config.VirtualMachine configuration to native
// virtualization framework datatypes. It also provides APIs to start/stop/...
// the virtualization framework virtual machine.
//
// The interaction with the virtualization framework is done using the
// Code-Hex/vz Objective-C bindings. This requires cgo, and this package cannot
// be easily cross-compiled, it must be built on macOS.
package vf

import (
	"context"
	"log/slog"

	"github.com/Code-Hex/vz/v3"
	"gitlab.com/tozd/go/errors"

	"github.com/walteh/ec1/pkg/vmm"
)

// var PlatformType string

func (vm *VirtualMachine) Start(ctx context.Context) error {
	if vm.vzvm == nil {
		return errors.Errorf("virtual machine not initialized")
	}
	slog.DebugContext(ctx, "Starting virtual machine")
	return vm.vzvm.Start()
}
func (hpv *VirtualMachine) VZ() *vz.VirtualMachine {
	return hpv.vzvm
}

type VirtualMachineConfiguration struct {
	id        string
	bl        vmm.Bootloader
	newVMOpts *vmm.NewVMOptions // go-friendly virtual machine configuration definition
	wrapper   *vzVirtioConverter
	internal  *vz.VirtualMachineConfiguration
	platform  string
}

func NewVirtualMachineConfiguration(ctx context.Context, id string, opts *vmm.NewVMOptions, bootLoader vmm.Bootloader) (*VirtualMachineConfiguration, error) {

	wrapper := &vzVirtioConverter{
		storageDevicesConfiguration:          make([]vz.StorageDeviceConfiguration, 0),
		directorySharingDevicesConfiguration: make([]vz.DirectorySharingDeviceConfiguration, 0),
		keyboardConfiguration:                make([]vz.KeyboardConfiguration, 0),
		pointingDevicesConfiguration:         make([]vz.PointingDeviceConfiguration, 0),
		graphicsDevicesConfiguration:         make([]vz.GraphicsDeviceConfiguration, 0),
		networkDevicesConfiguration:          make([]*vz.VirtioNetworkDeviceConfiguration, 0),
		entropyDevicesConfiguration:          make([]*vz.VirtioEntropyDeviceConfiguration, 0),
		memoryBalloonDevicesConfiguration:    make([]vz.MemoryBalloonDeviceConfiguration, 0),
		serialPortsConfiguration:             make([]*vz.VirtioConsoleDeviceSerialPortConfiguration, 0),
		socketDevicesConfiguration:           make([]vz.SocketDeviceConfiguration, 0),
		consolePortsConfiguration:            make([]*vz.VirtioConsolePortConfiguration, 0),
		useMacOSGPUGraphicsDevice:            false,
	}

	slog.DebugContext(ctx, "Creating virtual machine configuration")
	vzBootloader, err := toVzBootloader(bootLoader)
	if err != nil {
		return nil, errors.Errorf("converting bootloader to vz bootloader: %w", err)
	}

	vzVMConfig, err := vz.NewVirtualMachineConfiguration(vzBootloader, uint(opts.Vcpus), uint64(opts.Memory.ToBytes()))
	if err != nil {
		return nil, errors.Errorf("creating vz virtual machine configuration: %w", err)
	}

	var platformType string

	if macosBootloader, ok := bootLoader.(*vmm.MacOSBootloader); ok {
		platformConfig, err := NewMacPlatformConfiguration(macosBootloader.MachineIdentifierPath, macosBootloader.HardwareModelPath, macosBootloader.AuxImagePath)
		if err != nil {
			return nil, errors.Errorf("creating macos platform configuration: %w", err)
		}

		slog.DebugContext(ctx, "Setting platform type", "platform", "macos")
		wrapper.useMacOSGPUGraphicsDevice = true
		vzVMConfig.SetPlatformVirtualMachineConfiguration(platformConfig)
	}

	// if cfg.config.Timesync != nil && cfg.config.Timesync.VsockPort != 0 {
	// 	// automatically add the vsock device we'll need for communication over VsockPort
	// 	vsockDev := VirtioVsock{
	// 		Port:   cfg.config.Timesync.VsockPort,
	// 		Listen: false,
	// 	}
	// 	if err := vsockDev.AddToVirtualMachineConfig(cfg); err != nil {
	// 		return nil, err
	// 	}
	// }

	slog.DebugContext(ctx, "Adding devices to virtual machine configuration", "count", len(opts.Devices))

	for _, dev := range opts.Devices {
		if err := AddToVirtualMachineConfig(wrapper, dev); err != nil {
			return nil, errors.Errorf("adding device to virtual machine configuration: %T: %w", dev, err)
		}
	}

	vzVMConfig.SetStorageDevicesVirtualMachineConfiguration(wrapper.storageDevicesConfiguration)
	vzVMConfig.SetDirectorySharingDevicesVirtualMachineConfiguration(wrapper.directorySharingDevicesConfiguration)
	vzVMConfig.SetPointingDevicesVirtualMachineConfiguration(wrapper.pointingDevicesConfiguration)
	vzVMConfig.SetKeyboardsVirtualMachineConfiguration(wrapper.keyboardConfiguration)
	vzVMConfig.SetGraphicsDevicesVirtualMachineConfiguration(wrapper.graphicsDevicesConfiguration)
	vzVMConfig.SetNetworkDevicesVirtualMachineConfiguration(wrapper.networkDevicesConfiguration)
	vzVMConfig.SetEntropyDevicesVirtualMachineConfiguration(wrapper.entropyDevicesConfiguration)
	vzVMConfig.SetSerialPortsVirtualMachineConfiguration(wrapper.serialPortsConfiguration)
	// vzVMConfig.SetSocketDevicesVirtualMachineConfiguration(wrapper.socketDevicesConfiguration)
	// vzVMConfig.SetMemoryBalloonDevicesVirtualMachineConfiguration(wrapper.memoryBalloonDevicesConfiguration)

	if len(wrapper.consolePortsConfiguration) > 0 {
		slog.DebugContext(ctx, "Adding console devices to virtual machine configuration", "count", len(wrapper.consolePortsConfiguration))

		consoleDeviceConfiguration, err := vz.NewVirtioConsoleDeviceConfiguration()
		if err != nil {
			return nil, errors.Errorf("creating console device configuration: %w", err)
		}
		for i, portCfg := range wrapper.consolePortsConfiguration {
			consoleDeviceConfiguration.SetVirtioConsolePortConfiguration(i, portCfg)
		}
		vzVMConfig.SetConsoleDevicesVirtualMachineConfiguration([]vz.ConsoleDeviceConfiguration{consoleDeviceConfiguration})
	}

	// always add a vsock device
	vzdev, err := vz.NewVirtioSocketDeviceConfiguration()
	if err != nil {
		return nil, errors.Errorf("creating vsock device configuration: %w", err)
	}

	// len(cfg.socketDevicesConfiguration should be 0 or 1
	// https://developer.apple.com/documentation/virtualization/vzvirtiosocketdeviceconfiguration?language=objc
	vzVMConfig.SetSocketDevicesVirtualMachineConfiguration([]vz.SocketDeviceConfiguration{vzdev})

	// always add a memory balloon device
	bal, err := vz.NewVirtioTraditionalMemoryBalloonDeviceConfiguration()
	if err != nil {
		return nil, errors.Errorf("creating memory balloon device configuration: %w", err)
	}
	vzVMConfig.SetMemoryBalloonDevicesVirtualMachineConfiguration([]vz.MemoryBalloonDeviceConfiguration{bal})

	slog.DebugContext(ctx, "Validating virtual machine configuration")

	valid, err := vzVMConfig.Validate()
	if err != nil {
		return nil, errors.Errorf("validating virtual machine configuration: %w", err)
	}
	if !valid {
		return nil, errors.New("invalid virtual machine configuration")
	}

	return &VirtualMachineConfiguration{
		id:        id,
		bl:        bootLoader,
		newVMOpts: opts,
		wrapper:   wrapper,
		internal:  vzVMConfig,
		platform:  platformType,
	}, nil
}
