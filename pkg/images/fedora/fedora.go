package fedora

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"golang.org/x/crypto/ssh"
	"golang.org/x/mod/semver"

	"github.com/coreos/stream-metadata-go/fedoracoreos"

	"github.com/walteh/ec1/pkg/guest"
	"github.com/walteh/ec1/pkg/host"
	"github.com/walteh/ec1/pkg/unzbootgo"
	"github.com/walteh/ec1/pkg/vmm"
)

const fedoraVersion = "42.20250427.3.0"
const fedoraReleaseStream = fedoracoreos.StreamStable

// stream, err := fedoracoreos.FetchStream(fedoraReleaseStream)
// if err != nil {
// 	return nil, errors.Errorf("fetching stream: %w", err)
// }

// archInfo, err := stream.GetArchitecture(arch)
// if err != nil {
// 	return nil, errors.Errorf("getting architecture info: %w", err)
// }

// root := archInfo.Artifacts["metal"].Formats["pxe"]

func (prov *FedoraProvider) SupportsEFI() bool {
	return true
}

func (prov *FedoraProvider) GuestKernelType() guest.GuestKernelType {
	return guest.GuestKernelTypeLinux
}

var (
	_ vmm.VMIProvider             = &FedoraProvider{}
	_ vmm.DownloadableVMIProvider = &FedoraProvider{}
	_ vmm.LinuxVMIProvider        = &FedoraProvider{}
)

type FedoraProvider struct {
}

func NewFedoraProvider() *FedoraProvider {
	return &FedoraProvider{}
}

func (prov *FedoraProvider) Name() string {
	return "fedora"
}

func (prov *FedoraProvider) Version() string {
	return semver.Canonical(fmt.Sprintf("v%s", fedoraVersion))
}

func (prov *FedoraProvider) Downloads() map[string]string {

	coreos := `https://builds.coreos.fedoraproject.org/prod/streams/stable/builds/%[1]s/%[2]s/fedora-coreos-%[1]s-live-%[3]s.%[2]s%[4]s`
	// rawFedora := ` https://download.fedoraproject.org/pub/fedora/linux/releases/<release>/Everything/<architecture>/os/images/pxeboot/%[1]`

	arch := host.CurrentKernelArch()

	return map[string]string{
		"kernel":            fmt.Sprintf(coreos, fedoraVersion, arch, "kernel", ""),
		"initramfs.img":     fmt.Sprintf(coreos, fedoraVersion, arch, "initramfs", ".img"),
		"rootfs.img":        fmt.Sprintf(coreos, fedoraVersion, arch, "rootfs", ".img"),
		"kernel.sig":        fmt.Sprintf(coreos, fedoraVersion, arch, "kernel", ".sig"),
		"initramfs.img.sig": fmt.Sprintf(coreos, fedoraVersion, arch, "initramfs", ".img.sig"),
		"rootfs.img.sig":    fmt.Sprintf(coreos, fedoraVersion, arch, "rootfs", ".img.sig"),
		"fedora.gpg":        "https://fedoraproject.org/fedora.gpg",
	}
}

func (prov *FedoraProvider) ExtractDownloads(ctx context.Context, cacheDir map[string]io.Reader) (map[string]io.Reader, error) {
	// Extract the kernel if it's an EFI application
	kernelReader, err := unzbootgo.ProcessKernel(ctx, cacheDir["kernel"])
	if err != nil {
		slog.Error("failed to process kernel", "error", err)
		// Not an EFI application or extraction failed, return the original
		return cacheDir, nil
	}

	cacheDir["kernel"] = kernelReader

	return cacheDir, nil
}

func (prov *FedoraProvider) InitScript(ctx context.Context) (string, error) {
	script := `
#!/bin/sh

echo "Hello, world!"
`

	return script, nil
}

func (prov *FedoraProvider) RootfsPath() (path string) {
	return "rootfs.img"
}

func (prov *FedoraProvider) KernelPath() (path string) {
	return "kernel"
}

func (prov *FedoraProvider) InitramfsPath() (path string) {
	return "initramfs.img"
}

func (prov *FedoraProvider) KernelArgs() (args string) {
	return "coreos.live.rootfs_url=/dev/nvme0n1p1"
}

func (prov *FedoraProvider) BootProvisioners() []vmm.BootProvisioner {
	return []vmm.BootProvisioner{
		// ignition.NewIgnitionBootConfigProvider(cfg),
	}
}

func (fedora *FedoraProvider) RuntimeProvisioners() []vmm.RuntimeProvisioner {
	return []vmm.RuntimeProvisioner{}
}

func (fedora *FedoraProvider) ShutdownCommand() string {
	return "sudo shutdown -h now"
}

func (fedora *FedoraProvider) SSHConfig() *ssh.ClientConfig {
	return &ssh.ClientConfig{
		User: "vfkituser",
		Auth: []ssh.AuthMethod{ssh.Password("vfkittest")},
		// #nosec 106 -- the host SSH key of the VM will change each time it boots
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
}
