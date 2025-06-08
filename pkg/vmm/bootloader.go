package vmm

import (
	"context"
	"io"
	"log/slog"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"

	"gitlab.com/tozd/go/errors"

	"github.com/walteh/ec1/gen/harpoon/harpoon_initramfs_amd64"
	"github.com/walteh/ec1/gen/harpoon/harpoon_initramfs_arm64"
	"github.com/walteh/ec1/gen/harpoon/harpoon_vmlinux_amd64"
	"github.com/walteh/ec1/gen/harpoon/harpoon_vmlinux_arm64"
	"github.com/walteh/ec1/pkg/binembed"
	"github.com/walteh/ec1/pkg/ext/osx"
	"github.com/walteh/ec1/pkg/virtio"
)

// LinuxBootloader determines which kernel/initrd/kernel args to use when starting
// the virtual machine.
type LinuxBootloader struct {
	VmlinuzPath   string `json:"vmlinuzPath"`
	KernelCmdLine string `json:"kernelCmdLine"`
	InitrdPath    string `json:"initrdPath"`
}

// EFIBootloader allows to set a few options related to EFI variable storage
type EFIBootloader struct {
	EFIVariableStorePath string `json:"efiVariableStorePath"`
	// TODO: virtualization framework allow both create and overwrite
	CreateVariableStore bool `json:"createVariableStore"`
}

// MacOSBootloader provides necessary objects for booting macOS guests
type MacOSBootloader struct {
	MachineIdentifierPath string `json:"machineIdentifierPath"`
	HardwareModelPath     string `json:"hardwareModelPath"`
	AuxImagePath          string `json:"auxImagePath"`
}

// NewEFIBootloader creates a new bootloader to start a VM using EFI
// efiVariableStorePath is the path to a file for EFI storage
// create is a boolean indicating if the file for the store should be created or not
func NewEFIBootloader(efiVariableStorePath string, createVariableStore bool) *EFIBootloader {
	return &EFIBootloader{
		EFIVariableStorePath: efiVariableStorePath,
		CreateVariableStore:  createVariableStore,
	}
}

type Bootloader interface {
	isBootloader()
}

func (bootloader *LinuxBootloader) isBootloader() {}
func (bootloader *EFIBootloader) isBootloader()   {}
func (bootloader *MacOSBootloader) isBootloader() {}

func PrepareHarpoonLinuxBootloader(ctx context.Context, wrkdir string, imageConfig ConatinerImageConfig, wg *errgroup.Group) (Bootloader, []virtio.VirtioDevice, error) {
	targetVmLinuxPath := filepath.Join(wrkdir, "vmlinux")
	targetInitramfsPath := filepath.Join(wrkdir, "initramfs.cpio.gz")

	extraArgs := ""
	extraInitArgs := ""

	devices := []virtio.VirtioDevice{}

	groupErrs := errgroup.Group{}

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
		err = osx.WriteFileFromReaderAsync(ctx, path, reader, 0644, &groupErrs)
		if err != nil {
			return nil, nil, errors.Errorf("writing files: %w", err)
		}
		// os.Chown(path, 1000, 1000)
	}

	// cmdLine := linuxVMIProvider.KernelArgs() + " console=hvc0 cloud-init=disabled network-config=disabled" + extraArgs
	cmdLine := strings.TrimSpace(" console=hvc0 " + extraArgs + " -- " + extraInitArgs)

	err = groupErrs.Wait()
	if err != nil {
		return nil, nil, errors.Errorf("error waiting for errgroup: %w", err)
	}

	slog.InfoContext(ctx, "linux boot loader ready", "duration", time.Since(startTime))

	return &LinuxBootloader{
		InitrdPath:    targetInitramfsPath,
		VmlinuzPath:   targetVmLinuxPath,
		KernelCmdLine: cmdLine,
	}, devices, nil
}
