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
	"github.com/walteh/ec1/pkg/machines/bootloader"
	"github.com/walteh/ec1/pkg/machines/guest"
	"github.com/walteh/ec1/pkg/machines/host"
	"github.com/walteh/ec1/pkg/machines/virtio"
	"github.com/walteh/ec1/pkg/networks/gvnet/tapsock"
	"gitlab.com/tozd/go/errors"
	"golang.org/x/sync/errgroup"
)

func EmphericalBootLoaderConfigForGuest(ctx context.Context, provider VMIProvider, bootCacheDir string) (bootloader.Bootloader, error) {
	switch kt := provider.GuestKernelType(); kt {
	case guest.GuestKernelTypeLinux:
		if linuxVMIProvider, ok := provider.(LinuxVMIProvider); ok {
			return linuxVMIProvider.BootLoaderConfig(bootCacheDir), nil
		} else {
			return bootloader.NewEFIBootloader(filepath.Join(bootCacheDir, "efivars.fd"), true), nil
		}
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
		return errors.Errorf("downloading and extracting VMI: %w", err)
	}

	if customExtractor, ok := vmi.(CustomExtractorVMIProvider); ok {
		err = customExtractor.CustomExtraction(ctx, workingDir)
		if err != nil {
			return errors.Errorf("custom extraction: %w", err)
		}
	}

	if diskImageRawFileNameVMIProvider, ok := vmi.(DiskImageRawFileNameVMIProvider); ok {
		diskImageToRun := filepath.Join(workingDir, diskImageRawFileNameVMIProvider.DiskImageRawFileName())

		if _, err := os.Stat(diskImageToRun); os.IsNotExist(err) {
			return errors.Errorf("disk image does not exist: %s", diskImageToRun)
		}

		blkDev, err := virtio.VirtioBlkNew(diskImageToRun)
		if err != nil {
			return errors.Errorf("creating virtio block device: %w", err)
		}

		devices = append(devices, blkDev)
	}

	// add memory balloon device
	// memoryBalloonDev, err := virtio.VirtioBalloonNew()
	// if err != nil {
	// 	return errors.Errorf("creating memory balloon device: %w", err)
	// }
	// devices = append(devices, memoryBalloonDev)

	// run boot provisioner
	bootProvisioners := vmi.BootProvisioners()
	for _, bootProvisioner := range bootProvisioners {
		if bootProvisionerVirtioDevices, err := bootProvisioner.VirtioDevices(ctx); err != nil {
			return errors.Errorf("getting boot provisioner virtio devices: %w", err)
		} else {
			devices = append(devices, bootProvisionerVirtioDevices...)
		}
	}

	runtimeProvisioners := vmi.RuntimeProvisioners()

	socketPath := filepath.Join(workingDir, "gvproxy.sock")

	// os.Remove(socketPath)
	// os.Create(socketPath)

	gvnetProvisioner := NewGvproxyProvisioner(
		tapsock.NewVFKitVMSocket("unixgram://" + socketPath),
	)

	device, err := gvnetProvisioner.VirtioDevices(ctx)
	if err != nil {
		return errors.Errorf("getting gvproxy virtio devices: %w", err)
	}
	devices = append(devices, device...)

	// runtimeProvisioners = append(runtimeProvisioners, gvnetProvisioner)

	for _, runtimeProvisioner := range runtimeProvisioners {
		if runtimeProvisionerVirtioDevices, err := runtimeProvisioner.VirtioDevices(ctx); err != nil {
			return errors.Errorf("getting runtime provisioner virtio devices: %w", err)
		} else {
			devices = append(devices, runtimeProvisionerVirtioDevices...)
		}
	}

	bl, err := EmphericalBootLoaderConfigForGuest(ctx, vmi, workingDir)
	if err != nil {
		return errors.Errorf("getting boot loader config: %w", err)
	}

	opts := NewVMOptions{
		Vcpus:   vcpus,
		Memory:  memory,
		Devices: devices,
		Provisioners: []Provisioner{
			gvnetProvisioner,
		},
	}

	errgrp, ctx := errgroup.WithContext(ctx)

	errgrp.Go(func() error {
		return gvnetProvisioner.RunDuringRuntime(ctx, nil)
	})

	netDevs := virtio.VirtioDevicesOfType[*virtio.VirtioNet](devices)
	for _, netDev := range netDevs {
		if netDev.UnixSocketPath != "" {
			err := netDev.ConnectUnixPath(ctx)
			if err != nil {
				return errors.Errorf("connecting unix socket path: %w", err)
			}
		}
	}

	vm, err := hpv.NewVirtualMachine(ctx, id, opts, bl)
	if err != nil {
		return errors.Errorf("creating virtual machine: %w", err)
	}

	slog.WarnContext(ctx, "booting virtual machine")

	err = boot(ctx, vm, vmi)
	if err != nil {
		return errors.Errorf("booting virtual machine: %w", err)
	}

	// <-gvnetProvisioner.dev.ReadyChan

	slog.WarnContext(ctx, "running virtual machine")

	runErrGroup, runCancel, err := run(ctx, hpv, vm, runtimeProvisioners)
	if err != nil {
		return errors.Errorf("running virtual machine: %w", err)
	}

	defer func() {
		runCancel()
		if err := runErrGroup.Wait(); err != nil {
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

	go func() {
		if err := errgrp.Wait(); err != nil {
			slog.DebugContext(ctx, "error running gvproxy", "error", err)
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
		if vsock.Direction == virtio.VirtioVsockDirectionGuestConnectsAsClient {
			listenStr = " (listening)"
		}
		slog.InfoContext(ctx, "Exposing vsock port", "port", port, "socketURL", socketURL, "listenStr", listenStr)
		closer, err := ExposeVsock(ctx, vm, vsock)
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
		slog.InfoContext(ctx, "running boot provisioner", "provisioner", provisioner)
		errGroup.Go(func() error {
			return provisioner.RunDuringBoot(bootCtx, vm)
		})
	}

	if err := vm.Start(ctx); err != nil {
		return errors.Errorf("starting virtual machine: %w", err)
	}

	if err := WaitForVMState(ctx, vm, VirtualMachineStateTypeRunning, time.After(30*time.Second)); err != nil {
		return errors.Errorf("waiting for virtual machine to start: %w", err)
	}

	slog.InfoContext(ctx, "virtual machine is running")

	return nil
}

func run(ctx context.Context, hpv Hypervisor, vm VirtualMachine, provisioners []RuntimeProvisioner) (*errgroup.Group, func(), error) {
	runCtx, bootCancel := context.WithCancel(ctx)
	errGroup, ctx := errgroup.WithContext(runCtx)

	if err := vm.ListenNetworkBlockDevices(runCtx); err != nil {
		bootCancel()
		return nil, nil, errors.Errorf("listening network block devices: %w", err)
	}

	slog.WarnContext(ctx, "running runtime provisioners")
	for _, provisioner := range provisioners {
		errGroup.Go(func() error {
			slog.DebugContext(ctx, "running runtime provisioner", "provisioner", provisioner)
			err := provisioner.RunDuringRuntime(runCtx, vm)
			if err != nil {
				slog.DebugContext(ctx, "error running runtime provisioner", "error", err)
				return errors.Errorf("running runtime provisioner: %w", err)
			}
			return nil
		})
	}

	if err := startVSockDevices(runCtx, vm); err != nil {
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

	for _, provisioner := range provisioners {
		<-provisioner.ReadyChan()
	}

	return errGroup, bootCancel, nil

}
