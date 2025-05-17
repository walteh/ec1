package config

import (
	"io"

	"github.com/containers/common/pkg/strongunits"
)

const (
	INJECTED_VM_CACHE_DIR = "INJECTED_VM_CACHE_DIR"
	INJECTED_VM_TMP_DIR   = "INJECTED_VM_TMP_DIR"
)

type EmphericalVMConfig struct {
	memory     strongunits.B
	vcpus      uint
	diskImage  string
	bootLoader Bootloader
	// bootProvisioner
	hostFiles map[string]io.ReadCloser
	devices   []VirtioDevice
}
