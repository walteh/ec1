package machines

const (
	INJECTED_VM_RUNTIME_CACHE_DIR = "INJECTED_VM_RUNTIME_CACHE_DIR"
	INJECTED_VM_BOOT_TMP_DIR      = "INJECTED_VM_BOOT_TMP_DIR"
	INJECTED_VM_RUNTIME_TMP_DIR   = "INJECTED_VM_RUNTIME_TMP_DIR"
)

// type BootLoader interface{}

// type EmphericalVMBootInfo struct {
// 	ID         string
// 	Memory     strongunits.B
// 	Vcpus      uint
// 	BootLoader config.Bootloader
// 	// BootProvisioners    []BootProvisioner
// 	// RuntimeProvisioners []RuntimeProvisioner
// 	Provider         VMIProvider
// 	BootHostFiles    map[string]io.ReadCloser
// 	RuntimeHostFiles map[string]io.ReadCloser
// 	Devices          []virtio.VirtioDevice
// }

// func ResolveEmphericalVMBootInfo(ctx context.Context, provider VMIProvider, vcpus uint, memory strongunits.B) (*EmphericalVMBootInfo, error) {
// 	var err error
// 	id := "vm-" + xid.New().String()

// 	info := &EmphericalVMBootInfo{
// 		ID:       id,
// 		Memory:   memory,
// 		Vcpus:    vcpus,
// 		Provider: provider,
// 		// BootHostFiles:    map[string]io.ReadCloser{},
// 		// RuntimeHostFiles: map[string]io.ReadCloser{},
// 		Devices: []virtio.VirtioDevice{},
// 	}

// 	info.BootLoader, err = EmphericalBootLoaderConfigForGuest(ctx, provider)
// 	if err != nil {
// 		return nil, errors.Errorf("getting boot loader: %w", err)
// 	}

// 	// if d, err := provider.BootProvisioner().VirtioDevices(ctx); err != nil {
// 	// 	return nil, errors.Errorf("getting virtio devices: %w", err)
// 	// } else {
// 	// 	info.Devices = append(info.Devices, d...)
// 	// }

// 	// step 2: add the raw disk image to the machine
// 	return info, nil
// }

// func (me *EmphericalVMBootInfo) AddDevice(dev virtio.VirtioDevice) {
// 	me.Devices = append(me.Devices, dev)
// }

// func EmphericalBootLoaderConfigForGuest(ctx context.Context, provider VMIProvider) (config.Bootloader, error) {
// 	switch kt := provider.GuestKernelType(); kt {
// 	case guest.GuestKernelTypeLinux:
// 		return config.NewEFIBootloader(filepath.Join(config.INJECTED_VM_CACHE_DIR, "efivars.fd"), true), nil
// 	case guest.GuestKernelTypeDarwin:
// 		if mos, ok := provider.(MacOSVMIProvider); ok {
// 			return mos.BootLoaderConfig(), nil
// 		} else {
// 			return nil, errors.New("guest kernel type is darwin but provider does not support macOS")
// 		}
// 	default:
// 		return nil, errors.Errorf("unsupported guest kernel type: %s", kt)
// 	}
// }
