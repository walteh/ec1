package ubuntu

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"golang.org/x/crypto/ssh"
	"golang.org/x/mod/semver"

	"github.com/walteh/ec1/pkg/guest"
	"github.com/walteh/ec1/pkg/host"
	"github.com/walteh/ec1/pkg/unzbootgo"
	"github.com/walteh/ec1/pkg/vmm"
)

const ubuntuVersion = "24.10" // Ubuntu Oracular Oriole
const ubuntuCodename = "oracular"
const ubuntuRelease = "20250429" // Release date, adjust as needed

func (prov *UbuntuProvider) GuestKernelType() guest.GuestKernelType {
	return guest.GuestKernelTypeLinux
}

var (
	_ vmm.VMIProvider             = &UbuntuProvider{}
	_ vmm.DownloadableVMIProvider = &UbuntuProvider{}
	_ vmm.LinuxVMIProvider        = &UbuntuProvider{}
)

type UbuntuProvider struct {
}

func NewProvider() *UbuntuProvider {
	return &UbuntuProvider{}
}

func (prov *UbuntuProvider) Name() string {
	return "ubuntu"
}

func (prov *UbuntuProvider) Version() string {
	return semver.Canonical(fmt.Sprintf("v%s", ubuntuVersion))
}

func (prov *UbuntuProvider) Downloads() map[string]string {
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

func (prov *UbuntuProvider) ExtractDownloads(ctx context.Context, cacheDir map[string]io.Reader) (map[string]io.Reader, error) {

	// Extract the kernel if it's an EFI application
	kernelReader, err := unzbootgo.ProcessKernel(ctx, cacheDir["vmlinuz"])
	if err != nil {
		slog.Error("failed to process kernel", "error", err)
		// Not an EFI application or extraction failed, return the original
		return cacheDir, nil
	}

	slog.Info("kernel taken care of")

	cacheDir["vmlinuz"] = kernelReader

	// tmpOutFile, err := os.CreateTemp("", "rootfs.img")
	// if err != nil {
	// 	slog.Error("failed to create temp file", "error", err)
	// 	return cacheDir, nil
	// }
	// // defer os.Remove(tmpOutFile.Name())

	// rootfsReader := cacheDir["rootfs.img"]
	// if _, ok := rootfsReader.(io.ReaderAt); !ok {

	// 	tmpFile, err := os.CreateTemp("", "rootfs.img")
	// 	if err != nil {
	// 		slog.Error("failed to create temp file", "error", err)
	// 		return cacheDir, nil
	// 	}
	// 	defer func() {
	// 		tmpFile.Close()
	// 		os.Remove(tmpFile.Name())
	// 	}()
	// 	// write to the temp file
	// 	_, err = io.Copy(tmpFile, rootfsReader)
	// 	if err != nil {
	// 		slog.Error("failed to copy rootfs to temp file", "error", err)
	// 		return cacheDir, nil
	// 	}
	// 	rootfsReader = tmpFile
	// }
	// // defer os.Remove(tmpFile.Name())

	// err = host.ConvertQcow2ToRaw(ctx, rootfsReader.(io.ReaderAt), tmpOutFile)
	// if err != nil {
	// 	slog.Error("failed to convert rootfs to raw", "error", err)
	// 	return cacheDir, nil
	// }

	// cacheDir["rootfs.img"] = tmpOutFile
	// TODO: add the kernel args
	return cacheDir, nil
}

func (prov *UbuntuProvider) InitScript(ctx context.Context) (string, error) {
	script := `
#!/bin/sh

echo "Hello from Ubuntu!"
# Ensure vsock module is loaded
modprobe vmw_vsock_virtio_transport
`

	return script, nil
}

func (prov *UbuntuProvider) RootfsPath() (path string) {
	return "squashfs"
}

func (prov *UbuntuProvider) KernelPath() (path string) {
	return "vmlinuz"
}

func (prov *UbuntuProvider) InitramfsPath() (path string) {
	// Ubuntu cloud images do use a separate initramfs
	return ""
}

func (prov *UbuntuProvider) KernelArgs() (args string) {
	// Kernel args for Ubuntu with vsock support
	// Explicitly specify the root device properly
	return ""
}

func (prov *UbuntuProvider) BootProvisioners() []vmm.BootProvisioner {
	return []vmm.BootProvisioner{
		// Add any Ubuntu-specific boot provisioners here
	}
}

func (prov *UbuntuProvider) RuntimeProvisioners() []vmm.RuntimeProvisioner {
	return []vmm.RuntimeProvisioner{}
}

func (prov *UbuntuProvider) ShutdownCommand() string {
	return "sudo shutdown -h now"
}

func (prov *UbuntuProvider) SSHConfig() *ssh.ClientConfig {
	return &ssh.ClientConfig{
		User: "ubuntu",                                 // Default user for Ubuntu cloud images
		Auth: []ssh.AuthMethod{ssh.Password("ubuntu")}, // Default password is usually "ubuntu"
		// #nosec 106 -- the host SSH key of the VM will change each time it boots
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
}
