package vmm

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/containers/common/pkg/strongunits"
	"github.com/rs/xid"
	"gitlab.com/tozd/go/errors"

	"github.com/walteh/ec1/pkg/bootloader"
	"github.com/walteh/ec1/pkg/ext/iox"
	"github.com/walteh/ec1/pkg/ext/osx"
	"github.com/walteh/ec1/pkg/guest"
	"github.com/walteh/ec1/pkg/gvnet"
	"github.com/walteh/ec1/pkg/host"
	"github.com/walteh/ec1/pkg/initramfs"
	"github.com/walteh/ec1/pkg/port"
	"github.com/walteh/ec1/pkg/virtio"
)

func EmphericalBootLoaderConfigForGuest[VM VirtualMachine](ctx context.Context, wrkdir string, hpv Hypervisor[VM], provider VMIProvider, mem map[string]io.ReadCloser) (bootloader.Bootloader, []virtio.VirtioDevice, error) {

	var devices []virtio.VirtioDevice
	switch kt := provider.GuestKernelType(); kt {
	case guest.GuestKernelTypeLinux:
		extraArgs := ""
		if linuxVMIProvider, ok := provider.(LinuxVMIProvider); ok {

			entries := []slog.Attr{}

			if linuxVMIProvider.InitramfsPath() == "" {
				return nil, nil, errors.New("initramfs path is empty - ec1 does not yet support this yet")
			}

			initramfsReader, ok := mem[linuxVMIProvider.InitramfsPath()]
			if !ok {
				return nil, nil, errors.Errorf("initramfs file not found: %s", linuxVMIProvider.InitramfsPath())
			}

			decompressedInitBinData, err := LoadInitBinToMemory(ctx)
			if err != nil {
				return nil, nil, errors.Errorf("uncompressing init binary: %w", err)
			}

			initramfsReader, err = initramfs.InjectFileToCpio(ctx, initramfsReader, initramfs.NewExecHeader("init"), decompressedInitBinData)
			if err != nil {
				return nil, nil, errors.Errorf("preparing initramfs cpio: %w", err)
			}

			initramfsReader, err = hpv.EncodeLinuxInitramfs(ctx, initramfsReader)
			if err != nil {
				return nil, nil, errors.Errorf("encoding linux initramfs: %w", err)
			}

			initramfsPath := filepath.Join(wrkdir, "initramfs.cpio.gz")

			slog.InfoContext(ctx, "writing initramfs")
			initrdSize, err := osx.WriteFileFromReader(ctx, initramfsPath, initramfsReader, 0644)
			if err != nil {
				return nil, nil, errors.Errorf("creating initramfs file: %w", err)
			}

			slog.InfoContext(ctx, "initramfs size", "size", initrdSize)

			entries = append(entries, slog.Group("initramfs", "path", initramfsPath, "size", initrdSize))

			initramfsReader.Close()

			if linuxVMIProvider.RootfsPath() != "" {
				rootfsReader, ok := mem[linuxVMIProvider.RootfsPath()]
				if !ok {
					return nil, nil, errors.Errorf("rootfs file not found: %s", linuxVMIProvider.RootfsPath())
				}

				rootfsReader, err = hpv.EncodeLinuxRootfs(ctx, rootfsReader)
				if err != nil {
					return nil, nil, errors.Errorf("encoding linux rootfs: %w", err)
				}

				rootfsPath := filepath.Join(wrkdir, "rootfs")
				rootfsSize, err := osx.WriteFileFromReader(ctx, rootfsPath, rootfsReader, 0644)
				if err != nil {
					return nil, nil, errors.Errorf("creating rootfs file: %w", err)
				}

				entries = append(entries, slog.Group("rootfs", "path", rootfsPath, "size", rootfsSize))

				rootfsReader.Close()

				blkDev, err := virtio.NVMExpressControllerNew(rootfsPath)
				if err != nil {
					return nil, nil, errors.Errorf("creating rootfs file: %w", err)
				}

				devices = append(devices, blkDev)

				extraArgs = "  root=/dev/nvme0n1p2"
			}

			if linuxVMIProvider.KernelPath() == "" {
				return nil, nil, errors.New("kernel path is empty - ec1 does not yet support this yet")
			}

			kernelReader, ok := mem[linuxVMIProvider.KernelPath()]
			if !ok {
				return nil, nil, errors.Errorf("kernel file not found: %s", linuxVMIProvider.KernelPath())
			}

			kernelReader, err = hpv.EncodeLinuxKernel(ctx, kernelReader)
			if err != nil {
				return nil, nil, errors.Errorf("encoding linux kernel: %w", err)
			}

			kernelPath := filepath.Join(wrkdir, "vmlinuz")

			kernelSize, err := osx.WriteFileFromReader(ctx, kernelPath, kernelReader, 0644)
			if err != nil {
				return nil, nil, errors.Errorf("creating kernel file: %w", err)
			}

			entries = append(entries, slog.Group("kernel", "path", kernelPath, "size", kernelSize))

			kernelReader.Close()

			// cmdLine := linuxVMIProvider.KernelArgs() + " console=hvc0 cloud-init=disabled network-config=disabled" + extraArgs
			cmdLine := linuxVMIProvider.KernelArgs() + " console=hvc0 " + extraArgs

			slog.LogAttrs(ctx, slog.LevelInfo, "linux boot loader ready", entries...)

			return &bootloader.LinuxBootloader{
				InitrdPath:    initramfsPath,
				VmlinuzPath:   kernelPath,
				KernelCmdLine: cmdLine,
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

func cacheKeyFromMap(m map[string]string, vmi VMIProvider) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	keys = append(keys, reflect.TypeOf(vmi).String())
	dat := sha256.Sum256([]byte(strings.Join(keys, "-")))
	return hex.EncodeToString(dat[:])
}

func osCreationFunc(path string) (io.WriteCloser, error) {
	return os.Create(path)
}

func osDirCreationFunc(prefix string) func(path string) (io.WriteCloser, error) {
	return func(path string) (io.WriteCloser, error) {
		return os.Create(filepath.Join(prefix, path))
	}
}

func RunVirtualMachine[VM VirtualMachine](ctx context.Context, hpv Hypervisor[VM], vmi VMIProvider, vcpus uint, memory strongunits.B) (*RunningVM[VM], error) {
	id := "vm-" + xid.New().String()

	startTime := time.Now()

	workingDir, err := host.EmphiricalVMCacheDir(ctx, id)
	if err != nil {
		return nil, err
	}

	globalCacheDir, err := host.CacheDirPrefix()
	if err != nil {
		return nil, errors.Errorf("creating global cache directory: %w", err)
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

	diskImageURLVMIProvider, ok := vmi.(DownloadableVMIProvider)
	if !ok {
		return nil, errors.New("vmi does not support downloads, ec1 does not yet support this")
	}

	dls := diskImageURLVMIProvider.Downloads()

	files, err := host.DownloadAndExtractVMI(ctx, dls)
	if err != nil {
		return nil, errors.Errorf("downloading and extracting VMI: %w", err)
	}

	cacheKey := cacheKeyFromMap(dls, vmi)

	extractionCacheDir := filepath.Join(globalCacheDir, "extractions", cacheKey)

	// create cache directory if it doesn't exist
	err = os.MkdirAll(extractionCacheDir, 0755)
	if err != nil && !os.IsExist(err) {
		return nil, errors.Errorf("creating cache directory: %w", err)
	}

	// load files from cache
	entries, err := os.ReadDir(extractionCacheDir)
	if err != nil {
		return nil, errors.Errorf("reading cache directory: %w", err)
	}

	// open files from cache
	for _, entry := range entries {
		filePath := filepath.Join(extractionCacheDir, entry.Name())
		files[entry.Name()], err = os.Open(filePath)
		if err != nil {
			return nil, errors.Errorf("opening file: %w", err)
		}
	}

	// extract files
	extractedFiles, err := diskImageURLVMIProvider.ExtractDownloads(ctx, files)
	if err != nil {
		return nil, errors.Errorf("extracting downloads: %w", err)
	}

	tmpExtractDir, err := os.MkdirTemp("", "ec1-extract")
	if err != nil {
		return nil, errors.Errorf("creating temporary extraction directory: %w", err)
	}

	for name := range extractedFiles {
		pr, pw := io.Pipe()
		fr := extractedFiles[name]
		tr := io.TeeReader(fr, pw)
		extractedFiles[name] = iox.PreservedNopCloser(tr)

		// we almost are able to do this in the above for loop, but in cases we reuse the cache
		// we need to make sure we are not reading and writing to the same file
		// so instead we do the next best thing and do in a background go routine
		go func() {
			defer pw.Close()
			defer fr.Close()
			defer pr.Close()

			_, err = osx.WriteFileFromReader(ctx, filepath.Join(tmpExtractDir, name), pr, 0644)
			if err != nil {
				slog.WarnContext(ctx, "problem writing file to extraction cache, this is ignored", "file", name, "error", err)
			}
		}()
	}

	bl, bldev, err := EmphericalBootLoaderConfigForGuest(ctx, workingDir, hpv, vmi, extractedFiles)
	if err != nil {
		return nil, errors.Errorf("getting boot loader config: %w", err)
	}
	devices = append(devices, bldev...)

	// cmd.Exec a checksum.txt of the working directory, save it to the extraction cache dir
	// find . -type f -not -name "checksums.txt" -print0 | xargs -0 shasum -a 256 > checksums.txt
	cmd := exec.Command("sh", "-c", "find . -type f -not -name 'checksums.txt' -print0 | xargs -0 shasum -a 256 > checksums.txt")
	cmd.Dir = workingDir
	output, err := cmd.Output()
	if err != nil {
		return nil, errors.Errorf("getting sha256 checksum of working directory: %w: %s", err, string(output))
	}

	// checksumFile := filepath.Join(workingDir, "checksums.txt")
	// err = os.WriteFile(checksumFile, output, 0644)
	// if err != nil {
	// 	return nil, errors.Errorf("writing checksum file: %w", err)
	// }

	// slog.InfoContext(ctx, "sha256 checksum of working directory", "checksum", string(output))

	// we don't want to save the files if they were not valid
	go func() {
		defer os.RemoveAll(tmpExtractDir)
		err = osx.Copy(ctx, tmpExtractDir, extractionCacheDir)
		if err != nil {
			slog.WarnContext(ctx, "problem copying files to extraction cache, this is ignored", "error", err)
		}
	}()

	// // copy extracted files to the extraction cache dir
	// for name := range extractedFiles {
	// 	pr, pw := io.Pipe()
	// 	extractedFiles[name] = iox.PreservedNopCloser(io.TeeReader(extractedFiles[name], pw))
	// 	// we almost are able to do this in the above for loop, but in cases we reuse the cache
	// 	// we need to make sure we are not reading and writing to the same file
	// 	// so instead we do the next best thing and do in a background go routine
	// 	go func() {
	// 		_, err = osx.WriteFileFromReader(ctx, filepath.Join(extractionCacheDir, name), pr, 0644)
	// 		if err != nil {
	// 			slog.WarnContext(ctx, "problem writing file to extraction cache, this is ignored", "file", name, "error", err)
	// 		}
	// 	}()
	// }

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

	return NewRunningVM(ctx, vm, hostIPPort, startTime, errCh), nil

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
