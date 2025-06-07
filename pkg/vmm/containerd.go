package vmm

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/containerd/containerd/api/types"
	"github.com/containerd/containerd/v2/pkg/oci"
	"github.com/containers/common/pkg/strongunits"
	"github.com/lmittmann/tint"
	"github.com/opencontainers/runtime-spec/specs-go"
	"gitlab.com/tozd/go/errors"

	"github.com/walteh/ec1/pkg/ec1init"
	"github.com/walteh/ec1/pkg/ext/osx"
	"github.com/walteh/ec1/pkg/host"
	"github.com/walteh/ec1/pkg/units"
	"github.com/walteh/ec1/pkg/virtio"
)

type ContainerizedVMConfig struct {
	ID           string
	ExecID       string
	RootfsMounts []*types.Mount
	RootfsPath   string
	StderrFD     int
	StdoutFD     int
	StdinFD      int
	DNSPath      string
	StdinPath    string
	StdoutPath   string
	StderrPath   string
	Spec         *oci.Spec
	Memory       strongunits.B
	VCPUs        uint64
	Platform     units.Platform
}

// NewContainerizedVirtualMachineFromRootfs creates a VM using an already-prepared rootfs directory
// This is used by container runtimes like containerd that have already prepared the rootfs
func NewContainerizedVirtualMachineFromRootfs[VM VirtualMachine](
	ctx context.Context,
	hpv Hypervisor[VM],
	ctrconfig ContainerizedVMConfig,
	devices ...virtio.VirtioDevice) (*RunningVM[VM], error) {

	id := "harpoon-oci-" + ctrconfig.ID
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

	bindMounts, mountDevices, err := PrepareContainerMounts(ctx, ctrconfig.Spec)
	if err != nil {
		return nil, errors.Errorf("preparing container mounts: %w", err)
	}

	devices = append(devices, mountDevices...)

	slog.InfoContext(ctx, "about to set up rootfs",
		"ctrconfig.RootfsPath", ctrconfig.RootfsPath,
		"ctrconfig.RootfsMounts", tint.NewPrettyValue(ctrconfig.RootfsMounts),
		"bindMounts", tint.NewPrettyValue(bindMounts),
		"spec.Root.Path", ctrconfig.Spec.Root.Path,
		"spec.Root.Readonly", ctrconfig.Spec.Root.Readonly,
	)

	// Create virtio devices using the existing rootfs directory
	ec1Devices, err := PrepareContainerVirtioDevicesFromRootfs(ctx, workingDir, ctrconfig.Spec, bindMounts, errgrp)
	if err != nil {
		return nil, errors.Errorf("creating ec1 block device from rootfs: %w", err)
	}
	devices = append(devices, ec1Devices...)

	var bootloader Bootloader

	switch ctrconfig.Platform.OS() {
	case "linux":
		bl, bldevs, err := PrepareHarpoonLinuxBootloader(ctx, workingDir, ConatinerImageConfig{
			Platform: ctrconfig.Platform,
			Memory:   ctrconfig.Memory,
			VCPUs:    ctrconfig.VCPUs,
		}, errgrp)
		if err != nil {
			return nil, errors.Errorf("getting boot loader config: %w", err)
		}
		bootloader = bl
		devices = append(devices, bldevs...)
	default:
		return nil, errors.Errorf("unsupported OS: %s", ctrconfig.Platform.OS())
	}

	if ctrconfig.Spec.Process.Terminal {
		return nil, errors.New("terminal support is not implemented yet")
	} else {
		// setup a log
		devices = append(devices, &virtio.VirtioSerialLogFile{
			Path:   filepath.Join(workingDir, "console.log"),
			Append: false,
		})
		if ctrconfig.StdinPath != "" {
			devices = append(devices, &virtio.VirtioSerialFifoFile{Path: ctrconfig.StdinPath})
		}
		if ctrconfig.StdoutPath != "" {
			devices = append(devices, &virtio.VirtioSerialFifoFile{Path: ctrconfig.StdoutPath})
		}
		if ctrconfig.StderrPath != "" {
			devices = append(devices, &virtio.VirtioSerialFifoFile{Path: ctrconfig.StderrPath})
		}
	}

	// for all the bind mounts, we need to check if they are files or directories, bind the directories to the rootfs

	netdev, hostIPPort, err := PrepareVirtualNetwork(ctx, errgrp)
	if err != nil {
		return nil, errors.Errorf("creating net device: %w", err)
	}
	devices = append(devices, netdev)

	opts := NewVMOptions{
		Vcpus:   ctrconfig.VCPUs,
		Memory:  ctrconfig.Memory,
		Devices: devices,
	}

	vm, err := hpv.NewVirtualMachine(ctx, id, &opts, bootloader)
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

	// For container runtimes, we want the VM to stay running, not wait for it to stop
	slog.InfoContext(ctx, "VM is ready for container execution")

	// Create an error channel that will receive VM state changes
	errCh := make(chan error, 1)
	go func() {
		// Wait for errgroup to finish (this handles cleanup when context is cancelled)
		if err := errgrp.Wait(); err != nil && err != context.Canceled {
			slog.ErrorContext(ctx, "error running gvproxy", "error", err)
		}

		// Wait for runtime services to finish
		if err := runErrGroup.Wait(); err != nil && err != context.Canceled {
			slog.ErrorContext(ctx, "error running runtime services", "error", err)
			errCh <- err
			return
		}

		// Only send error if VM actually encounters an error state
		stateNotify := vm.StateChangeNotify(ctx)
		for {
			select {
			case state := <-stateNotify:
				if state.StateType == VirtualMachineStateTypeError {
					errCh <- fmt.Errorf("VM entered error state")
					return
				}
				if state.StateType == VirtualMachineStateTypeStopped {
					slog.InfoContext(ctx, "VM stopped")
					errCh <- nil
					return
				}
				slog.InfoContext(ctx, "VM state changed", "state", state.StateType, "metadata", state.Metadata)
			case <-ctx.Done():
				runCancel()
				return
			}
		}
	}()

	return NewRunningVM(ctx, vm, hostIPPort, startTime, errCh), nil
}

func PrepareContainerMounts(ctx context.Context, spec *oci.Spec) ([]specs.Mount, []virtio.VirtioDevice, error) {
	bindMounts := []specs.Mount{}
	devices := []virtio.VirtioDevice{}

	for _, mount := range spec.Mounts {
		if mount.Type == "bind" {
			if fi, err := os.Stat(mount.Source); err == nil {
				var dir string
				if fi.IsDir() {
					dir = mount.Source
				} else {
					dir = filepath.Dir(mount.Source)
				}
				hash := sha256.Sum256([]byte(dir))
				tag := "bind-" + hex.EncodeToString(hash[:8])
				// create a new fs direcotry share
				shareDev, err := virtio.VirtioFsNew(mount.Source, tag)
				if err != nil {
					return nil, nil, errors.Errorf("creating share device: %w", err)
				}
				devices = append(devices, shareDev)
				bindMounts = append(bindMounts, specs.Mount{
					Type:        "ec1-virtiofs",
					Source:      tag,
					Destination: mount.Destination,
					Options:     mount.Options,
					UIDMappings: mount.UIDMappings,
					GIDMappings: mount.GIDMappings,
				})
			}
		}
	}

	return bindMounts, devices, nil
}

// PrepareContainerVirtioDevicesFromRootfs creates virtio devices using an existing rootfs directory
func PrepareContainerVirtioDevicesFromRootfs(ctx context.Context, wrkdir string, ctrconfig *oci.Spec, bindMounts []specs.Mount, wg *errgroup.Group) ([]virtio.VirtioDevice, error) {
	ec1DataPath := filepath.Join(wrkdir, "harpoon-runtime-fs-device")

	devices := []virtio.VirtioDevice{}

	err := os.MkdirAll(ec1DataPath, 0755)
	if err != nil {
		return nil, errors.Errorf("creating block device directory: %w", err)
	}

	// i think the prob is that ctrconfig.Root.Path is set to 'rootfs'
	// Create a VirtioFs device pointing to the existing rootfs directory
	blkDev, err := virtio.VirtioFsNew(ctrconfig.Root.Path, ec1init.RootfsVirtioTag)
	if err != nil {
		return nil, errors.Errorf("creating rootfs virtio device: %w", err)
	}

	// consoleAttachment := virtio.NewFileHandleDeviceAttachment(os.NewFile(uintptr(ctrconfig.StdinFD), "ptymaster"), virtio.DeviceSerial)
	// consoleConfig.SetAttachment(consoleAttachment)

	devices = append(devices, blkDev)

	specBytes, err := json.Marshal(ctrconfig)
	if err != nil {
		return nil, errors.Errorf("marshalling spec: %w", err)
	}

	bindMounts = append(bindMounts, specs.Mount{
		Type:        "virtiofs",
		Source:      ec1init.Ec1VirtioTag,
		Destination: ec1init.Ec1AbsPath,
		Options:     []string{},
	})

	mountsBytes, err := json.Marshal(bindMounts)
	if err != nil {
		return nil, errors.Errorf("marshalling mounts: %w", err)
	}

	files := map[string][]byte{
		ec1init.ContainerSpecFile:   specBytes,
		ec1init.ContainerMountsFile: mountsBytes,
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
		return nil, errors.Errorf("creating ec1 virtio device: %w", err)
	}

	devices = append(devices, ec1Dev)

	return devices, nil
}
