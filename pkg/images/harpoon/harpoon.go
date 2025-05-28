package harpoon

import (
	"bytes"
	"context"
	"io"

	"golang.org/x/crypto/ssh"
	"golang.org/x/mod/semver"

	"github.com/mholt/archives"
	"gitlab.com/tozd/go/errors"

	"github.com/walteh/ec1/gen/harpoon/harpoon_initramfs_arm64"
	"github.com/walteh/ec1/gen/harpoon/harpoon_vmlinux_arm64"
	"github.com/walteh/ec1/pkg/guest"
	"github.com/walteh/ec1/pkg/vmm"
)

const puipuiVersion = "v0.0.1"

const (
	extractedCpioName   = "initramfs.ec1-extract." + harpoon_initramfs_arm64.BinaryXZChecksum + ".cpio"
	extractedKernelName = "vmlinux.ec1-extract." + harpoon_vmlinux_arm64.BinaryXZChecksum
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
	return "harpoon"
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

	r, err := (archives.Xz{}).OpenReader(bytes.NewReader(harpoon_vmlinux_arm64.BinaryXZ))
	if err != nil {
		return nil, errors.Errorf("failed to open kernel: %w", err)
	}

	gzr, err := (archives.Xz{}).OpenReader(bytes.NewReader(harpoon_initramfs_arm64.BinaryXZ))
	if err != nil {
		return nil, errors.Errorf("failed to open initramfs: %w", err)
	}

	gzr, err = (archives.Gz{}).OpenReader(gzr)
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
