package vf

import (
	"fmt"
	"runtime"

	"github.com/Code-Hex/vz/v3"

	"github.com/walteh/ec1/pkg/host"
	"github.com/walteh/ec1/pkg/vmm"
)

func toVzLinuxBootloader(bootloader *vmm.LinuxBootloader) (vz.BootLoader, error) {
	if runtime.GOARCH == "arm64" {
		uncompressed, err := host.IsKernelUncompressed(bootloader.VmlinuzPath)
		if err != nil {
			return nil, err
		}
		if !uncompressed {
			return nil, fmt.Errorf("kernel must be uncompressed, %s is a compressed file", bootloader.VmlinuzPath)
		}
	}

	opts := []vz.LinuxBootLoaderOption{}
	if bootloader.InitrdPath != "" {
		opts = append(opts, vz.WithInitrd(bootloader.InitrdPath))
	}
	if bootloader.KernelCmdLine != "" {
		opts = append(opts, vz.WithCommandLine(bootloader.KernelCmdLine))
	}

	return vz.NewLinuxBootLoader(
		bootloader.VmlinuzPath,
		opts...,
	)
}

func toVzEFIBootloader(bootloader *vmm.EFIBootloader) (vz.BootLoader, error) {
	var efiVariableStore *vz.EFIVariableStore
	var err error

	if bootloader.CreateVariableStore {
		efiVariableStore, err = vz.NewEFIVariableStore(bootloader.EFIVariableStorePath, vz.WithCreatingEFIVariableStore())
	} else {
		efiVariableStore, err = vz.NewEFIVariableStore(bootloader.EFIVariableStorePath)
	}
	if err != nil {
		return nil, err
	}

	return vz.NewEFIBootLoader(
		vz.WithEFIVariableStore(efiVariableStore),
	)
}

func toVzBootloader(bl vmm.Bootloader) (vz.BootLoader, error) {

	switch b := bl.(type) {
	case *vmm.LinuxBootloader:
		return toVzLinuxBootloader(b)
	case *vmm.EFIBootloader:
		return toVzEFIBootloader(b)
	case *vmm.MacOSBootloader:
		return toVzMacOSBootloader(b)
	default:
		return nil, fmt.Errorf("Unexpected bootloader type: %T", b)
	}
}
