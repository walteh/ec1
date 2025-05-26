package harpoon

import (
	"bytes"
	"context"
	"io"

	"golang.org/x/crypto/ssh"
	"golang.org/x/mod/semver"

	"github.com/mholt/archives"
	"gitlab.com/tozd/go/errors"

	"github.com/walteh/ec1/gen/initramfs/initramfs_aarch64"
	"github.com/walteh/ec1/gen/kernel/vmlinux_aarch64"
	"github.com/walteh/ec1/pkg/guest"
	"github.com/walteh/ec1/pkg/vmm"
)

const puipuiVersion = "v0.0.1"

const (
	extractedCpioName   = "initramfs.ec1-extract." + initramfs_aarch64.BinaryGZChecksum + ".cpio"
	extractedKernelName = "vmlinux.ec1-extract." + vmlinux_aarch64.BinaryXZChecksum
)

var (
	_ vmm.VMIProvider             = &HarpoonProvider{}
	_ vmm.LinuxVMIProvider        = &HarpoonProvider{}
	_ vmm.DownloadableVMIProvider = &HarpoonProvider{}
)

type HarpoonProvider struct {
	// vmlinuz    string
	// initramfs  string
	// kernelArgs string
}

// BootLoaderConfig implements vmm.LinuxVMIProvider.

// BootProvisioners implements vmm.VMIProvider.
func (prov *HarpoonProvider) BootProvisioners() []vmm.BootProvisioner {
	return []vmm.BootProvisioner{}
}

// GuestKernelType implements vmm.VMIProvider.
func (prov *HarpoonProvider) GuestKernelType() guest.GuestKernelType {
	return guest.GuestKernelTypeLinux
}

// RuntimeProvisioners implements vmm.VMIProvider.
func (prov *HarpoonProvider) RuntimeProvisioners() []vmm.RuntimeProvisioner {
	return []vmm.RuntimeProvisioner{}
}

// SupportsEFI implements vmm.VMIProvider.
func (prov *HarpoonProvider) SupportsEFI() bool {
	return false
}

func NewHarpoonProvider() *HarpoonProvider {
	return &HarpoonProvider{}
}

func (prov *HarpoonProvider) Name() string {
	return "puipui"
}

func (prov *HarpoonProvider) Version() string {
	return semver.Canonical(puipuiVersion)
}

func (prov *HarpoonProvider) InitramfsPath() (path string) {
	return extractedCpioName
}

func (prov *HarpoonProvider) KernelPath() (path string) {
	return extractedKernelName
}

func (prov *HarpoonProvider) KernelArgs() (args string) {
	return ""
}

func (prov *HarpoonProvider) RootfsPath() (path string) {
	return "" // no rootfs
}

func (prov *HarpoonProvider) InitScript(ctx context.Context) (string, error) {
	script := `
#!/bin/sh

echo "Hello, world!"
`

	return script, nil
}

func (prov *HarpoonProvider) Downloads() map[string]string {
	return map[string]string{}
}

func (prov *HarpoonProvider) ExtractDownloads(ctx context.Context, cacheDir map[string]io.Reader) (map[string]io.Reader, error) {
	out := make(map[string]io.Reader)

	_, cpioExists := cacheDir[extractedCpioName]
	_, kernelExists := cacheDir[extractedKernelName]

	if cpioExists && kernelExists {
		out[extractedCpioName] = cacheDir[extractedCpioName]
		out[extractedKernelName] = cacheDir[extractedKernelName]
		return out, nil
	}

	r, err := (archives.Xz{}).OpenReader(bytes.NewReader(vmlinux_aarch64.BinaryXZ))
	if err != nil {
		return nil, errors.Errorf("failed to open kernel: %w", err)
	}

	gzr, err := (archives.Gz{}).OpenReader(bytes.NewReader(initramfs_aarch64.BinaryGZ))
	if err != nil {
		return nil, errors.Errorf("failed to open initramfs: %w", err)
	}

	out[extractedCpioName] = gzr
	out[extractedKernelName] = r

	return out, nil
}

func (puipui *HarpoonProvider) ShutdownCommand() string {
	return "poweroff"
}

func (puipui *HarpoonProvider) SSHConfig() *ssh.ClientConfig {
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
