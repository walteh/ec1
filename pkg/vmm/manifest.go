package vmm

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/containers/common/pkg/strongunits"
	"github.com/rs/xid"
	"gitlab.com/tozd/go/errors"

	"github.com/walteh/ec1/pkg/ec1init"
	"github.com/walteh/ec1/pkg/ext/osx"
	"github.com/walteh/ec1/pkg/host"
	"github.com/walteh/ec1/pkg/oci"
	"github.com/walteh/ec1/pkg/units"
	"github.com/walteh/ec1/pkg/virtio"
)

type ManifestImageConfig struct {
	ImageRef     string
	Platform     units.Platform
	Memory       strongunits.B
	VCPUs        uint64
	StdinReader  io.Reader
	StdoutWriter io.Writer
	StderrWriter io.Writer
}

func NewManifestVirtualMachine[VM VirtualMachine](
	ctx context.Context,
	hpv Hypervisor[VM],
	cache oci.ImageFetchConverter,
	imageConfig ManifestImageConfig,
	devices ...virtio.VirtioDevice) (*RunningVM[VM], error) {

	id := "vm-manifest-" + xid.New().String()
	errgrp, ctx := errgroup.WithContext(ctx)

	ctx = appendContext(ctx, id)

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

	var bootloader Bootloader

	switch imageConfig.Platform.OS() {
	case "linux":
		bl, bldevs, err := PrepareHarpoonLinuxBootloaderAsync(ctx, workingDir, imageConfig.Platform, errgrp)
		if err != nil {
			return nil, errors.Errorf("getting boot loader config: %w", err)
		}
		bootloader = bl
		devices = append(devices, bldevs...)
	default:
		return nil, errors.Errorf("unsupported OS: %s", imageConfig.Platform.OS())
	}

	devices = append(devices, &virtio.VirtioSerialLogFile{
		Path:   filepath.Join(workingDir, "console.log"),
		Append: false,
	})

	netdev, hostIPPort, err := PrepareVirtualNetwork(ctx)
	if err != nil {
		return nil, errors.Errorf("creating net device: %w", err)
	}
	devices = append(devices, netdev.VirtioNetDevice())

	opts := NewVMOptions{
		Vcpus:   imageConfig.VCPUs,
		Memory:  imageConfig.Memory,
		Devices: devices,
	}

	if err = errgrp.Wait(); err != nil {
		return nil, errors.Errorf("error waiting for errgroup: %w", err)
	}

	vm, err := hpv.NewVirtualMachine(ctx, id, &opts, bootloader)
	if err != nil {
		return nil, errors.Errorf("creating virtual machine: %w", err)
	}

	runner := &RunningVM[VM]{
		bootloader:             bootloader,
		start:                  startTime,
		vm:                     vm,
		stdin:                  imageConfig.StdinReader,
		stdout:                 imageConfig.StdoutWriter,
		stderr:                 imageConfig.StderrWriter,
		portOnHostIP:           hostIPPort,
		wait:                   make(chan error, 1),
		guestServiceConnection: nil,
		workingDir:             workingDir,
		netdev:                 netdev,
	}

	// if ctx.Err() != nil {
	// 	return nil, errors.Errorf("ahhh context cancelled: %w", ctx.Err())
	// }

	// err = bootContainerVM(ctx, vm)
	// if err != nil {
	// 	return nil, errors.Errorf("booting virtual machine: %w", err)
	// }

	// errgrp.Go(func() error {
	// 	return vm.ServeBackgroundTasks(ctx)
	// })

	// // For container runtimes, we want the VM to stay running, not wait for it to stop
	// slog.InfoContext(ctx, "VM is ready for container execution")

	// // Create an error channel that will receive VM state changes
	// errCh := make(chan error, 1)
	// go func() {
	// 	// Wait for errgroup to finish (this handles cleanup when context is cancelled)
	// 	if err := errgrp.Wait(); err != nil && err != context.Canceled {
	// 		slog.ErrorContext(ctx, "error running gvproxy", "error", err)
	// 	}

	// 	// Only send error if VM actually encounters an error state
	// 	stateNotify := vm.StateChangeNotify(ctx)
	// 	for {
	// 		select {
	// 		case state := <-stateNotify:
	// 			if state.StateType == VirtualMachineStateTypeError {
	// 				errCh <- fmt.Errorf("VM entered error state")
	// 				return
	// 			}
	// 			if state.StateType == VirtualMachineStateTypeStopped {
	// 				slog.InfoContext(ctx, "VM stopped")
	// 				errCh <- nil
	// 				return
	// 			}
	// 		case <-ctx.Done():
	// 			return
	// 		}
	// 	}
	// }()

	return runner, nil
}

func PrepareContainerVirtioDevices(ctx context.Context, wrkdir string, imageConfig ManifestImageConfig, cache oci.ImageFetchConverter, wg *errgroup.Group) ([]virtio.VirtioDevice, error) {

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

func bootContainerVM[VM VirtualMachine](ctx context.Context, vm VM) error {
	bootCtx, bootCancel := context.WithCancel(ctx)
	errGroup, ctx := errgroup.WithContext(bootCtx)
	defer func() {
		if r := recover(); r != nil {
			slog.ErrorContext(ctx, "panic in bootContainerVM", "panic", r)
			panic(r)
		}
		// clean up the boot provisioners - this shouldn't throw an error because they prob are going to throw something
		bootCancel()
		if err := errGroup.Wait(); err != nil {
			slog.DebugContext(ctx, "error running boot provisioners", "error", err)
		}

	}()

	go func() {
		for {
			select {
			case <-bootCtx.Done():
				return
			case <-vm.StateChangeNotify(bootCtx):
				slog.InfoContext(bootCtx, "virtual machine state changed", "state", vm.CurrentState())
			}
		}
	}()

	slog.InfoContext(ctx, "starting virtual machine")

	if err := vm.Start(ctx); err != nil {
		return errors.Errorf("starting virtual machine: %w", err)
	}

	if err := WaitForVMState(ctx, vm, VirtualMachineStateTypeRunning, time.After(30*time.Second)); err != nil {
		return errors.Errorf("waiting for virtual machine to start: %w", err)
	}

	slog.InfoContext(ctx, "virtual machine is running")

	return nil
}

// func runContainerVM[VM VirtualMachine](ctx context.Context, vm VM) (func(), error) {
// 	runCtx, bootCancel := context.WithCancel(ctx)

// 	// if err := vm.ListenNetworkBlockDevices(runCtx); err != nil {
// 	// 	bootCancel()
// 	// 	return nil, errors.Errorf("listening network block devices: %w", err)
// 	// }

// 	// if err := StartVSockDevices(runCtx, vm); err != nil {
// 	// 	bootCancel()
// 	// 	return nil, errors.Errorf("starting vsock devices: %w", err)
// 	// }

// 	// gpuDevs := virtio.VirtioDevicesOfType[*virtio.VirtioGPU](vm.Devices())
// 	// for _, gpuDev := range gpuDevs {
// 	// 	if gpuDev.UsesGUI {
// 	// 		runtime.LockOSThread()
// 	// 		err := vm.StartGraphicApplication(float64(gpuDev.Width), float64(gpuDev.Height))
// 	// 		runtime.UnlockOSThread()
// 	// 		if err != nil {
// 	// 			bootCancel()
// 	// 			return nil, errors.Errorf("starting graphic application: %w", err)
// 	// 		}
// 	// 		break
// 	// 	} else {
// 	// 		slog.DebugContext(ctx, "not starting GUI")
// 	// 	}
// 	// }

// 	return bootCancel, nil

// }
