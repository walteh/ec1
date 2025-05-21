package ubuntuminimal

import (
	"context"
	"fmt"
	"io"

	"golang.org/x/crypto/ssh"
	"golang.org/x/mod/semver"

	"gitlab.com/tozd/go/errors"

	"github.com/walteh/ec1/pkg/guest"
	"github.com/walteh/ec1/pkg/host"
	"github.com/walteh/ec1/pkg/unzbootgo"
	"github.com/walteh/ec1/pkg/vmm"
)

const ubuntuVersion = "24.10" // Ubuntu Oracular Oriole
const ubuntuCodename = "oracular"
const ubuntuRelease = "20250429" // Release date, adjust as needed

func (prov *UbuntuMinimalProvider) GuestKernelType() guest.GuestKernelType {
	return guest.GuestKernelTypeLinux
}

var (
	_ vmm.VMIProvider             = &UbuntuMinimalProvider{}
	_ vmm.DownloadableVMIProvider = &UbuntuMinimalProvider{}
	_ vmm.LinuxVMIProvider        = &UbuntuMinimalProvider{}
)

type UbuntuMinimalProvider struct {
}

func NewProvider() *UbuntuMinimalProvider {
	return &UbuntuMinimalProvider{}
}

func (prov *UbuntuMinimalProvider) Name() string {
	return "ubuntu"
}

func (prov *UbuntuMinimalProvider) Version() string {
	return semver.Canonical(fmt.Sprintf("v%s", ubuntuVersion))
}

func (prov *UbuntuMinimalProvider) Downloads() map[string]string {
	arch := host.CurrentKernelArch()

	// Convert aarch64 to arm64 for Ubuntu naming convention
	ubuntuArch := arch
	if arch == "aarch64" {
		ubuntuArch = "arm64"
	} else if arch == "x86_64" {
		ubuntuArch = "amd64"
	}

	// Base URL for Ubuntu cloud images
	baseURL := fmt.Sprintf("https://cloud-images.ubuntu.com/minimal/releases/%s/release-%s/",
		ubuntuCodename, ubuntuRelease)

	return map[string]string{
		// Kernel image
		"vmlinuz": fmt.Sprintf("%sunpacked/ubuntu-%s-minimal-cloudimg-%s-vmlinuz-generic",
			baseURL, ubuntuVersion, ubuntuArch),

		// Root filesystem archive - better than raw image for our purposes
		"squashfs": fmt.Sprintf("%subuntu-%s-minimal-cloudimg-%s.squashfs",
			baseURL, ubuntuVersion, ubuntuArch),

		// GPG signature for verification
		"SHA256SUMS.gpg": fmt.Sprintf("%sunpacked/SHA256SUMS.gpg", baseURL),
	}
}

func (prov *UbuntuMinimalProvider) ExtractDownloads(ctx context.Context, cacheDir map[string]io.Reader) (map[string]io.ReadCloser, error) {

	// Extract the kernel if it's an EFI application
	kernelReader, err := unzbootgo.ProcessKernel(ctx, cacheDir["vmlinuz"])
	if err != nil {
		return nil, errors.Errorf("processing kernel: %w", err)
	}

	out := make(map[string]io.ReadCloser)
	out["vmlinuz"] = kernelReader

	return out, nil
}

func (prov *UbuntuMinimalProvider) InitScript(ctx context.Context) (string, error) {
	script := `
#!/bin/sh

echo "Hello from Ubuntu!"
# Ensure vsock module is loaded
modprobe vmw_vsock_virtio_transport
`

	return script, nil
}

func (prov *UbuntuMinimalProvider) RootfsPath() (path string) {
	return "squashfs"
}

func (prov *UbuntuMinimalProvider) KernelPath() (path string) {
	return "vmlinuz"
}

func (prov *UbuntuMinimalProvider) InitramfsPath() (path string) {
	// Ubuntu cloud images do use a separate initramfs
	return ""
}

func (prov *UbuntuMinimalProvider) KernelArgs() (args string) {
	// Kernel args for Ubuntu with vsock support
	// Explicitly specify the root device properly
	return ""
}

func (prov *UbuntuMinimalProvider) BootProvisioners() []vmm.BootProvisioner {
	return []vmm.BootProvisioner{
		// Add any Ubuntu-specific boot provisioners here
	}
}

func (prov *UbuntuMinimalProvider) RuntimeProvisioners() []vmm.RuntimeProvisioner {
	return []vmm.RuntimeProvisioner{}
}

func (prov *UbuntuMinimalProvider) ShutdownCommand() string {
	return "sudo shutdown -h now"
}

func (prov *UbuntuMinimalProvider) SSHConfig() *ssh.ClientConfig {
	return &ssh.ClientConfig{
		User: "ubuntu",                                 // Default user for Ubuntu cloud images
		Auth: []ssh.AuthMethod{ssh.Password("ubuntu")}, // Default password is usually "ubuntu"
		// #nosec 106 -- the host SSH key of the VM will change each time it boots
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
}
