package puipui

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"golang.org/x/crypto/ssh"
	"golang.org/x/mod/semver"

	"github.com/walteh/ec1/pkg/machines/bootloader"
	"github.com/walteh/ec1/pkg/machines/guest"
	"github.com/walteh/ec1/pkg/machines/host"
	"github.com/walteh/ec1/pkg/vmm"
)

// const puipuiMemoryMiB = 1 * 1024
// const puipuiCPUs = 2

const puipuiVersion = "v1.0.3"

var (
	_ vmm.VMIProvider                = &PuiPuiProvider{}
	_ vmm.LinuxVMIProvider           = &PuiPuiProvider{}
	_ vmm.CustomExtractorVMIProvider = &PuiPuiProvider{}
)

type PuiPuiProvider struct {
	// vmlinuz    string
	// initramfs  string
	// kernelArgs string
}

// BootLoaderConfig implements vmm.LinuxVMIProvider.
func (prov *PuiPuiProvider) BootLoaderConfig(cacheDir string) *bootloader.LinuxBootloader {
	out := "bzImage"
	if host.CurrentKernelArch() == "aarch64" {
		out = "Image"
	}
	return &bootloader.LinuxBootloader{
		VmlinuzPath:   filepath.Join(cacheDir, out),
		KernelCmdLine: "quiet",
		InitrdPath:    filepath.Join(cacheDir, "initramfs.cpio.gz"),
	}
}

// BootProvisioners implements vmm.VMIProvider.
func (prov *PuiPuiProvider) BootProvisioners() []vmm.BootProvisioner {
	return []vmm.BootProvisioner{}
}

// DiskImageURL implements vmm.VMIProvider.
func (prov *PuiPuiProvider) DiskImageURL() string {
	return prov.URL()
}

// GuestKernelType implements vmm.VMIProvider.
func (prov *PuiPuiProvider) GuestKernelType() guest.GuestKernelType {
	return guest.GuestKernelTypeLinux
}

// RuntimeProvisioners implements vmm.VMIProvider.
func (prov *PuiPuiProvider) RuntimeProvisioners() []vmm.RuntimeProvisioner {
	return []vmm.RuntimeProvisioner{}
}

// SupportsEFI implements vmm.VMIProvider.
func (prov *PuiPuiProvider) SupportsEFI() bool {
	return false
}

func NewPuipuiProvider() *PuiPuiProvider {
	return &PuiPuiProvider{}
}

func (prov *PuiPuiProvider) Name() string {
	return "puipui"
}

func (prov *PuiPuiProvider) Version() string {
	return semver.Canonical(puipuiVersion)
}

func (prov *PuiPuiProvider) URL() string {
	return fmt.Sprintf("https://github.com/Code-Hex/puipui-linux/releases/download/%s/puipui_linux_%s_%s.tar.gz", puipuiVersion, puipuiVersion, host.CurrentKernelArch())
}

func (prov *PuiPuiProvider) CustomExtraction(ctx context.Context, cacheDir string) error {
	if host.CurrentKernelArch() == "aarch64" {
		// we need to extract the Image.gz file
		compressed, err := os.ReadFile(filepath.Join(cacheDir, "Image.gz"))
		if err != nil {
			return err
		}

		gzipped, err := gzip.NewReader(bytes.NewReader(compressed))
		if err != nil {
			return err
		}
		defer gzipped.Close()

		unzipped, err := io.ReadAll(gzipped)
		if err != nil {
			return err
		}

		err = os.WriteFile(filepath.Join(cacheDir, "Image"), unzipped, 0644)
		if err != nil {
			return err
		}
	}

	return nil
}

// func (prov *PuiPuiProvider) Initialize(ctx context.Context, cacheDir string) error {
// 	filez, err := os.ReadDir(cacheDir)
// 	if err != nil {
// 		return err
// 	}
// 	files := []string{}
// 	for _, file := range filez {
// 		files = append(files, filepath.Join(cacheDir, file.Name()))
// 	}
// 	prov.vmlinuz, err = findKernel(ctx, files)
// 	if err != nil {
// 		return err
// 	}
// 	prov.initramfs, err = findFile(files, "initramfs.cpio.gz")
// 	if err != nil {
// 		return err
// 	}
// 	prov.kernelArgs = "quiet"
// 	return nil
// }

// func (puipui *PuiPuiProvider) ToVirtualMachine(ctx context.Context) (*config.VirtualMachine, error) {
// 	bootloader := config.NewLinuxBootloader(puipui.vmlinuz, puipui.kernelArgs, puipui.initramfs)
// 	vm := config.NewVirtualMachine(puipuiCPUs, puipuiMemoryMiB, bootloader)
// 	return vm, nil
// }

func (puipui *PuiPuiProvider) ShutdownCommand() string {
	return "poweroff"
}

func (puipui *PuiPuiProvider) SSHConfig() *ssh.ClientConfig {
	return &ssh.ClientConfig{
		User: "root",
		Auth: []ssh.AuthMethod{ssh.Password("passwd")},
		// #nosec 106 -- the host SSH key of the VM will change each time it boots
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
}

// func findKernel(ctx context.Context, files []string) (string, error) {
// 	switch runtime.GOARCH {
// 	case "amd64":
// 		return findFile(files, "bzImage")
// 	case "arm64":
// 		compressed, err := findFile(files, "Image.gz")
// 		if err != nil {
// 			return "", err
// 		}

// 		dir := filepath.Dir(compressed)

// 		err = extractIntoDir(ctx, compressed, dir)
// 		if err != nil {
// 			return "", err
// 		}

// 		expectedOutFile := filepath.Join(dir, "Image")

// 		// make sure the file is in the dir
// 		if _, err := os.Stat(expectedOutFile); err != nil {
// 			return "", errors.Errorf("expected file %s not found", expectedOutFile)
// 		}

// 		return expectedOutFile, nil
// 	default:
// 		return "", fmt.Errorf("unsupported architecture '%s'", runtime.GOARCH)
// 	}
// }
