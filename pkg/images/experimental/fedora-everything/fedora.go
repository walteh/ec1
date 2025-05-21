package fedoraeverything

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

const fedoraVersion = "42"

func (prov *FedoraEverythingProvider) GuestKernelType() guest.GuestKernelType {
	return guest.GuestKernelTypeLinux
}

var (
	_ vmm.VMIProvider             = &FedoraEverythingProvider{}
	_ vmm.DownloadableVMIProvider = &FedoraEverythingProvider{}
	_ vmm.LinuxVMIProvider        = &FedoraEverythingProvider{}
)

type FedoraEverythingProvider struct {
}

func NewProvider() *FedoraEverythingProvider {
	return &FedoraEverythingProvider{}
}

func (prov *FedoraEverythingProvider) Name() string {
	return "fedora"
}

func (prov *FedoraEverythingProvider) Version() string {
	return semver.Canonical(fmt.Sprintf("v%s", fedoraVersion))
}

func (prov *FedoraEverythingProvider) Downloads() map[string]string {

	rawFedora := `https://download.fedoraproject.org/pub/fedora/linux/releases/%[1]s/Everything/%[2]s/os/images/pxeboot/%[3]s%[4]s`

	arch := host.CurrentKernelArch()

	return map[string]string{
		"vmlinuz":    fmt.Sprintf(rawFedora, fedoraVersion, arch, "vmlinuz", ""),
		"initrd.img": fmt.Sprintf(rawFedora, fedoraVersion, arch, "initrd", ".img"),
		"fedora.gpg": "https://fedoraproject.org/fedora.gpg",
	}
}

func (prov *FedoraEverythingProvider) ExtractDownloads(ctx context.Context, cacheDir map[string]io.Reader) (map[string]io.Reader, error) {
	// Extract the kernel if it's an EFI application
	kernelReader, err := unzbootgo.ProcessKernel(ctx, cacheDir["vmlinuz"])
	if err != nil {
		slog.Error("failed to process kernel", "error", err)
		// Not an EFI application or extraction failed, return the original
		return cacheDir, nil
	}

	cacheDir["vmlinuz"] = kernelReader

	return cacheDir, nil
}

func (prov *FedoraEverythingProvider) InitScript(ctx context.Context) (string, error) {
	script := `
#!/bin/sh

echo "Hello, world!"
`

	return script, nil
}

func (prov *FedoraEverythingProvider) RootfsPath() (path string) {
	return ""
}

func (prov *FedoraEverythingProvider) KernelPath() (path string) {
	return "vmlinuz"
}

func (prov *FedoraEverythingProvider) InitramfsPath() (path string) {
	return "initrd.img"
}

func (prov *FedoraEverythingProvider) KernelArgs() (args string) {
	return "vsock.ko virtio_vsock.ko modprobe.blacklist=floppy"
}

func (prov *FedoraEverythingProvider) BootProvisioners() []vmm.BootProvisioner {
	return []vmm.BootProvisioner{
		// ignition.NewIgnitionBootConfigProvider(cfg),
	}
}

func (fedora *FedoraEverythingProvider) RuntimeProvisioners() []vmm.RuntimeProvisioner {
	return []vmm.RuntimeProvisioner{}
}

func (fedora *FedoraEverythingProvider) ShutdownCommand() string {
	return "sudo shutdown -h now"
}

func (fedora *FedoraEverythingProvider) SSHConfig() *ssh.ClientConfig {
	return &ssh.ClientConfig{
		User: "vfkituser",
		Auth: []ssh.AuthMethod{ssh.Password("vfkittest")},
		// #nosec 106 -- the host SSH key of the VM will change each time it boots
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
}
