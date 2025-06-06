package vf

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/term/termios"
	"golang.org/x/sys/unix"

	"github.com/Code-Hex/vz/v3"
	"github.com/crc-org/vfkit/pkg/util"

	log "github.com/sirupsen/logrus"

	"github.com/walteh/ec1/pkg/virtio"
	"github.com/walteh/ec1/pkg/vmm"
)

type vzNetworkBlockDevice struct {
	*vz.VirtioBlockDeviceConfiguration
	config *virtio.NetworkBlockDevice
}

func toVzNVMExpressController(dev *virtio.NVMExpressController) (vz.StorageDeviceConfiguration, error) {
	attachment, err := toVzDiskStorageConfig(&dev.DiskStorageConfig)
	if err != nil {
		return nil, err
	}
	devConfig, err := vz.NewNVMExpressControllerDeviceConfiguration(attachment)
	if err != nil {
		return nil, err
	}

	return devConfig, nil
}

func (vmConfig *vzVirtioDeviceApplier) applyNVMExpressController(dev *virtio.NVMExpressController) error {
	storageDeviceConfig, err := toVzNVMExpressController(dev)
	if err != nil {
		return err
	}
	log.Infof("Adding nvme device (imagePath: %s)", dev.ImagePath)
	vmConfig.storageDevicesToSet = append(vmConfig.storageDevicesToSet, storageDeviceConfig)

	return nil
}

func toVzVirtioBlk(dev *virtio.VirtioBlk) (vz.StorageDeviceConfiguration, error) {
	attachment, err := toVzDiskStorageConfig(&dev.DiskStorageConfig)
	if err != nil {
		return nil, err
	}

	devConfig, err := vz.NewVirtioBlockDeviceConfiguration(attachment)
	if err != nil {
		return nil, err
	}

	if dev.DeviceIdentifier != "" {
		err := devConfig.SetBlockDeviceIdentifier(dev.DeviceIdentifier)
		if err != nil {
			return nil, err
		}
	}

	return devConfig, nil
}

func (vmConfig *vzVirtioDeviceApplier) applyVirtioBlk(dev *virtio.VirtioBlk) error {
	storageDeviceConfig, err := toVzVirtioBlk(dev)
	if err != nil {
		return err
	}
	log.Infof("Adding virtio-blk device (imagePath: %s)", dev.ImagePath)
	vmConfig.storageDevicesToSet = append(vmConfig.storageDevicesToSet, storageDeviceConfig)

	return nil
}

func toVzVirtioInput(dev *virtio.VirtioInput) (interface{}, error) {
	var inputConfig interface{}
	if dev.InputType == virtio.VirtioInputPointingDevice {
		inputConfig, err := vz.NewUSBScreenCoordinatePointingDeviceConfiguration()
		if err != nil {
			return nil, fmt.Errorf("failed to create pointing device configuration: %w", err)
		}

		return inputConfig, nil
	}

	inputConfig, err := vz.NewUSBKeyboardConfiguration()
	if err != nil {
		return nil, fmt.Errorf("failed to create keyboard device configuration: %w", err)
	}

	return inputConfig, nil
}

func (vmConfig *vzVirtioDeviceApplier) applyVirtioInput(dev *virtio.VirtioInput) error {
	inputDeviceConfig, err := toVzVirtioInput(dev)
	if err != nil {
		return err
	}

	switch conf := inputDeviceConfig.(type) {
	case vz.PointingDeviceConfiguration:
		log.Info("Adding virtio-input pointing device")
		vmConfig.pointingDevicesToSet = append(vmConfig.pointingDevicesToSet, conf)
	case vz.KeyboardConfiguration:
		log.Info("Adding virtio-input keyboard device")
		vmConfig.keyboardToSet = append(vmConfig.keyboardToSet, conf)
	}

	return nil
}

func newVirtioGraphicsDeviceConfiguration(dev *virtio.VirtioGPU) (vz.GraphicsDeviceConfiguration, error) {
	gpuDeviceConfig, err := vz.NewVirtioGraphicsDeviceConfiguration()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize virtio graphics device: %w", err)
	}
	graphicsScanoutConfig, err := vz.NewVirtioGraphicsScanoutConfiguration(int64(dev.Width), int64(dev.Height))

	if err != nil {
		return nil, fmt.Errorf("failed to create graphics scanout: %w", err)
	}

	gpuDeviceConfig.SetScanouts(
		graphicsScanoutConfig,
	)

	return gpuDeviceConfig, nil
}

func toVzVirtioGPU(dev *virtio.VirtioGPU, useMacOSGPUGraphicsDevice bool) (vz.GraphicsDeviceConfiguration, error) {
	log.Debugf("Setting up graphics device with %vx%v resolution.", dev.Width, dev.Height)

	if useMacOSGPUGraphicsDevice {
		return newMacGraphicsDeviceConfiguration(dev)
	}
	return newVirtioGraphicsDeviceConfiguration(dev)

}

func (vmConfig *vzVirtioDeviceApplier) applyVirtioGPU(dev *virtio.VirtioGPU) error {
	_, isMacOS := vmConfig.bootLoader.(*vmm.MacOSBootloader)
	gpuDeviceConfig, err := toVzVirtioGPU(dev, isMacOS)
	if err != nil {
		return err
	}

	log.Infof("Adding virtio-gpu device")

	vmConfig.graphicsDevicesToSet = append(vmConfig.graphicsDevicesToSet, gpuDeviceConfig)

	return nil
}

func toVzVirtioFs(dev *virtio.VirtioFs) (vz.DirectorySharingDeviceConfiguration, error) {
	if dev.SharedDir == "" {
		return nil, fmt.Errorf("missing mandatory 'sharedDir' option for virtio-fs device")
	}
	var mountTag string
	if dev.MountTag != "" {
		mountTag = dev.MountTag
	} else {
		mountTag = filepath.Base(dev.SharedDir)
	}

	sharedDir, err := vz.NewSharedDirectory(dev.SharedDir, false)
	if err != nil {
		return nil, err
	}
	sharedDirConfig, err := vz.NewSingleDirectoryShare(sharedDir)
	if err != nil {
		return nil, err
	}
	fileSystemDeviceConfig, err := vz.NewVirtioFileSystemDeviceConfiguration(mountTag)
	if err != nil {
		return nil, err
	}
	fileSystemDeviceConfig.SetDirectoryShare(sharedDirConfig)

	return fileSystemDeviceConfig, nil
}

func (vmConfig *vzVirtioDeviceApplier) applyVirtioFs(dev *virtio.VirtioFs) error {
	fileSystemDeviceConfig, err := toVzVirtioFs(dev)
	if err != nil {
		return err
	}
	log.Infof("Adding virtio-fs device")
	vmConfig.directorySharingDevicesToSet = append(vmConfig.directorySharingDevicesToSet, fileSystemDeviceConfig)
	return nil
}

func toVzVirtioRng(dev *virtio.VirtioRng) (*vz.VirtioEntropyDeviceConfiguration, error) {
	return vz.NewVirtioEntropyDeviceConfiguration()
}

func (vmConfig *vzVirtioDeviceApplier) applyVirtioRng(dev *virtio.VirtioRng) error {
	log.Infof("Adding virtio-rng device")
	entropyConfig, err := toVzVirtioRng(dev)
	if err != nil {
		return err
	}
	vmConfig.entropyDevicesToSet = append(vmConfig.entropyDevicesToSet, entropyConfig)

	return nil
}

func toVzVirtioBalloon(dev *virtio.VirtioBalloon) (*vz.VirtioTraditionalMemoryBalloonDeviceConfiguration, error) {
	return vz.NewVirtioTraditionalMemoryBalloonDeviceConfiguration()
}

func unixFd(fd uintptr) int {
	// On unix the underlying fd is int, overflow is not possible.
	return int(fd) //#nosec G115 -- potential integer overflow
}

// https://developer.apple.com/documentation/virtualization/running_linux_in_a_virtual_machine#3880009
func setRawMode(f *os.File) error {
	// Get settings for terminal
	var attr unix.Termios
	if err := termios.Tcgetattr(f.Fd(), &attr); err != nil {
		return err
	}

	// Put stdin into raw mode, disabling local echo, input canonicalization,
	// and CR-NL mapping.
	attr.Iflag &^= unix.ICRNL
	attr.Lflag &^= unix.ICANON | unix.ECHO

	// reflects the changed settings
	return termios.Tcsetattr(f.Fd(), termios.TCSANOW, &attr)
}

func toVzVirtioSerial(dev *virtio.VirtioSerial) (*vz.VirtioConsoleDeviceSerialPortConfiguration, error) {
	var serialPortAttachment vz.SerialPortAttachment
	var retErr error
	switch {
	case dev.UsesStdio:
		var stdin, stdout *os.File
		if dev.FD != 0 {
			fd1, fd2 := dev.FD, dev.FD
			stdin = os.NewFile(uintptr(fd1), "stdin")
			stdout = os.NewFile(uintptr(fd2), "stdout")
		} else {
			stdin = os.Stdin
			stdout = os.Stdout
		}
		if err := setRawMode(stdin); err != nil {
			return nil, err
		}
		serialPortAttachment, retErr = vz.NewFileHandleSerialPortAttachment(stdin, stdout)
	default:
		serialPortAttachment, retErr = vz.NewFileSerialPortAttachment(dev.LogFile, false)
	}
	if retErr != nil {
		return nil, retErr
	}

	return vz.NewVirtioConsoleDeviceSerialPortConfiguration(serialPortAttachment)
}

func toVzVirtioSerialPtyConsole(dev *virtio.VirtioSerial) (*vz.VirtioConsolePortConfiguration, error) {
	master, slave, err := termios.Pty()
	if err != nil {
		return nil, err
	}

	// the master fd and slave fd must stay open for vfkit's lifetime
	util.RegisterExitHandler(func() {
		_ = master.Close()
		_ = slave.Close()
	})

	dev.PtyName = slave.Name()

	if err := setRawMode(master); err != nil {
		return nil, err
	}
	serialPortAttachment, retErr := vz.NewFileHandleSerialPortAttachment(master, master)
	if retErr != nil {
		return nil, retErr
	}
	return vz.NewVirtioConsolePortConfiguration(
		vz.WithVirtioConsolePortConfigurationAttachment(serialPortAttachment),
		vz.WithVirtioConsolePortConfigurationIsConsole(true))
}

func toVzVirtioSerialRawConsole(dev *virtio.VirtioSerial) (*vz.VirtioConsolePortConfiguration, error) {
	fd1, fd2 := dev.FD, dev.FD
	stdin := os.NewFile(uintptr(fd1), "stdin")
	stdout := os.NewFile(uintptr(fd2), "stdout")

	// Set raw mode on stdin
	if err := setRawMode(stdin); err != nil {
		return nil, err
	}

	serialPortAttachment, retErr := vz.NewFileHandleSerialPortAttachment(stdin, stdout)
	if retErr != nil {
		return nil, retErr
	}
	return vz.NewVirtioConsolePortConfiguration(
		vz.WithVirtioConsolePortConfigurationAttachment(serialPortAttachment),
		vz.WithVirtioConsolePortConfigurationIsConsole(true))
}

func (vmConfig *vzVirtioDeviceApplier) applyVirtioSerial(dev *virtio.VirtioSerial) error {
	if dev.LogFile != "" {
		log.Infof("Adding virtio-serial device (logFile: %s)", dev.LogFile)
	}
	// if dev.UsesStdio {
	// 	log.Infof("Adding stdio console")
	// }
	if dev.PtyName != "" {
		return fmt.Errorf("VirtioSerial.PtyName must be empty (current value: %s)", dev.PtyName)
	}

	if dev.UsesStdio {
		consoleConfig, err := toVzVirtioSerial(dev)
		if err != nil {
			return err
		}
		vmConfig.serialPortsToSet = append(vmConfig.serialPortsToSet, consoleConfig)
	} else {
		var consolePortConfig *vz.VirtioConsolePortConfiguration
		var err error
		if dev.FD != 0 {
			consolePortConfig, err = toVzVirtioSerialRawConsole(dev)
		} else {
			consolePortConfig, err = toVzVirtioSerialPtyConsole(dev)
		}
		if err != nil {
			return err
		}
		vmConfig.consolePortsToSet = append(vmConfig.consolePortsToSet, consolePortConfig)
	}

	return nil
}

// func (dev *VirtioVsock) AddToVirtualMachineConfig(vmConfig *vzVitualMachineConfigurationWrapper) error {
// 	if len(vmConfig.socketDevicesConfiguration) != 0 {
// 		log.Debugf("virtio-vsock device already present, not adding a second one")
// 		return nil
// 	}
// 	log.Infof("Adding virtio-vsock device")
// 	vzdev, err := vz.NewVirtioSocketDeviceConfiguration()
// 	if err != nil {
// 		return err
// 	}
// 	vmConfig.socketDevicesConfiguration = append(vmConfig.socketDevicesConfiguration, vzdev)

// 	return nil
// }

func toVzNetworkBlockDevice(dev *virtio.NetworkBlockDevice) (vz.StorageDeviceConfiguration, error) {
	if err := validateNbdURI(dev.URI); err != nil {
		return nil, fmt.Errorf("invalid NBD device 'uri': %s", err.Error())
	}

	if err := validateNbdDeviceIdentifier(dev.DeviceIdentifier); err != nil {
		return nil, fmt.Errorf("invalid NBD device 'deviceId': %s", err.Error())
	}

	attachment, err := vz.NewNetworkBlockDeviceStorageDeviceAttachment(dev.URI, dev.Timeout, dev.ReadOnly, synchronizationModeVZ(dev))
	if err != nil {
		return nil, err
	}

	vzdev, err := vz.NewVirtioBlockDeviceConfiguration(attachment)
	if err != nil {
		return nil, err
	}
	err = vzdev.SetBlockDeviceIdentifier(dev.DeviceIdentifier)
	if err != nil {
		return nil, err
	}

	return vzNetworkBlockDevice{VirtioBlockDeviceConfiguration: vzdev, config: dev}, nil
}

func validateNbdURI(uri string) error {
	if uri == "" {
		return fmt.Errorf("'uri' must be specified")
	}

	parsed, err := url.Parse(uri)
	if err != nil {
		return fmt.Errorf("error: %w", err)
	}

	// The format specified by https://github.com/NetworkBlockDevice/nbd/blob/master/doc/uri.md
	if parsed.Scheme != "nbd" && parsed.Scheme != "nbds" && parsed.Scheme != "nbd+unix" && parsed.Scheme != "nbds+unix" {
		return fmt.Errorf("invalid scheme: %s. Expected one of: 'nbd', 'nbds', 'nbd+unix', or 'nbds+unix'", parsed.Scheme)
	}

	return nil
}

func validateNbdDeviceIdentifier(deviceID string) error {
	if deviceID == "" {
		return fmt.Errorf("'deviceId' must be specified")
	}

	if strings.Contains(deviceID, "/") {
		return fmt.Errorf("invalid 'deviceId': it cannot contain any forward slash")
	}

	if len(deviceID) > 255 {
		return fmt.Errorf("invalid 'deviceId': exceeds maximum length")
	}

	return nil
}

func synchronizationModeVZ(dev *virtio.NetworkBlockDevice) vz.DiskSynchronizationMode {
	if dev.SynchronizationMode == virtio.SynchronizationNoneMode {
		return vz.DiskSynchronizationModeNone
	}
	return vz.DiskSynchronizationModeFull
}

func (vmConfig *vzVirtioDeviceApplier) applyNetworkBlockDevice(dev *virtio.NetworkBlockDevice) error {
	storageDeviceConfig, err := toVzNetworkBlockDevice(dev)
	if err != nil {
		return err
	}
	log.Infof("Adding NBD device (uri: %s, deviceId: %s)", dev.URI, dev.DeviceIdentifier)
	vmConfig.storageDevicesToSet = append(vmConfig.storageDevicesToSet, storageDeviceConfig)

	return nil
}

// func AddToVirtualMachineConfig(vmConfig *vzVirtioConverter, dev virtio.VirtioDevice) error {
// 	switch d := dev.(type) {
// 	case *virtio.USBMassStorage:
// 		return (*USBMassStorage)(d).AddToVirtualMachineConfig(vmConfig)
// 	case *virtio.VirtioBlk:
// 		return (*VirtioBlk)(d).AddToVirtualMachineConfig(vmConfig)
// 	case *virtio.RosettaShare:
// 		return (*RosettaShare)(d).AddToVirtualMachineConfig(vmConfig)
// 	case *virtio.NVMExpressController:
// 		return (*NVMExpressController)(d).AddToVirtualMachineConfig(vmConfig)
// 	case *virtio.VirtioFs:
// 		return (*VirtioFs)(d).AddToVirtualMachineConfig(vmConfig)
// 	case *virtio.VirtioNet:
// 		dev := VirtioNet{VirtioNet: d}
// 		return dev.AddToVirtualMachineConfig(vmConfig)
// 	case *virtio.VirtioRng:
// 		return (*VirtioRng)(d).AddToVirtualMachineConfig(vmConfig)
// 	case *virtio.VirtioSerial:
// 		return (*VirtioSerial)(d).AddToVirtualMachineConfig(vmConfig)
// 	case *virtio.VirtioVsock:
// 		// return (*VirtioVsock)(d).AddToVirtualMachineConfig(vmConfig)
// 		return nil // vsocks get handled by the host via a proxy
// 	case *virtio.VirtioInput:
// 		return (*VirtioInput)(d).AddToVirtualMachineConfig(vmConfig)
// 	case *virtio.VirtioGPU:
// 		return (*VirtioGPU)(d).AddToVirtualMachineConfig(vmConfig)
// 	case *virtio.VirtioBalloon:
// 		return (*VirtioBalloon)(d).AddToVirtualMachineConfig(vmConfig)
// 	case *virtio.NetworkBlockDevice:
// 		return (*NetworkBlockDevice)(d).AddToVirtualMachineConfig(vmConfig)
// 	default:
// 		return fmt.Errorf("Unexpected virtio device type: %T", d)
// 	}
// }

func toVzDiskStorageConfig(dev *virtio.DiskStorageConfig) (vz.StorageDeviceAttachment, error) {
	if dev.ImagePath == "" {
		return nil, fmt.Errorf("missing mandatory 'path' option for %s device", dev.DevName)
	}
	syncMode := vz.DiskImageSynchronizationModeFsync
	caching := vz.DiskImageCachingModeCached

	// return vz.NewDiskBlockDeviceStorageDeviceAttachment(config.=, config.ReadOnly, syncMode)

	return vz.NewDiskImageStorageDeviceAttachmentWithCacheAndSync(dev.ImagePath, dev.ReadOnly, caching, syncMode)
}

func toVzUSBMassStorage(dev *virtio.USBMassStorage) (vz.StorageDeviceConfiguration, error) {
	attachment, err := toVzDiskStorageConfig(&dev.DiskStorageConfig)
	if err != nil {
		return nil, err
	}
	return vz.NewUSBMassStorageDeviceConfiguration(attachment)
}

func (vmConfig *vzVirtioDeviceApplier) applyUSBMassStorage(dev *virtio.USBMassStorage) error {
	storageDeviceConfig, err := toVzUSBMassStorage(dev)
	if err != nil {
		return err
	}
	log.Infof("Adding USB mass storage device (imagePath: %s)", dev.ImagePath)
	vmConfig.storageDevicesToSet = append(vmConfig.storageDevicesToSet, storageDeviceConfig)

	return nil
}
