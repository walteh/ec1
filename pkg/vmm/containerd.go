package vmm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/containerd/containerd/v2/pkg/oci"
	"github.com/containers/common/pkg/strongunits"
	"github.com/rs/xid"
	"gitlab.com/tozd/go/errors"

	"github.com/walteh/ec1/pkg/ec1init"
	"github.com/walteh/ec1/pkg/ext/osx"
	"github.com/walteh/ec1/pkg/host"
	"github.com/walteh/ec1/pkg/units"
	"github.com/walteh/ec1/pkg/virtio"
)

type ContainerizedVMConfig struct {
	RootfsPath string
	StderrFD   int
	StdoutFD   int
	StdinFD    int
	Spec       *oci.Spec
	Memory     strongunits.B
	VCPUs      uint64
	Platform   units.Platform
}

// NewContainerizedVirtualMachineFromRootfs creates a VM using an already-prepared rootfs directory
// This is used by container runtimes like containerd that have already prepared the rootfs
func NewContainerizedVirtualMachineFromRootfs[VM VirtualMachine](
	ctx context.Context,
	hpv Hypervisor[VM],
	ctrconfig ContainerizedVMConfig,
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

	// Create virtio devices using the existing rootfs directory
	ec1Devices, err := PrepareContainerVirtioDevicesFromRootfs(ctx, workingDir, ctrconfig, errgrp)
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

	devices = append(devices, &virtio.VirtioSerial{
		UsesStdio: !ctrconfig.Spec.Process.Terminal,
		RawFDs: func() (int, int) {
			return ctrconfig.StdinFD, ctrconfig.StdoutFD
		},
	})

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
			case <-ctx.Done():
				runCancel()
				return
			}
		}
	}()

	return NewRunningVM(ctx, vm, hostIPPort, startTime, errCh), nil
}

// PrepareContainerVirtioDevicesFromRootfs creates virtio devices using an existing rootfs directory
func PrepareContainerVirtioDevicesFromRootfs(ctx context.Context, wrkdir string, ctrconfig ContainerizedVMConfig, wg *errgroup.Group) ([]virtio.VirtioDevice, error) {
	ec1DataPath := filepath.Join(wrkdir, "harpoon-runtime-fs-device")

	devices := []virtio.VirtioDevice{}

	err := os.MkdirAll(ec1DataPath, 0755)
	if err != nil {
		return nil, errors.Errorf("creating block device directory: %w", err)
	}

	// Create a VirtioFs device pointing to the existing rootfs directory
	blkDev, err := virtio.VirtioFsNew(ctrconfig.RootfsPath, ec1init.RootfsVirtioTag)
	if err != nil {
		return nil, errors.Errorf("creating rootfs virtio device: %w", err)
	}

	// consoleAttachment := virtio.NewFileHandleDeviceAttachment(os.NewFile(uintptr(ctrconfig.StdinFD), "ptymaster"), virtio.DeviceSerial)
	// consoleConfig.SetAttachment(consoleAttachment)

	devices = append(devices, blkDev)

	// Create minimal container metadata since we don't have full image info
	// TODO: Extract actual metadata from the OCI spec if needed
	metadata := map[string]interface{}{
		"rootfs": map[string]interface{}{
			"type":     "layers",
			"diff_ids": []string{}, // Empty since we don't have layer info
		},
		"config": map[string]interface{}{
			"Env": []string{}, // Default empty environment
		},
	}

	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		return nil, errors.Errorf("marshalling metadata: %w", err)
	}

	files := map[string][]byte{
		ec1init.ContainerManifestFile: metadataBytes,
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
