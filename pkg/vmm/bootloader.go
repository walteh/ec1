package vmm

// LinuxBootloader determines which kernel/initrd/kernel args to use when starting
// the virtual machine.
type LinuxBootloader struct {
	VmlinuzPath   string `json:"vmlinuzPath"`
	KernelCmdLine string `json:"kernelCmdLine"`
	InitrdPath    string `json:"initrdPath"`
}

// EFIBootloader allows to set a few options related to EFI variable storage
type EFIBootloader struct {
	EFIVariableStorePath string `json:"efiVariableStorePath"`
	// TODO: virtualization framework allow both create and overwrite
	CreateVariableStore bool `json:"createVariableStore"`
}

// MacOSBootloader provides necessary objects for booting macOS guests
type MacOSBootloader struct {
	MachineIdentifierPath string `json:"machineIdentifierPath"`
	HardwareModelPath     string `json:"hardwareModelPath"`
	AuxImagePath          string `json:"auxImagePath"`
}

// NewEFIBootloader creates a new bootloader to start a VM using EFI
// efiVariableStorePath is the path to a file for EFI storage
// create is a boolean indicating if the file for the store should be created or not
func NewEFIBootloader(efiVariableStorePath string, createVariableStore bool) *EFIBootloader {
	return &EFIBootloader{
		EFIVariableStorePath: efiVariableStorePath,
		CreateVariableStore:  createVariableStore,
	}
}

type Bootloader interface {
	isBootloader()
}

func (bootloader *LinuxBootloader) isBootloader() {}
func (bootloader *EFIBootloader) isBootloader()   {}
func (bootloader *MacOSBootloader) isBootloader() {}
