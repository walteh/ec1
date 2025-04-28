package guest

import (
	"context"
	"path/filepath"

	"github.com/walteh/ec1/pkg/hypervisors/vf/config"
	"gitlab.com/tozd/go/errors"
)

type GuestKernelType string

const (
	GuestKernelTypeLinux   GuestKernelType = "linux"
	GuestKernelTypeWindows GuestKernelType = "windows"
	GuestKernelTypeDarwin  GuestKernelType = "darwin"
)

func (t GuestKernelType) EmphericalBootLoaderConfig(ctx context.Context, cacheDir string) (config.Bootloader, error) {
	switch t {
	case GuestKernelTypeLinux:
		return config.NewEFIBootloader(filepath.Join(cacheDir, "efivars.fd"), true), nil
	case GuestKernelTypeDarwin:
		return &config.MacOSBootloader{
			MachineIdentifierPath: filepath.Join(cacheDir, "machine-identifier.bin"),
			HardwareModelPath:     filepath.Join(cacheDir, "hardware-model.bin"),
			AuxImagePath:          filepath.Join(cacheDir, "aux.img"),
		}, nil
	}
	return nil, errors.New("unsupported guest kernel type")
}
