package vmm

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/containers/common/pkg/strongunits"
	"github.com/rs/xid"
	"gitlab.com/tozd/go/errors"

	"github.com/walteh/ec1/pkg/bootloader"
	"github.com/walteh/ec1/pkg/guest"
	"github.com/walteh/ec1/pkg/gvnet"
	"github.com/walteh/ec1/pkg/host"
	"github.com/walteh/ec1/pkg/port"
	"github.com/walteh/ec1/pkg/unzbootgo"
	"github.com/walteh/ec1/pkg/virtio"
)

func EmphericalBootLoaderConfigForGuest(ctx context.Context, provider VMIProvider, bootCacheDir string) (bootloader.Bootloader, []virtio.VirtioDevice, error) {
	var err error
	var devices []virtio.VirtioDevice
	switch kt := provider.GuestKernelType(); kt {
	case guest.GuestKernelTypeLinux:
		extraArgs := ""
		if linuxVMIProvider, ok := provider.(LinuxVMIProvider); ok {
			initramfsFileName := linuxVMIProvider.InitramfsPath()
			initramfsPath := filepath.Join(bootCacheDir, initramfsFileName)
			if initramfsFileName != "" {
				initramfsPath, err = bootloader.PrepareInitramfsCpio(ctx, initramfsPath)
				if err != nil {
					return nil, nil, errors.Errorf("preparing initramfs cpio: %w", err)
				}
			}
			rootfsFileName := linuxVMIProvider.RootfsPath()
			// rootfsFileName = ""
			rootfsPath := filepath.Join(bootCacheDir, rootfsFileName)
			if rootfsFileName != "" {
				if initramfsFileName == "" {
					rootfsPath, err = bootloader.PrepareRootFS(ctx, rootfsPath)
					if err != nil {
						return nil, nil, errors.Errorf("preparing rootfs: %w", err)
					}
				}
				blkDev, err := virtio.VirtioBlkNew(rootfsPath)
				if err != nil {
					return nil, nil, errors.Errorf("creating virtio block device: %w", err)
				}
				devices = append(devices, blkDev)
				extraArgs = "  root=/dev/vda1 rw rootfstype=ext4 rootdelay=5"
			}

			kernelFileName := linuxVMIProvider.KernelPath()
			if kernelFileName == "" {
				return nil, nil, errors.New("kernel file name is empty")
			}
			kernelPath := filepath.Join(bootCacheDir, kernelFileName)

			err = unzbootgo.ExtractKernel(kernelPath, kernelPath+".ec1")
			if err != nil {
				// Not an EFI application or extraction failed, return the original
				return nil, nil, errors.Errorf("extracting kernel: %w", err)
			}

			slog.InfoContext(ctx, "kernel path", "path", kernelPath)

			return &bootloader.LinuxBootloader{
				InitrdPath:    initramfsPath,
				VmlinuzPath:   kernelPath,
				KernelCmdLine: linuxVMIProvider.KernelArgs() + " console=hvc0" + extraArgs,
			}, devices, nil
		}
		// return bootloader.NewEFIBootloader(filepath.Join(bootCacheDir, "efivars.fd"), true), nil
		return nil, nil, errors.New("unsupported guest kernel type: linux")

	case guest.GuestKernelTypeDarwin:
		if mos, ok := provider.(MacOSVMIProvider); ok {
			return mos.BootLoaderConfig(), nil, nil
		} else {
			return nil, nil, errors.New("guest kernel type is darwin but provider does not support macOS")
		}
	default:
		return nil, nil, errors.Errorf("unsupported guest kernel type: %s", kt)
	}
}

// obviously this is not secure, we need something better long term
// for now its fine because im not even sure it will be used
// if this key thing is depended upon we need to move it to a more secure location
func addSSHKeyToVM(ctx context.Context, workingDir string) error {
	sshKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return errors.Errorf("creating ssh key: %w", err)
	}

	m, err := x509.MarshalPKCS8PrivateKey(sshKey)
	if err != nil {
		return errors.Errorf("marshalling ssh key: %w", err)
	}

	sshKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: m})

	sshKeyFile := filepath.Join(workingDir, "id_ecdsa")
	err = os.WriteFile(sshKeyFile, sshKeyPEM, 0600)
	if err != nil {
		return errors.Errorf("writing ssh key: %w", err)
	}

	return nil
}

func RunVirtualMachine[VM VirtualMachine](ctx context.Context, hpv Hypervisor[VM], vmi VMIProvider, vcpus uint, memory strongunits.B) (*RunningVM[VM], error) {
	id := "vm-" + xid.New().String()

	startTime := time.Now()

	workingDir, err := host.EmphiricalVMCacheDir(ctx, id)
	if err != nil {
		return nil, err
	}

	err = os.MkdirAll(workingDir, 0755)
	if err != nil {
		return nil, errors.Errorf("creating working directory: %w", err)
	}

	devices := []virtio.VirtioDevice{}
	provisioners := []Provisioner{}

	err = addSSHKeyToVM(ctx, workingDir)
	if err != nil {
		return nil, errors.Errorf("adding ssh key to vm: %w", err)
	}

	// create an ssh private key for this vm

	if diskImageURLVMIProvider, ok := vmi.(DownloadableVMIProvider); ok {
		files, err := host.DownloadAndExtractVMI(ctx, diskImageURLVMIProvider.Downloads())
		if err != nil {
			return nil, errors.Errorf("downloading and extracting VMI: %w", err)
		}

		// fdupforcache := fdup()

		extrfles, err := diskImageURLVMIProvider.ExtractDownloads(ctx, files)
		if err != nil {
			return nil, errors.Errorf("extracting downloads: %w", err)
		}

		// err = host.SaveDownloadedFilesToCache(ctx, diskImageURLVMIProvider.Downloads(), fdupforcache)
		// if err != nil {
		// 	return nil, errors.Errorf("saving downloaded files to cache: %w", err)
		// }

		for name, file := range extrfles {
			filePath := filepath.Join(workingDir, name)
			out, err := os.Create(filePath)
			if err != nil {
				return nil, errors.Errorf("creating file: %w", err)
			}
			defer out.Close()
			_, err = io.Copy(out, file)
			if err != nil {
				return nil, errors.Errorf("writing file: %w", err)
			}
		}

		// if customExtractor, ok := vmi.(CustomExtractorVMIProvider); ok {
		// 	err = customExtractor.CustomExtraction(ctx, workingDir)
		// 	if err != nil {
		// 		return nil, errors.Errorf("custom extraction: %w", err)
		// 	}
		// }

		// if diskImageRawFileNameVMIProvider, ok := vmi.(DiskImageRawFileNameVMIProvider); ok {
		// 	diskImageToRun := filepath.Join(workingDir, diskImageRawFileNameVMIProvider.DiskImageRawFileName())

		// 	if _, err := os.Stat(diskImageToRun); os.IsNotExist(err) {
		// 		return nil, errors.Errorf("disk image does not exist: %s", diskImageToRun)
		// 	}

		// 	blkDev, err := virtio.VirtioBlkNew(diskImageToRun)
		// 	if err != nil {
		// 		return nil, errors.Errorf("creating virtio block device: %w", err)
		// 	}

		// 	devices = append(devices, blkDev)
		// }

	}

	devices = append(devices, &virtio.VirtioSerial{
		LogFile: filepath.Join(workingDir, "console.log"),
	})

	// run boot provisioner
	bootProvisioners := vmi.BootProvisioners()
	for _, bootProvisioner := range bootProvisioners {
		if bootProvisionerVirtioDevices, err := bootProvisioner.VirtioDevices(ctx); err != nil {
			return nil, errors.Errorf("getting boot provisioner virtio devices: %w", err)
		} else {
			devices = append(devices, bootProvisionerVirtioDevices...)
		}
		provisioners = append(provisioners, bootProvisioner)
	}

	runtimeProvisioners := vmi.RuntimeProvisioners()

	errgrp, ctx := errgroup.WithContext(ctx)

	netdev, hostIPPort, err := NewNetDevice(ctx, errgrp)
	if err != nil {
		return nil, errors.Errorf("creating net device: %w", err)
	}
	devices = append(devices, netdev)

	for _, runtimeProvisioner := range runtimeProvisioners {
		if runtimeProvisionerVirtioDevices, err := runtimeProvisioner.VirtioDevices(ctx); err != nil {
			return nil, errors.Errorf("getting runtime provisioner virtio devices: %w", err)
		} else {
			devices = append(devices, runtimeProvisionerVirtioDevices...)
		}
		provisioners = append(provisioners, runtimeProvisioner)
	}

	bl, bldev, err := EmphericalBootLoaderConfigForGuest(ctx, vmi, workingDir)
	if err != nil {
		return nil, errors.Errorf("getting boot loader config: %w", err)
	}
	devices = append(devices, bldev...)

	opts := NewVMOptions{
		Vcpus:        vcpus,
		Memory:       memory,
		Devices:      devices,
		Provisioners: provisioners,
	}

	slog.InfoContext(ctx, "creating virtual machine")

	vm, err := hpv.NewVirtualMachine(ctx, id, opts, bl)
	if err != nil {
		return nil, errors.Errorf("creating virtual machine: %w", err)
	}

	slog.WarnContext(ctx, "booting virtual machine")

	err = boot(ctx, vm, vmi)
	if err != nil {
		return nil, errors.Errorf("booting virtual machine: %w", err)
	}

	slog.WarnContext(ctx, "running virtual machine")

	runErrGroup, runCancel, err := run(ctx, hpv, vm, runtimeProvisioners)
	if err != nil {
		return nil, errors.Errorf("running virtual machine: %w", err)
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
		if err := errgrp.Wait(); err != nil && err != context.Canceled {
			slog.ErrorContext(ctx, "error running gvproxy", "error", err)
		}
	}()

	runtimeInfo := &RunningVM[VM]{
		portOnHostIP: hostIPPort,
		vm:           vm,
		wait:         errCh,
		start:        startTime,
	}

	return runtimeInfo, nil
}

func NewNetDevice(ctx context.Context, groupErrs *errgroup.Group) (*virtio.VirtioNet, uint16, error) {
	port, err := port.ReservePort(ctx)
	if err != nil {
		return nil, 0, errors.Errorf("reserving port: %w", err)
	}
	cfg := &gvnet.GvproxyConfig{
		VMHostPort:         fmt.Sprintf("tcp://127.0.0.1:%d", port),
		EnableDebug:        false,
		EnableStdioSocket:  false,
		EnableNoConnectAPI: true,
	}

	dev, waiter, err := gvnet.NewProxy(ctx, cfg)
	if err != nil {
		return nil, 0, errors.Errorf("creating gvproxy: %w", err)
	}

	groupErrs.Go(func() error {
		slog.InfoContext(ctx, "waiting on error from gvproxy")
		return waiter(ctx)
	})

	return dev, port, nil

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
		_, _, closer, err := ExposeVsock(ctx, vm, vsock.Port, vsock.Direction)
		if err != nil {
			slog.WarnContext(ctx, "error exposing vsock port", "port", port, "error", err)
			continue
		}
		defer closer()
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

func run[VM VirtualMachine](ctx context.Context, hpv Hypervisor[VM], vm VM, provisioners []RuntimeProvisioner) (*errgroup.Group, func(), error) {
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
