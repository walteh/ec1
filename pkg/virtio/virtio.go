package virtio

import (
	"fmt"
	"math"
	"time"
)

type VirtioDevices []VirtioDevice

func VirtioDevicesOfType[T VirtioDevice](devices VirtioDevices) []T {
	var result []T
	for _, device := range devices {
		if device, ok := device.(T); ok {
			result = append(result, device)
		}
	}

	return result
}

// The VirtioDevice interface is an interface which is implemented by all virtio devices.
type VirtioDevice interface {
	isVirtioDevice()
}

// func is[T VirtioDevice](v ...VirtioDevice) T { return *new(T) }

const (
	// Possible values for VirtioInput.InputType
	VirtioInputPointingDevice = "pointing"
	VirtioInputKeyboardDevice = "keyboard"

	// Options for VirtioGPUResolution
	VirtioGPUResolutionWidth  = "width"
	VirtioGPUResolutionHeight = "height"

	// Default VirtioGPU Resolution
	defaultVirtioGPUResolutionWidth  = 800
	defaultVirtioGPUResolutionHeight = 600
)

// VirtioInput configures an input device, such as a keyboard or pointing device
// (mouse) that the virtual machine can use
type VirtioInput struct {
	InputType string `json:"inputType"` // currently supports "pointing" and "keyboard" input types
}

func (v *VirtioInput) isVirtioDevice() {}

var _ VirtioDevice = &VirtioInput{}

type VirtioGPUResolution struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

func (v *VirtioGPUResolution) isVirtioDevice() {}

var _ VirtioDevice = &VirtioGPUResolution{}

// VirtioGPU configures a GPU device, such as the host computer's display
type VirtioGPU struct {
	UsesGUI bool `json:"usesGUI"`
	VirtioGPUResolution
	IsMacOS bool `json:"isMacOS"`
}

func (v *VirtioGPU) isVirtioDevice() {}

var _ VirtioDevice = &VirtioGPU{}

type VirtioVsockDirection string

const (
	VirtioVsockDirectionGuestListensAsServer  VirtioVsockDirection = "guest-listens-as-server"
	VirtioVsockDirectionGuestConnectsAsClient VirtioVsockDirection = "guest-connects-as-client"
)

// VirtioVsock configures of a virtio-vsock device allowing 2-way communication
// between the host and the virtual machine type
type VirtioVsock struct {
	// Port is the virtio-vsock port used for this device, see `man vsock` for more
	// details.
	Port uint32 `json:"port"`
	// SocketURL is the path to a unix socket on the host to use for the virtio-vsock communication with the guest.
	SocketURL string `json:"socketURL"`
	// If true, vsock connections will have to be done from guest to host. If false, vsock connections will only be possible
	// from host to guest
	Direction VirtioVsockDirection `json:"direction,omitempty"`
}

func (v *VirtioVsock) isVirtioDevice() {}

// VirtioBlk configures a disk device.
type VirtioBlk struct {
	DiskStorageConfig
	DeviceIdentifier string `json:"deviceIdentifier,omitempty"`
}

var _ VirtioDevice = &VirtioBlk{}

func (v *VirtioBlk) isVirtioDevice() {}

type DirectorySharingConfig struct {
	MountTag string `json:"mountTag"`
}

var _ VirtioDevice = &DirectorySharingConfig{}

func (v *DirectorySharingConfig) isVirtioDevice() {}

// VirtioFs configures directory sharing between the guest and the host.
type VirtioFs struct {
	DirectorySharingConfig
	SharedDir string `json:"sharedDir"`
}

var _ VirtioDevice = &VirtioFs{}

func (v *VirtioFs) isVirtioDevice() {}

// RosettaShare configures rosetta support in the guest to run Intel binaries on Apple CPUs
type RosettaShare struct {
	DirectorySharingConfig
	InstallRosetta  bool `json:"installRosetta"`
	IgnoreIfMissing bool `json:"ignoreIfMissing"`
}

var _ VirtioDevice = &RosettaShare{}

func (v *RosettaShare) isVirtioDevice() {}

// NVMExpressController configures a NVMe controller in the guest
type NVMExpressController struct {
	DiskStorageConfig
}

var _ VirtioDevice = &NVMExpressController{}

func (v *NVMExpressController) isVirtioDevice() {}

// VirtioRng configures a random number generator (RNG) device.
type VirtioRng struct {
}

var _ VirtioDevice = &VirtioRng{}

func (v *VirtioRng) isVirtioDevice() {}

// VirtioSerial configures the virtual machine serial ports.
type VirtioSerial struct {
	LogFile   string `json:"logFile,omitempty"`
	UsesStdio bool   `json:"usesStdio,omitempty"`
	UsesPty   bool   `json:"usesPty,omitempty"`
	// PtyName must not be set when creating the VM, from a user perspective, it's read-only,
	// vfkit will set it during VM startup.
	PtyName string  `json:"ptyName,omitempty"`
	FD      uintptr `json:"fd,omitempty"`
}

var _ VirtioDevice = &VirtioSerial{}

func (v *VirtioSerial) isVirtioDevice() {}

type NBDSynchronizationMode string

const (
	SynchronizationFullMode NBDSynchronizationMode = "full"
	SynchronizationNoneMode NBDSynchronizationMode = "none"
)

var _ VirtioDevice = &NetworkBlockDevice{}

func (v *NetworkBlockDevice) isVirtioDevice() {}

type NetworkBlockDevice struct {
	NetworkBlockStorageConfig
	DeviceIdentifier    string
	Timeout             time.Duration
	SynchronizationMode NBDSynchronizationMode
}

var _ VirtioDevice = &VirtioBalloon{}

func (v *VirtioBalloon) isVirtioDevice() {}

type VirtioBalloon struct{}

func VirtioBalloonNew() (VirtioDevice, error) {
	return &VirtioBalloon{}, nil
}

var _ VirtioDevice = &USBMassStorage{}

func (v *USBMassStorage) isVirtioDevice() {}

func USBMassStorageNewEmpty() (VirtioDevice, error) {
	return &USBMassStorage{}, nil
}

// VirtioSerialNew creates a new serial device for the virtual machine. The
// output the virtual machine sent to the serial port will be written to the
// file at logFilePath.
func VirtioSerialNew(logFilePath string) (VirtioDevice, error) {
	return &VirtioSerial{
		LogFile: logFilePath,
	}, nil
}

func VirtioSerialNewStdio() (VirtioDevice, error) {
	return &VirtioSerial{
		UsesStdio: true,
	}, nil
}

func VirtioSerialNewPty() (VirtioDevice, error) {
	return &VirtioSerial{
		UsesPty: true,
	}, nil
}

func (dev *VirtioSerial) validate() error {
	if dev.LogFile != "" && dev.UsesStdio {
		return fmt.Errorf("'logFilePath' and 'stdio' cannot be set at the same time")
	}
	if dev.LogFile != "" && dev.UsesPty {
		return fmt.Errorf("'logFilePath' and 'pty' cannot be set at the same time")
	}
	if dev.UsesStdio && dev.UsesPty {
		return fmt.Errorf("'stdio' and 'pty' cannot be set at the same time")
	}
	if dev.LogFile == "" && !dev.UsesStdio && !dev.UsesPty {
		return fmt.Errorf("one of 'logFilePath', 'stdio' or 'pty' must be set")
	}

	return nil
}

// VirtioInputNew creates a new input device for the virtual machine.
// The inputType parameter is the type of virtio-input device that will be added
// to the machine.
func VirtioInputNew(inputType string) (VirtioDevice, error) {
	dev := &VirtioInput{
		InputType: inputType,
	}
	if err := dev.validate(); err != nil {
		return nil, err
	}

	return dev, nil
}

func (dev *VirtioInput) validate() error {
	if dev.InputType != VirtioInputPointingDevice && dev.InputType != VirtioInputKeyboardDevice {
		return fmt.Errorf("unknown option for virtio-input devices: %s", dev.InputType)
	}

	return nil
}

// func (dev *VirtioInput) ToCmdLine() ([]string, error) {
// 	if err := dev.validate(); err != nil {
// 		return nil, err
// 	}

// 	return []string{"--device", fmt.Sprintf("virtio-input,%s", dev.InputType)}, nil
// }

// func (dev *VirtioInput) FromOptions(options []option) error {
// 	for _, option := range options {
// 		switch option.key {
// 		case VirtioInputPointingDevice, VirtioInputKeyboardDevice:
// 			if option.value != "" {
// 				return fmt.Errorf("unexpected value for virtio-input %s option: %s", option.key, option.value)
// 			}
// 			dev.InputType = option.key
// 		default:
// 			return fmt.Errorf("unknown option for virtio-input devices: %s", option.key)
// 		}
// 	}
// 	return dev.validate()
// }

// VirtioGPUNew creates a new gpu device for the virtual machine.
// The usesGUI parameter determines whether a graphical application window will
// be displayed
func VirtioGPUNew() (VirtioDevice, error) {
	return &VirtioGPU{
		UsesGUI: false,
		VirtioGPUResolution: VirtioGPUResolution{
			Width:  defaultVirtioGPUResolutionWidth,
			Height: defaultVirtioGPUResolutionHeight,
		},
	}, nil
}

func (dev *VirtioGPU) validate() error {
	if dev.Height < 1 || dev.Width < 1 {
		return fmt.Errorf("invalid dimensions for virtio-gpu device resolution: %dx%d", dev.Width, dev.Height)
	}

	return nil
}

// VirtioRngNew creates a new random number generator device to feed entropy
// into the virtual machine.
func VirtioRngNew() (VirtioDevice, error) {
	return &VirtioRng{}, nil
}

func nvmExpressControllerNewEmpty() *NVMExpressController {
	return &NVMExpressController{
		DiskStorageConfig: DiskStorageConfig{
			StorageConfig: StorageConfig{
				DevName: "nvme",
			},
		},
	}
}

// NVMExpressControllerNew creates a new NVMExpress controller to use in the
// virtual machine. It will use the file at imagePath as the disk image. This
// image must be in raw format.
func NVMExpressControllerNew(imagePath string) (*NVMExpressController, error) {
	r := nvmExpressControllerNewEmpty()
	r.ImagePath = imagePath
	return r, nil
}

func virtioBlkNewEmpty() *VirtioBlk {
	return &VirtioBlk{
		DiskStorageConfig: DiskStorageConfig{
			StorageConfig: StorageConfig{
				DevName: "virtio-blk",
			},
		},
		DeviceIdentifier: "",
	}
}

// VirtioBlkNew creates a new disk to use in the virtual machine. It will use
// the file at imagePath as the disk image. This image must be in raw format.
func VirtioBlkNew(imagePath string) (*VirtioBlk, error) {
	virtioBlk := virtioBlkNewEmpty()
	virtioBlk.ImagePath = imagePath

	return virtioBlk, nil
}

func (dev *VirtioBlk) SetDeviceIdentifier(devID string) {
	dev.DeviceIdentifier = devID
}

// VirtioVsockNew creates a new virtio-vsock device for 2-way communication
// between the host and the virtual machine. The communication will happen on
// vsock port, and on the host it will use the unix socket at socketURL.
// When listen is true, the host will be listening for connections over vsock.
// When listen  is false, the guest will be listening for connections over vsock.
func VirtioVsockNew(port uint, socketURL string, shouldHostActAsServer bool) (VirtioDevice, error) {
	if port > math.MaxUint32 {
		return nil, fmt.Errorf("invalid vsock port: %d", port)
	}
	direction := VirtioVsockDirectionGuestListensAsServer
	if shouldHostActAsServer {
		direction = VirtioVsockDirectionGuestConnectsAsClient
	}
	return &VirtioVsock{
		Port:      uint32(port), //#nosec G115 -- was compared to math.MaxUint32
		SocketURL: socketURL,
		Direction: direction,
	}, nil
}

// func (dev *VirtioVsock) ToCmdLine() ([]string, error) {
// 	if dev.Port == 0 || dev.SocketURL == "" {
// 		return nil, fmt.Errorf("virtio-vsock needs both a port and a socket URL")
// 	}
// 	var listenStr string
// 	if dev.Direction == VirtioVsockDirectionGuestConnectsAsClient {
// 		// the listenStr is for the host, so it's the opposite of the direction
// 		listenStr = "listen"
// 	} else {
// 		listenStr = "connect"
// 	}
// 	return []string{"--device", fmt.Sprintf("virtio-vsock,port=%d,socketURL=%s,%s", dev.Port, dev.SocketURL, listenStr)}, nil
// }

// func (dev *VirtioVsock) FromOptions(options []option) error {
// 	// default to listen for backwards compatibliity
// 	dev.Direction = VirtioVsockDirectionGuestListensAsServer
// 	for _, option := range options {
// 		switch option.key {
// 		case "socketURL":
// 			dev.SocketURL = option.value
// 		case "port":
// 			port, err := strconv.ParseUint(option.value, 10, 32)
// 			if err != nil {
// 				return err
// 			}
// 			dev.Port = uint32(port) //#nosec G115 -- ParseUint(_, _, 32) guarantees no overflow
// 		case "listen":
// 			dev.Direction = VirtioVsockDirectionGuestListensAsServer
// 		case "connect":
// 			dev.Direction = VirtioVsockDirectionGuestConnectsAsClient
// 		default:
// 			return fmt.Errorf("unknown option for virtio-vsock devices: %s", option.key)
// 		}
// 	}
// 	return nil
// }

// VirtioFsNew creates a new virtio-fs device for file sharing. It will share
// the directory at sharedDir with the virtual machine. This directory can be
// mounted in the VM using `mount -t virtiofs mountTag /some/dir`
func VirtioFsNew(sharedDir string, mountTag string) (VirtioDevice, error) {
	return &VirtioFs{
		DirectorySharingConfig: DirectorySharingConfig{
			MountTag: mountTag,
		},
		SharedDir: sharedDir,
	}, nil
}

// func (dev *VirtioFs) ToCmdLine() ([]string, error) {
// 	if dev.SharedDir == "" {
// 		return nil, fmt.Errorf("virtio-fs needs the path to the directory to share")
// 	}
// 	if dev.MountTag != "" {
// 		return []string{"--device", fmt.Sprintf("virtio-fs,sharedDir=%s,mountTag=%s", dev.SharedDir, dev.MountTag)}, nil
// 	}

// 	return []string{"--device", fmt.Sprintf("virtio-fs,sharedDir=%s", dev.SharedDir)}, nil
// }

// func (dev *VirtioFs) FromOptions(options []option) error {
// 	for _, option := range options {
// 		switch option.key {
// 		case "sharedDir":
// 			dev.SharedDir = option.value
// 		case "mountTag":
// 			dev.MountTag = option.value
// 		default:
// 			return fmt.Errorf("unknown option for virtio-fs devices: %s", option.key)
// 		}
// 	}
// 	return nil
// }

// RosettaShareNew RosettaShare creates a new rosetta share for running x86_64 binaries on M1 machines.
// It will share a directory containing the linux rosetta binaries with the
// virtual machine. This directory can be mounted in the VM using `mount -t
// virtiofs mountTag /some/dir`
func RosettaShareNew(mountTag string) (VirtioDevice, error) {
	return &RosettaShare{
		DirectorySharingConfig: DirectorySharingConfig{
			MountTag: mountTag,
		},
	}, nil
}

// func (dev *RosettaShare) ToCmdLine() ([]string, error) {
// 	if dev.MountTag == "" {
// 		return nil, fmt.Errorf("rosetta shares require a mount tag to be specified")
// 	}
// 	builder := strings.Builder{}
// 	builder.WriteString("rosetta")
// 	fmt.Fprintf(&builder, ",mountTag=%s", dev.MountTag)
// 	if dev.InstallRosetta {
// 		builder.WriteString(",install")
// 	}
// 	if dev.IgnoreIfMissing {
// 		builder.WriteString(",ignore-if-missing")
// 	}

// 	return []string{"--device", builder.String()}, nil
// }

// func (dev *RosettaShare) FromOptions(options []option) error {
// 	for _, option := range options {
// 		switch option.key {
// 		case "mountTag":
// 			dev.MountTag = option.value
// 		case "install":
// 			dev.InstallRosetta = true
// 		case "ignore-if-missing":
// 			dev.IgnoreIfMissing = true
// 		default:
// 			return fmt.Errorf("unknown option for rosetta share: %s", option.key)
// 		}
// 	}
// 	return nil
// }

func networkBlockDeviceNewEmpty() *NetworkBlockDevice {
	return &NetworkBlockDevice{
		NetworkBlockStorageConfig: NetworkBlockStorageConfig{
			StorageConfig: StorageConfig{
				DevName: "nbd",
			},
		},
		DeviceIdentifier:    "",
		Timeout:             time.Duration(15000 * time.Millisecond), // set a default timeout to 15s
		SynchronizationMode: SynchronizationFullMode,                 // default mode to full
	}
}

// NetworkBlockDeviceNew creates a new disk by connecting to a remote Network Block Device (NBD) server.
// The provided uri must be in the format <scheme>://<address>/<export-name>
// where scheme could have any of these value: nbd, nbds, nbd+unix and nbds+unix.
// More info can be found at https://github.com/NetworkBlockDevice/nbd/blob/master/doc/uri.md
// This allows the virtual machine to access and use the remote storage as if it were a local disk.
func NetworkBlockDeviceNew(uri string, timeout uint32, synchronization NBDSynchronizationMode) (*NetworkBlockDevice, error) {
	nbd := networkBlockDeviceNewEmpty()
	nbd.URI = uri
	nbd.Timeout = time.Duration(timeout) * time.Millisecond
	nbd.SynchronizationMode = synchronization

	return nbd, nil
}

// func (nbd *NetworkBlockDevice) ToCmdLine() ([]string, error) {
// 	cmdLine, err := nbd.NetworkBlockStorageConfig.ToCmdLine()
// 	if err != nil {
// 		return []string{}, err
// 	}
// 	if len(cmdLine) != 2 {
// 		return []string{}, fmt.Errorf("unexpected storage config commandline")
// 	}

// 	if nbd.DeviceIdentifier != "" {
// 		cmdLine[1] = fmt.Sprintf("%s,deviceId=%s", cmdLine[1], nbd.DeviceIdentifier)
// 	}
// 	if nbd.Timeout.Milliseconds() > 0 {
// 		cmdLine[1] = fmt.Sprintf("%s,timeout=%d", cmdLine[1], nbd.Timeout.Milliseconds())
// 	}
// 	if nbd.SynchronizationMode == "none" || nbd.SynchronizationMode == "full" {
// 		cmdLine[1] = fmt.Sprintf("%s,sync=%s", cmdLine[1], nbd.SynchronizationMode)
// 	}

// 	return cmdLine, nil
// }

// func (nbd *NetworkBlockDevice) FromOptions(options []option) error {
// 	unhandledOpts := []option{}
// 	for _, option := range options {
// 		switch option.key {
// 		case "deviceId":
// 			nbd.DeviceIdentifier = option.value
// 		case "timeout":
// 			timeoutMS, err := strconv.ParseInt(option.value, 10, 32)
// 			if err != nil {
// 				return err
// 			}
// 			nbd.Timeout = time.Duration(timeoutMS) * time.Millisecond
// 		case "sync":
// 			switch option.value {
// 			case string(SynchronizationFullMode):
// 				nbd.SynchronizationMode = SynchronizationFullMode
// 			case string(SynchronizationNoneMode):
// 				nbd.SynchronizationMode = SynchronizationNoneMode
// 			default:
// 				return fmt.Errorf("invalid sync mode: %s, must be 'full' or 'none'", option.value)
// 			}
// 		default:
// 			unhandledOpts = append(unhandledOpts, option)
// 		}
// 	}

// 	return nbd.NetworkBlockStorageConfig.FromOptions(unhandledOpts)
// }

type USBMassStorage struct {
	DiskStorageConfig
}

func usbMassStorageNewEmpty() *USBMassStorage {
	return &USBMassStorage{
		DiskStorageConfig: DiskStorageConfig{
			StorageConfig: StorageConfig{
				DevName: "usb-mass-storage",
			},
		},
	}
}

// USBMassStorageNew creates a new USB disk to use in the virtual machine. It will use
// the file at imagePath as the disk image. This image must be in raw or ISO format.
func USBMassStorageNew(imagePath string) (*USBMassStorage, error) {
	usbMassStorage := usbMassStorageNewEmpty()
	usbMassStorage.ImagePath = imagePath

	return usbMassStorage, nil
}

func (dev *USBMassStorage) SetReadOnly(readOnly bool) {
	dev.StorageConfig.ReadOnly = readOnly
}

// StorageConfig configures a disk device.
type StorageConfig struct {
	DevName  string `json:"devName"`
	ReadOnly bool   `json:"readOnly,omitempty"`
}

type DiskStorageConfig struct {
	StorageConfig
	ImagePath string `json:"imagePath,omitempty"`
}

type NetworkBlockStorageConfig struct {
	StorageConfig
	URI string `json:"uri,omitempty"`
}

// func (config *DiskStorageConfig) ToCmdLine() ([]string, error) {
// 	if config.ImagePath == "" {
// 		return nil, fmt.Errorf("%s devices need the path to a disk image", config.DevName)
// 	}

// 	value := fmt.Sprintf("%s,path=%s", config.DevName, config.ImagePath)

// 	if config.ReadOnly {
// 		value += ",readonly"
// 	}
// 	return []string{"--device", value}, nil
// }

// func (config *DiskStorageConfig) FromOptions(options []option) error {
// 	for _, option := range options {
// 		switch option.key {
// 		case "path":
// 			config.ImagePath = option.value
// 		case "readonly":
// 			if option.value != "" {
// 				return fmt.Errorf("unexpected value for virtio-blk 'readonly' option: %s", option.value)
// 			}
// 			config.ReadOnly = true
// 		default:
// 			return fmt.Errorf("unknown option for %s devices: %s", config.DevName, option.key)
// 		}
// 	}
// 	return nil
// }

// func (config *NetworkBlockStorageConfig) ToCmdLine() ([]string, error) {
// 	if config.URI == "" {
// 		return nil, fmt.Errorf("%s devices need the uri to a remote block device", config.DevName)
// 	}

// 	value := fmt.Sprintf("%s,uri=%s", config.DevName, config.URI)

// 	if config.ReadOnly {
// 		value += ",readonly"
// 	}
// 	return []string{"--device", value}, nil
// }

// func (config *NetworkBlockStorageConfig) FromOptions(options []option) error {
// 	for _, option := range options {
// 		switch option.key {
// 		case "uri":
// 			config.URI = option.value
// 		case "readonly":
// 			if option.value != "" {
// 				return fmt.Errorf("unexpected value for virtio-blk 'readonly' option: %s", option.value)
// 			}
// 			config.ReadOnly = true
// 		default:
// 			return fmt.Errorf("unknown option for %s devices: %s", config.DevName, option.key)
// 		}
// 	}
// 	return nil
// }
