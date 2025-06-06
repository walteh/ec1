package guest

type GuestKernelType string

const (
	GuestKernelTypeLinux     GuestKernelType = "linux"
	GuestKernelTypeWindows   GuestKernelType = "windows"
	GuestKernelTypeDarwin    GuestKernelType = "darwin"
	GuestKernelTypeBareMetal GuestKernelType = "baremetal"
)

// func (t GuestKernelType) EmphericalBootLoaderConfig(ctx context.Context, cacheDir string) (bootloader.Bootloader, error) {
// 	switch t {
// 	case GuestKernelTypeLinux:
// 		return bootloader.NewEFIBootloader(filepath.Join(cacheDir, "efivars.fd"), true), nil
// 	case GuestKernelTypeDarwin:
// 		return &bootloader.MacOSBootloader{
// 			MachineIdentifierPath: filepath.Join(cacheDir, "machine-identifier.bin"),
// 			HardwareModelPath:     filepath.Join(cacheDir, "hardware-model.bin"),
// 			AuxImagePath:          filepath.Join(cacheDir, "aux.img"),
// 		}, nil
// 	}
// 	return nil, errors.New("unsupported guest kernel type")
// }
