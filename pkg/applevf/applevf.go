//go:build darwin
// +build darwin

package applevf

/*
Copyright 2021, Red Hat, Inc - All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/Code-Hex/vz/v3"
	"github.com/crc-org/vfkit/pkg/cmdline"
	"github.com/kdomanski/iso9660"
	"github.com/walteh/ec1/pkg/hypervisors/vf"
	"github.com/walteh/ec1/pkg/hypervisors/vf/config"
	"github.com/walteh/ec1/pkg/hypervisors/vf/rest"
	restvf "github.com/walteh/ec1/pkg/hypervisors/vf/rest/vf"

	"gitlab.com/tozd/go/errors"

	"github.com/crc-org/vfkit/pkg/util"
)

func newLegacyBootloader(opts *cmdline.Options) config.Bootloader {
	if opts.VmlinuzPath == "" && opts.KernelCmdline == "" && opts.InitrdPath == "" {
		return nil
	}

	return config.NewLinuxBootloader(
		opts.VmlinuzPath,
		opts.KernelCmdline,
		opts.InitrdPath,
	)
}

func NewBootloaderConfiguration(ctx context.Context, opts *cmdline.Options) (config.Bootloader, error) {
	legacyBootloader := newLegacyBootloader(opts)

	if legacyBootloader != nil {
		return legacyBootloader, nil
	}

	return config.BootloaderFromCmdLine(opts.Bootloader.GetSlice())
}

func NewVMConfiguration(ctx context.Context, opts *cmdline.Options) (*config.VirtualMachine, error) {
	bootloader, err := NewBootloaderConfiguration(ctx, opts)
	if err != nil {
		return nil, err
	}

	slog.InfoContext(ctx, "options", slog.Any("options", opts))
	slog.InfoContext(ctx, "boot parameters", "bootloader", bootloader)

	vmConfig := config.NewVirtualMachine(
		opts.Vcpus,
		uint64(opts.MemoryMiB),
		bootloader,
	)
	slog.InfoContext(ctx, "virtual machine parameters", "vCPUs", opts.Vcpus, "memory", opts.MemoryMiB)

	if err := vmConfig.AddTimeSyncFromCmdLine(opts.TimeSync); err != nil {
		return nil, err
	}

	cloudInitISO, err := GenerateCloudInitImage(ctx, opts.CloudInitFiles.GetSlice())
	if err != nil {
		return nil, err
	}

	// if it generated a valid cloudinit config ISO file we add it to the devices
	if cloudInitISO != "" {
		opts.Devices = append(opts.Devices, fmt.Sprintf("virtio-blk,path=%s", cloudInitISO))
	}

	if err := vmConfig.AddDevicesFromCmdLine(opts.Devices); err != nil {
		return nil, err
	}

	if err := vmConfig.AddIgnitionFileFromCmdLine(opts.IgnitionPath); err != nil {
		return nil, fmt.Errorf("failed to add ignition file: %w", err)
	}

	return vmConfig, nil
}

func waitForVMState(ctx context.Context, vm *vf.VirtualMachine, state vz.VirtualMachineState, timeout <-chan time.Time) error {
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGPIPE)

	slog.DebugContext(ctx, "waiting for VM state", "state", state, "current state", vm.State())

	for {
		select {
		case s := <-signalCh:
			slog.DebugContext(ctx, "ignoring signal", "signal", s)
		case newState := <-vm.StateChangedNotify():

			slog.DebugContext(ctx, "VM state changed", "state", newState)

			if newState == state {
				return nil
			}
			if newState == vz.VirtualMachineStateError {
				return fmt.Errorf("hypervisor virtualization error")
			}
		case <-timeout:
			return fmt.Errorf("timeout waiting for VM state")
		}
	}
}

func RunVFKit(ctx context.Context, vmConfig *config.VirtualMachine, opts *cmdline.Options) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	util.SetupExitSignalHandling()

	gpuDevs := vmConfig.VirtioGPUDevices()
	if opts.UseGUI && len(gpuDevs) > 0 {
		gpuDevs[0].UsesGUI = true
	}

	vfVM, err := vf.NewVirtualMachine(*vmConfig)
	if err != nil {
		slog.ErrorContext(ctx, "creating virtual machine", "error", err, slog.Any("config", vmConfig))
		return errors.Errorf("creating virtual machine: %w", err)
	}

	// Do not enable the rests server if user sets scheme to None
	if opts.RestfulURI != cmdline.DefaultRestfulURI {
		restVM := restvf.NewVzVirtualMachine(vfVM)
		srv, err := rest.NewServer(restVM, restVM, opts.RestfulURI)
		if err != nil {
			return errors.WithStack(errors.Errorf("creating rest server: %w", err))
		}
		srv.Start()
	}
	return runVirtualMachine(ctx, vmConfig, vfVM)
}

func runVirtualMachine(ctx context.Context, vmConfig *config.VirtualMachine, vm *vf.VirtualMachine) error {
	if vm.Config().Ignition != nil {
		file, err := os.Open(vmConfig.Ignition.ConfigPath)
		if err != nil {
			return errors.Errorf("opening ignition file: %w", err)
		}
		go func() {
			defer file.Close()
			reader := file
			err = StartIgnitionProvisionerServer(ctx, reader, vmConfig.Ignition.SocketPath)
			slog.ErrorContext(ctx, "ignition vsock server exited", "error", err)
		}()
	}

	if err := vm.Start(); err != nil {
		return errors.Errorf("starting virtual machine: %w", err)
	}

	if err := waitForVMState(ctx, vm, vz.VirtualMachineStateRunning, time.After(5*time.Second)); err != nil {
		return errors.Errorf("waiting for virtual machine to start: %w", err)
	}
	slog.InfoContext(ctx, "virtual machine is running")

	vsockDevs := vmConfig.VirtioVsockDevices()
	for _, vsock := range vsockDevs {
		port := vsock.Port
		socketURL := vsock.SocketURL
		if socketURL == "" {
			// the timesync code adds a vsock device without an associated URL.
			continue
		}
		var listenStr string
		if vsock.Listen {
			listenStr = " (listening)"
		}
		slog.InfoContext(ctx, "Exposing vsock port", "port", port, "socketURL", socketURL, "listenStr", listenStr)
		closer, err := vf.ExposeVsock(vm, port, socketURL, vsock.Listen)
		if err != nil {
			slog.WarnContext(ctx, "error exposing vsock port", "port", port, "error", err)
			continue
		}
		defer closer.Close()
	}

	if err := vf.ListenNetworkBlockDevices(vm); err != nil {
		// slog.DebugContext(ctx, "error listening network block devices", "error", err)
		return errors.Errorf("listening network block devices: %w", err)
	}

	if err := setupGuestTimeSync(ctx, vm, vmConfig.TimeSync()); err != nil {
		slog.WarnContext(ctx, "Error configuring guest time synchronization", "error", err)
		// return errors.Errorf("configuring guest time synchronization: %w", err)
	}

	slog.InfoContext(ctx, "waiting for VM to stop")

	errCh := make(chan error, 1)
	go func() {
		if err := waitForVMState(ctx, vm, vz.VirtualMachineStateStopped, nil); err != nil {
			errCh <- fmt.Errorf("virtualization error: %v", err)
		} else {
			slog.InfoContext(ctx, "VM is stopped")
			errCh <- nil
		}
	}()

	if len(vmConfig.VirtioGPUDevices()) == 0 {
		slog.DebugContext(ctx, "no GPU devices found in vmConfig")
		// close(errCh)
		// return errors.New("no GPU devices found in vmConfig")
	}

	for _, gpuDev := range vmConfig.VirtioGPUDevices() {
		if gpuDev.UsesGUI {
			runtime.LockOSThread()
			err := vm.StartGraphicApplication(float64(gpuDev.Width), float64(gpuDev.Height))
			runtime.UnlockOSThread()
			if err != nil {
				return errors.Errorf("starting graphic application: %w", err)
			}
			break
		} else {
			slog.DebugContext(ctx, "not starting GUI")
		}
	}

	return <-errCh
}

func StartIgnitionProvisionerServer(ctx context.Context, ignitionReader io.Reader, ignitionSocketPath string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		_, err := io.Copy(w, ignitionReader)
		if err != nil {
			slog.ErrorContext(ctx, "failed to serve ignition file", "error", err)
		}
	})

	listener, err := net.Listen("unix", ignitionSocketPath)
	if err != nil {
		return errors.Errorf("listening on ignition socket: %w", err)
	}

	util.RegisterExitHandler(func() {
		os.Remove(ignitionSocketPath)
	})

	defer func() {
		if err := listener.Close(); err != nil {
			slog.ErrorContext(ctx, "failed to close ignition socket", "error", err)
		}
	}()

	srv := &http.Server{
		Handler:           mux,
		Addr:              ignitionSocketPath,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	slog.DebugContext(ctx, "ignition socket", "socket", ignitionSocketPath)
	return srv.Serve(listener)
}

type CloudInitFiles struct {
	UserData string
	MetaData string
}

// it generates a cloud init image by taking the files passed by the user
// as cloud-init expects files with a specific name (e.g user-data, meta-data) we check the filenames to retrieve the correct info
// if some file is not passed by the user, an empty file will be copied to the cloud-init ISO to
// guarantee it to work (user-data and meta-data files are mandatory and both must exists, even if they are empty)
// if both files are missing it returns an error
func (me *CloudInitFiles) GenerateISO(ctx context.Context) (string, error) {

	configFiles := map[string]io.Reader{
		"user-data": strings.NewReader(me.UserData),
		"meta-data": strings.NewReader(me.MetaData),
	}

	return createCloudInitISO(ctx, configFiles)
}

func GenerateCloudInitImage(ctx context.Context, files []string) (string, error) {
	if len(files) == 0 {
		return "", nil
	}

	configFiles := map[string]io.Reader{
		"user-data": nil,
		"meta-data": nil,
	}
	hasConfigFile := false
	for _, path := range files {
		if path == "" {
			continue
		}
		file, err := os.Open(path)
		if err != nil {
			return "", errors.Errorf("opening file: %w", err)
		}
		defer file.Close()

		filename := filepath.Base(path)
		if _, ok := configFiles[filename]; ok {
			hasConfigFile = true
			configFiles[filename] = file
		}
	}

	if !hasConfigFile {
		return "", fmt.Errorf("cloud-init needs user-data and meta-data files to work")
	}

	return createCloudInitISO(ctx, configFiles)
}

// It generates a temp ISO file containing the files passed by the user
// It also register an exit handler to delete the file when vfkit exits
func createCloudInitISO(ctx context.Context, files map[string]io.Reader) (string, error) {
	writer, err := iso9660.NewWriter()
	if err != nil {
		return "", fmt.Errorf("failed to create writer: %w", err)
	}

	defer func() {
		if err := writer.Cleanup(); err != nil {
			slog.ErrorContext(ctx, "failed to cleanup writer", "error", err)
		}
	}()

	for name, reader := range files {
		// if reader is nil, we set it to an empty file
		if reader == nil {
			reader = bytes.NewReader([]byte{})
		}
		err = writer.AddFile(reader, name)
		if err != nil {
			return "", fmt.Errorf("failed to add %s file: %w", name, err)
		}
	}

	isoFile, err := os.CreateTemp("", "vfkit-cloudinit-*.iso")
	if err != nil {
		return "", fmt.Errorf("unable to create temporary cloud-init ISO file: %w", err)
	}

	defer func() {
		if err := isoFile.Close(); err != nil {
			slog.ErrorContext(ctx, "failed to close cloud-init ISO file", "error", err)
		}
	}()

	// register handler to remove isoFile when exiting
	util.RegisterExitHandler(func() {
		os.Remove(isoFile.Name())
	})

	err = writer.WriteTo(isoFile, "cidata")
	if err != nil {
		return "", fmt.Errorf("failed to write cloud-init ISO image: %w", err)
	}

	return isoFile.Name(), nil
}
