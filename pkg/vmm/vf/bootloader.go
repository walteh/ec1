package vf

import (
	"fmt"
	"runtime"

	"github.com/Code-Hex/vz/v3"

	"github.com/walteh/ec1/pkg/bootloader"
	"github.com/walteh/ec1/pkg/host"
)

func toVzLinuxBootloader(bootloader *bootloader.LinuxBootloader) (vz.BootLoader, error) {
	if runtime.GOARCH == "arm64" {
		uncompressed, err := host.IsKernelUncompressed(bootloader.VmlinuzPath)
		if err != nil {
			return nil, err
		}
		if !uncompressed {
			return nil, fmt.Errorf("kernel must be uncompressed, %s is a compressed file", bootloader.VmlinuzPath)
		}
	}

	return vz.NewLinuxBootLoader(
		bootloader.VmlinuzPath,
		vz.WithCommandLine(bootloader.KernelCmdLine),
		vz.WithInitrd(bootloader.InitrdPath),
	)
}

func toVzEFIBootloader(bootloader *bootloader.EFIBootloader) (vz.BootLoader, error) {
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

func toVzBootloader(bl bootloader.Bootloader) (vz.BootLoader, error) {

	switch b := bl.(type) {
	case *bootloader.LinuxBootloader:
		return toVzLinuxBootloader(b)
	case *bootloader.EFIBootloader:
		return toVzEFIBootloader(b)
	case *bootloader.MacOSBootloader:
		return toVzMacOSBootloader(b)
	default:
		return nil, fmt.Errorf("Unexpected bootloader type: %T", b)
	}
}
