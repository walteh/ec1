package vmm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/containers/common/pkg/strongunits"
	"github.com/rs/xid"
	"gitlab.com/tozd/go/errors"

	"github.com/walteh/ec1/gen/harpoon/harpoon_initramfs_amd64"
	"github.com/walteh/ec1/gen/harpoon/harpoon_initramfs_arm64"
	"github.com/walteh/ec1/gen/harpoon/harpoon_vmlinux_amd64"
	"github.com/walteh/ec1/gen/harpoon/harpoon_vmlinux_arm64"
	"github.com/walteh/ec1/pkg/binembed"
	"github.com/walteh/ec1/pkg/bootloader"
	"github.com/walteh/ec1/pkg/ec1init"
	"github.com/walteh/ec1/pkg/ext/osx"
	"github.com/walteh/ec1/pkg/host"
	"github.com/walteh/ec1/pkg/oci"
	"github.com/walteh/ec1/pkg/units"
	"github.com/walteh/ec1/pkg/virtio"
)

type ConatinerImageConfig struct {
	ImageRef string
	// Cmdline  []string
	Platform units.Platform
	Memory   strongunits.B
	VCPUs    uint64
}

func NewContainerizedVirtualMachine[VM VirtualMachine](
	ctx context.Context,
	hpv Hypervisor[VM],
	cache oci.ImageFetchConverter,
	imageConfig ConatinerImageConfig,
	devices ...virtio.VirtioDevice) (*RunningVM[VM], error) {
	id := "vm-" + xid.New().String()
	errgrp, ctx := errgroup.WithContext(ctx)

	startTime := time.Now()

	workingDir, err := host.EmphiricalVMCacheDir(ctx, id)
	if err != nil {
		return nil, err
	}

	err = os.MkdirAll(workingDir, 0755)
	if err != nil {
		return nil, errors.Errorf("creating working directory: %w", err)
	}

	ec1Devices, err := PrepareContainerVirtioDevices(ctx, workingDir, imageConfig, cache, errgrp)
	if err != nil {
		return nil, errors.Errorf("creating ec1 block device: %w", err)
	}
	devices = append(devices, ec1Devices...)

	var bootloader bootloader.Bootloader

	switch imageConfig.Platform.OS() {
	case "linux":
		bl, bldevs, err := PrepareHarpoonLinuxBootloader(ctx, workingDir, imageConfig, errgrp)
		if err != nil {
			return nil, errors.Errorf("getting boot loader config: %w", err)
		}
		bootloader = bl
		devices = append(devices, bldevs...)
	default:
		return nil, errors.Errorf("unsupported OS: %s", imageConfig.Platform.OS())
	}

	devices = append(devices, &virtio.VirtioSerial{
		LogFile: filepath.Join(workingDir, "console.log"),
	})

	netdev, hostIPPort, err := PrepareVirtualNetwork(ctx, errgrp)
	if err != nil {
		return nil, errors.Errorf("creating net device: %w", err)
	}
	devices = append(devices, netdev)

	opts := NewVMOptions{
		Vcpus:   imageConfig.VCPUs,
		Memory:  imageConfig.Memory,
		Devices: devices,
	}

	vm, err := hpv.NewVirtualMachine(ctx, id, opts, bootloader)
	if err != nil {
		return nil, errors.Errorf("creating virtual machine: %w", err)
	}

	err = bootContainerVM(ctx, vm)
	if err != nil {
		return nil, errors.Errorf("booting virtual machine: %w", err)
	}

	runErrGroup, runCancel, err := runContainerVM(ctx, hpv, vm)
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

	return NewRunningVM(ctx, vm, hostIPPort, startTime, errCh), nil

}

func PrepareContainerVirtioDevices(ctx context.Context, wrkdir string, imageConfig ConatinerImageConfig, cache oci.ImageFetchConverter, wg *errgroup.Group) ([]virtio.VirtioDevice, error) {

	ec1DataPath := filepath.Join(wrkdir, "harpoon-runtime-fs-device")

	devices := []virtio.VirtioDevice{}

	for _, path := range []string{ec1DataPath} {
		err := os.MkdirAll(path, 0755)
		if err != nil {
			return nil, errors.Errorf("creating block device directory: %w", err)
		}
	}

	diskPath, err := oci.FetchAndConvertImage(ctx, cache, imageConfig.ImageRef, imageConfig.Platform)
	if err != nil {
		return nil, errors.Errorf("container to virtio device: %w", err)
	}

	blkDev, err := virtio.VirtioFsNew(diskPath.RootfsPath, ec1init.RootfsVirtioTag)
	if err != nil {
		return nil, errors.Errorf("creating block device: %w", err)
	}

	devices = append(devices, blkDev)

	// save all the files to a temp file
	metadataBytes, err := json.Marshal(diskPath.Metadata)
	if err != nil {
		return nil, errors.Errorf("marshalling metadata: %w", err)
	}

	// cmdlineBytes, err := json.Marshal(imageConfig.Cmdline)
	// if err != nil {
	// 	return nil, errors.Errorf("marshalling cmdline: %w", err)
	// }

	files := map[string][]byte{
		ec1init.ContainerManifestFile: metadataBytes,
		// ec1init.ContainerCmdlineFile:  cmdlineBytes,
	}

	for name, file := range files {
		filePath := filepath.Join(ec1DataPath, name)
		err = osx.WriteFileFromReaderAsync(ctx, filePath, bytes.NewReader(file), 0644, wg)
		if err != nil {
			return nil, errors.Errorf("writing file to block device: %w", err)
		}
	}

	ec1Dev, err := virtio.VirtioFsNew(ec1DataPath, ec1init.Ec1VirtioTag)
	if err != nil {
		return nil, errors.Errorf("creating block device: %w", err)
	}

	devices = append(devices, ec1Dev)

	return devices, nil
}

func PrepareHarpoonLinuxBootloader(ctx context.Context, wrkdir string, imageConfig ConatinerImageConfig, wg *errgroup.Group) (bootloader.Bootloader, []virtio.VirtioDevice, error) {
	targetVmLinuxPath := filepath.Join(wrkdir, "vmlinux")
	targetInitramfsPath := filepath.Join(wrkdir, "initramfs.cpio.gz")

	extraArgs := ""
	extraInitArgs := ""

	devices := []virtio.VirtioDevice{}

	var kernelXz, initramfsGz io.Reader
	var err error

	startTime := time.Now()

	if imageConfig.Platform.Arch() == "arm64" {
		kernelXz, err = binembed.GetDecompressed(harpoon_vmlinux_arm64.BinaryXZChecksum)
		if err != nil {
			return nil, nil, errors.Errorf("getting kernel: %w", err)
		}
		initramfsGz, err = binembed.GetDecompressed(harpoon_initramfs_arm64.BinaryXZChecksum)
		if err != nil {
			return nil, nil, errors.Errorf("getting initramfs: %w", err)
		}
	} else {
		kernelXz, err = binembed.GetDecompressed(harpoon_vmlinux_amd64.BinaryXZChecksum)
		if err != nil {
			return nil, nil, errors.Errorf("getting kernel: %w", err)
		}
		initramfsGz, err = binembed.GetDecompressed(harpoon_initramfs_amd64.BinaryXZChecksum)
		if err != nil {
			return nil, nil, errors.Errorf("getting initramfs: %w", err)
		}
	}

	files := map[string]io.Reader{
		targetVmLinuxPath:   kernelXz,
		targetInitramfsPath: initramfsGz,
	}

	for path, reader := range files {
		err = osx.WriteFileFromReaderAsync(ctx, path, reader, 0644, wg)
		if err != nil {
			return nil, nil, errors.Errorf("writing files: %w", err)
		}
	}

	// cmdLine := linuxVMIProvider.KernelArgs() + " console=hvc0 cloud-init=disabled network-config=disabled" + extraArgs
	cmdLine := strings.TrimSpace(" console=hvc0 " + extraArgs + " -- " + extraInitArgs)

	slog.InfoContext(ctx, "linux boot loader ready", "duration", time.Since(startTime))

	return &bootloader.LinuxBootloader{
		InitrdPath:    targetInitramfsPath,
		VmlinuzPath:   targetVmLinuxPath,
		KernelCmdLine: cmdLine,
	}, devices, nil
}

func bootContainerVM[VM VirtualMachine](ctx context.Context, vm VM) error {
	bootCtx, bootCancel := context.WithCancel(ctx)
	errGroup, ctx := errgroup.WithContext(bootCtx)
	defer func() {
		// clean up the boot provisioners - this shouldn't throw an error because they prob are going to throw something
		bootCancel()
		if err := errGroup.Wait(); err != nil {
			slog.DebugContext(ctx, "error running boot provisioners", "error", err)
		}
	}()

	if err := vm.Start(ctx); err != nil {
		return errors.Errorf("starting virtual machine: %w", err)
	}

	if err := WaitForVMState(ctx, vm, VirtualMachineStateTypeRunning, time.After(30*time.Second)); err != nil {
		return errors.Errorf("waiting for virtual machine to start: %w", err)
	}

	slog.InfoContext(ctx, "virtual machine is running")

	return nil
}

func runContainerVM[VM VirtualMachine](ctx context.Context, hpv Hypervisor[VM], vm VM) (*errgroup.Group, func(), error) {
	runCtx, bootCancel := context.WithCancel(ctx)
	errGroup, ctx := errgroup.WithContext(runCtx)

	if err := vm.ListenNetworkBlockDevices(runCtx); err != nil {
		bootCancel()
		return nil, nil, errors.Errorf("listening network block devices: %w", err)
	}

	if err := StartVSockDevices(runCtx, vm); err != nil {
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
