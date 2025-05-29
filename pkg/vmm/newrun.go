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
	"github.com/containers/image/v5/types"
	"github.com/diskfs/go-diskfs/backend/file"
	"github.com/diskfs/go-diskfs/filesystem/fat32"
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
	"github.com/walteh/ec1/pkg/virtio"
)

type ConatinerImageConfig struct {
	ImageRef string
	Cmdline  []string
	Arch     string
	OS       string
	Memory   strongunits.B
	VCPUs    uint
}

func NewContainerizedVirtualMachine[VM VirtualMachine](
	ctx context.Context,
	hpv Hypervisor[VM],
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

	ec1Devices, err := PrepareContainerVirtioDevices(ctx, workingDir, imageConfig, errgrp)
	if err != nil {
		return nil, errors.Errorf("creating ec1 block device: %w", err)
	}
	devices = append(devices, ec1Devices...)

	var bootloader bootloader.Bootloader

	switch imageConfig.OS {
	case "linux":
		bl, bldevs, err := PrepareLinuxBootloader(ctx, workingDir, imageConfig, errgrp)
		if err != nil {
			return nil, errors.Errorf("getting boot loader config: %w", err)
		}
		bootloader = bl
		devices = append(devices, bldevs...)
	default:
		return nil, errors.Errorf("unsupported OS: %s", imageConfig.OS)
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

	slog.InfoContext(ctx, "creating virtual machine")

	vm, err := hpv.NewVirtualMachine(ctx, id, opts, bootloader)
	if err != nil {
		return nil, errors.Errorf("creating virtual machine: %w", err)
	}

	slog.WarnContext(ctx, "booting virtual machine")

	err = bootContainerVM(ctx, vm)
	if err != nil {
		return nil, errors.Errorf("booting virtual machine: %w", err)
	}

	slog.WarnContext(ctx, "running virtual machine")

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

func PrepareContainerVirtioDevices(ctx context.Context, wrkdir string, imageConfig ConatinerImageConfig, wg *errgroup.Group) ([]virtio.VirtioDevice, error) {

	// rootfsPath := filepath.Join(wrkdir, "harpoon-rootfs-fs-device")
	ec1DataPath := filepath.Join(wrkdir, "harpoon-runtime-fs-device")
	// tempPath := filepath.Join(wrkdir, "harpoon-writable-overlay-blk-device.img")

	devices := []virtio.VirtioDevice{}

	for _, path := range []string{ec1DataPath} {
		err := os.MkdirAll(path, 0755)
		if err != nil {
			return nil, errors.Errorf("creating block device directory: %w", err)
		}
	}

	diskPath, metadata, err := oci.ContainerToVirtioDeviceCached(ctx, oci.ContainerToVirtioOptions{
		ImageRef: imageConfig.ImageRef,
		Platform: &types.SystemContext{
			OSChoice:           imageConfig.OS,
			ArchitectureChoice: imageConfig.Arch,
		},
		// OutputDir: rootfsPath,
	})
	if err != nil {
		return nil, errors.Errorf("container to virtio device: %w", err)
	}

	// wo, err := emptyWritableOverlayFS(ctx, tempPath, humanize.MiByte*128)
	// if err != nil {
	// 	return nil, errors.Errorf("creating writable overlay block device: %w", err)
	// }
	// wo.DeviceIdentifier = ec1init.TempVirtioTag
	// devices = append(devices, wo)

	blkDev, err := virtio.VirtioFsNew(diskPath.ReadonlyFSPath, ec1init.RootfsVirtioTag)
	if err != nil {
		return nil, errors.Errorf("creating block device: %w", err)
	}

	devices = append(devices, blkDev)

	// blkDev, err := virtio.VirtioBlkNew(diskPath.ReadonlyExt4Path)
	// if err != nil {
	// 	return nil, errors.Errorf("creating block device: %w", err)
	// }
	// blkDev.DeviceIdentifier = ec1init.RootfsVirtioTag
	// blkDev.ReadOnly = true
	// devices = append(devices, blkDev)

	// save all the files to a temp file
	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		return nil, errors.Errorf("marshalling metadata: %w", err)
	}

	cmdlineBytes, err := json.Marshal(imageConfig.Cmdline)
	if err != nil {
		return nil, errors.Errorf("marshalling cmdline: %w", err)
	}

	files := map[string][]byte{
		ec1init.ContainerManifestFile: metadataBytes,
		ec1init.ContainerCmdlineFile:  cmdlineBytes,
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

func emptyWritableOverlayFS(ctx context.Context, filename string, size int64) (blkDev *virtio.VirtioBlk, err error) {
	tempFile, err := os.OpenFile(filename, os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		return nil, errors.Errorf("creating temp file: %w", err)
	}
	defer tempFile.Close()

	err = tempFile.Truncate(size)
	if err != nil {
		return nil, errors.Errorf("truncating temp file: %w", err)
	}

	// make file read write by all
	// err = tempFile.Chmod(os.ModePerm)
	// if err != nil {
	// 	return nil, errors.Errorf("chmoding temp file: %w", err)
	// }

	// err = tempFile.Truncate(size)
	// if err != nil {
	// 	return nil, errors.Errorf("truncating temp file: %w", err)
	// }

	// pr, pw := io.Pipe()
	// go func() {
	// 	defer pw.Close()
	// 	err = (&archives.Tar{}).Archive(ctx, pw, []archives.FileInfo{})
	// 	if err != nil {
	// 		slog.ErrorContext(ctx, "archiving temp file", "error", err)
	// 	}
	// }()

	// // MaximumDiskSize tells tar2ext4 how big the raw device is
	// err = tar2ext4.ConvertTarToExt4(
	// 	pr,
	// 	tempFile,
	// 	tar2ext4.MaximumDiskSize(size),
	// 	tar2ext4.InlineData,
	// )
	// if err != nil {
	// 	return nil, err
	// }
	// tempFile.Sync()

	blkDev, err = virtio.VirtioBlkNew(filename)
	if err != nil {
		return nil, errors.Errorf("creating block device: %w", err)
	}

	blkDev.ReadOnly = false

	return blkDev, nil
}

func emptyWritableOverlayFS3(ctx context.Context, filename string, size int64) (blkDev *virtio.VirtioBlk, err error) {

	be, err := file.CreateFromPath(filename, size)
	if err != nil {
		return nil, errors.Errorf("creating temp file: %w", err)
	}

	fat32d, err := fat32.Create(be, size, 0, 512, "harpoon-overlay")
	if err != nil {
		return nil, errors.Errorf("creating fat32 filesystem: %w", err)
	}

	if err := fat32d.Close(); err != nil {
		return nil, errors.Errorf("closing fat32 filesystem: %w", err)
	}
	// ext4d, err := ext4.Create(be, size, 0, 512, &ext4.Params{
	// 	UUID:               &uuid,
	// 	VolumeName:         "ec1-overlay",
	// 	Checksum:           true,
	// 	SparseSuperVersion: 1,
	// 	BlocksPerGroup:     1024,
	// 	InodeRatio:         1024,

	// 	// JournalDevice: "journal",

	// 	Features: []ext4.FeatureOpt{
	// 		ext4.WithFeatureReadOnly(false),
	// 		ext4.WithFeatureBTreeDirectory(true),
	// 		ext4.WithFeatureHasJournal(true),
	// 		ext4.WithFeatureCompression(true),
	// 		ext4.WithFeatureSnapshot(true),
	// 		ext4.WithFeatureGDTChecksum(true),
	// 		ext4.WithFeatureFS64Bit(true),
	// 	},
	// // })
	// if err != nil {
	// 	return nil, errors.Errorf("creating ext4 filesystem: %w", err)
	// }

	// ext4d, err := fat32.Create(be, size, 0, 512,  )
	// if err != nil {
	// 	return nil, errors.Errorf("creating ext4 filesystem: %w", err)
	// // }
	// if err := ext4d.Close(); err != nil {
	// 	return nil, errors.Errorf("closing ext4 filesystem: %w", err)
	// }

	blkDev, err = virtio.VirtioBlkNew(filename)
	if err != nil {
		return nil, errors.Errorf("creating block device: %w", err)
	}

	blkDev.ReadOnly = false

	return blkDev, nil
}

func init() {
	// Pre-load the binaries into memory asynchronously for faster boot times
	// This starts decompression in background goroutines and returns immediately
	// binembed.PreloadAsync(harpoon_vmlinux_arm64.BinaryXZChecksum)
	// binembed.PreloadAsync(harpoon_initramfs_arm64.BinaryXZChecksum)
	// binembed.PreloadAsync(harpoon_vmlinux_amd64.BinaryXZChecksum)
	// binembed.PreloadAsync(harpoon_initramfs_amd64.BinaryXZChecksum)

	// Alternative: Use PreloadAllSync() if you need to ensure all binaries
	// are ready before proceeding. This will decompress all registered binaries
	// concurrently and wait for completion:
	//
	// if err := binembed.PreloadAllSync(); err != nil {
	//     slog.Error("failed to preload binaries", "error", err)
	// }
}

func PrepareLinuxBootloader(ctx context.Context, wrkdir string, imageConfig ConatinerImageConfig, wg *errgroup.Group) (bootloader.Bootloader, []virtio.VirtioDevice, error) {
	targetVmLinuxPath := filepath.Join(wrkdir, "vmlinux")
	targetInitramfsPath := filepath.Join(wrkdir, "initramfs.cpio.gz")

	extraArgs := ""
	extraInitArgs := ""

	devices := []virtio.VirtioDevice{}

	var kernelXz, initramfsGz io.Reader
	var err error

	startTime := time.Now()

	if imageConfig.Arch == "arm64" {
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

	return errGroup, bootCancel, nil

}
