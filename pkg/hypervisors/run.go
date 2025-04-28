package hypervisors

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/containers/common/pkg/strongunits"
	"github.com/rs/xid"
	"github.com/walteh/ec1/pkg/hypervisors/vf/config"
	"github.com/walteh/ec1/pkg/machines/guest"
	"github.com/walteh/ec1/pkg/machines/host"
	"github.com/walteh/ec1/pkg/machines/virtio"
	"gitlab.com/tozd/go/errors"
	"golang.org/x/sync/errgroup"
)

func EmphericalBootLoaderConfigForGuest(ctx context.Context, provider VMIProvider) (config.Bootloader, error) {
	switch kt := provider.GuestKernelType(); kt {
	case guest.GuestKernelTypeLinux:
		return config.NewEFIBootloader(filepath.Join(config.INJECTED_VM_CACHE_DIR, "efivars.fd"), true), nil
	case guest.GuestKernelTypeDarwin:
		if mos, ok := provider.(MacOSVMIProvider); ok {
			return mos.BootLoaderConfig(), nil
		} else {
			return nil, errors.New("guest kernel type is darwin but provider does not support macOS")
		}
	default:
		return nil, errors.Errorf("unsupported guest kernel type: %s", kt)
	}
}

func RunVirtualMachine(ctx context.Context, hpv Hypervisor, vmi VMIProvider, vcpus uint, memory strongunits.B) error {
	id := "vm-" + xid.New().String()

	workingDir, err := host.EmphiricalVMCacheDir(ctx, id)
	if err != nil {
		return err
	}

	devices := []virtio.VirtioDevice{}

	err = host.DownloadAndExtractVMI(ctx, vmi.DiskImageURL(), workingDir)
	if err != nil {
		return err
	}

	diskImageToRun := filepath.Join(workingDir, vmi.DiskImageRawFileName())

	if _, err := os.Stat(diskImageToRun); os.IsNotExist(err) {
		return errors.Errorf("disk image does not exist: %s", diskImageToRun)
	}

	blkDev, err := virtio.VirtioBlkNew(diskImageToRun)
	if err != nil {
		return err
	}

	devices = append(devices, blkDev)

	// run boot provisioner
	bootProvisioners := vmi.BootProvisioners()
	for _, bootProvisioner := range bootProvisioners {
		if bootProvisionerVirtioDevices, err := bootProvisioner.VirtioDevices(ctx); err != nil {
			return err
		} else {
			devices = append(devices, bootProvisionerVirtioDevices...)
		}
	}

	runtimeProvisioners := vmi.RuntimeProvisioners()
	for _, runtimeProvisioner := range runtimeProvisioners {
		if runtimeProvisionerVirtioDevices, err := runtimeProvisioner.VirtioDevices(ctx); err != nil {
			return err
		} else {
			devices = append(devices, runtimeProvisionerVirtioDevices...)
		}
	}

	opts := NewVMOptions{
		Vcpus:   vcpus,
		Memory:  memory,
		Devices: devices,
	}

	vm, err := hpv.NewVirtualMachine(ctx, opts)
	if err != nil {
		return errors.Errorf("creating virtual machine: %w", err)
	}

	err = boot(ctx, vm, vmi)
	if err != nil {
		return errors.Errorf("booting virtual machine: %w", err)
	}

	errGroup, runCancel, err := run(ctx, hpv, vm, vmi)
	if err != nil {
		return errors.Errorf("running virtual machine: %w", err)
	}

	defer func() {
		runCancel()
		if err := errGroup.Wait(); err != nil {
			slog.DebugContext(ctx, "error running runtime provisioners", "error", err)
		}
	}()

	slog.InfoContext(ctx, "waiting for VM to stop")

	errCh := make(chan error, 1)
	go func() {
		if err := WaitForVMState(ctx, vm, VirtualMachineStateTypeStopped, nil); err != nil {
			errCh <- fmt.Errorf("virtualization error: %v", err)
		} else {
			slog.InfoContext(ctx, "VM is stopped")
			errCh <- nil
		}
	}()

	return <-errCh
}

func startVSockDevices(ctx context.Context, vm VirtualMachine) error {
	vsockDevs := virtio.VirtioDevicesOfType[*virtio.VirtioVsock](vm.Devices())
	for _, vsock := range vsockDevs {
		port := vsock.Port
		socketURL := vsock.SocketURL
		if socketURL == "" {
			// the timesync code adds a vsock device without an associated URL.
			// the ones that don't have urls are already set up on the main vsock
			continue
		}
		var listenStr string
		if vsock.Listen {
			listenStr = " (listening)"
		}
		slog.InfoContext(ctx, "Exposing vsock port", "port", port, "socketURL", socketURL, "listenStr", listenStr)
		closer, err := ExposeVsock(ctx, vm, port, socketURL, vsock.Listen)
		if err != nil {
			slog.WarnContext(ctx, "error exposing vsock port", "port", port, "error", err)
			continue
		}
		defer closer.Close()
	}
	return nil
}

func boot(ctx context.Context, vm VirtualMachine, vmi VMIProvider) error {
	bootCtx, bootCancel := context.WithCancel(ctx)
	errGroup, ctx := errgroup.WithContext(bootCtx)
	defer func() {
		// clean up the boot provisioners - this shouldn't throw an error because they prob are going to throw something
		bootCancel()
		if err := errGroup.Wait(); err != nil {
			slog.DebugContext(ctx, "error running boot provisioners", "error", err)
		}
	}()

	for _, provisioner := range vmi.BootProvisioners() {
		errGroup.Go(func() error {
			return provisioner.RunDuringBoot(bootCtx, vm)
		})
	}

	if err := vm.Start(ctx); err != nil {
		return errors.Errorf("starting virtual machine: %w", err)
	}

	if err := WaitForVMState(ctx, vm, VirtualMachineStateTypeRunning, time.After(5*time.Second)); err != nil {
		return errors.Errorf("waiting for virtual machine to start: %w", err)
	}

	slog.InfoContext(ctx, "virtual machine is running")

	return nil
}

func run(ctx context.Context, hpv Hypervisor, vm VirtualMachine, vmi VMIProvider) (*errgroup.Group, func(), error) {
	bootCtx, bootCancel := context.WithCancel(ctx)
	errGroup, ctx := errgroup.WithContext(bootCtx)

	if err := hpv.ListenNetworkBlockDevices(bootCtx, vm); err != nil {
		bootCancel()
		return nil, nil, errors.Errorf("listening network block devices: %w", err)
	}

	for _, provisioner := range vmi.RuntimeProvisioners() {
		errGroup.Go(func() error {
			return provisioner.RunDuringRuntime(bootCtx, vm)
		})
	}

	if err := startVSockDevices(bootCtx, vm); err != nil {
		bootCancel()
		return nil, nil, errors.Errorf("starting vsock devices: %w", err)
	}

	gpuDevs := virtio.VirtioDevicesOfType[*virtio.VirtioGPU](vm.Devices())
	for _, gpuDev := range gpuDevs {
		if gpuDev.UsesGUI {
			runtime.LockOSThread()
			err := vm.StartGraphicApplication(float64(gpuDev.Width), float64(gpuDev.Height))
			runtime.UnlockOSThread()
			if err != nil {
				bootCancel()
				return nil, nil, errors.Errorf("starting graphic application: %w", err)
			}
			break
		} else {
			slog.DebugContext(ctx, "not starting GUI")
		}
	}

	return errGroup, bootCancel, nil

}
