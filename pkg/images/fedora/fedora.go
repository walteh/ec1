package fedora

import (
	"context"
	"fmt"
	"io"

	"golang.org/x/crypto/ssh"
	"golang.org/x/mod/semver"

	"github.com/coreos/stream-metadata-go/fedoracoreos"

	types_exp "github.com/coreos/ignition/v2/config/v3_6_experimental/types"

	"github.com/walteh/ec1/pkg/guest"
	"github.com/walteh/ec1/pkg/host"
	"github.com/walteh/ec1/pkg/provisioner/ignition"
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

	initrd := `https://builds.coreos.fedoraproject.org/prod/streams/stable/builds/%[1]s/%[2]s/fedora-coreos-%[1]s-live-%[3]s.%[2]s%[4]s`

	arch := host.CurrentKernelArch()

	return map[string]string{
		"kernel":            fmt.Sprintf(initrd, fedoraVersion, arch, "kernel", ""),
		"initramfs.img":     fmt.Sprintf(initrd, fedoraVersion, arch, "initramfs", ".img"),
		"rootfs.img":        fmt.Sprintf(initrd, fedoraVersion, arch, "rootfs", ".img"),
		"kernel.sig":        fmt.Sprintf(initrd, fedoraVersion, arch, "kernel", ".sig"),
		"initramfs.img.sig": fmt.Sprintf(initrd, fedoraVersion, arch, "initramfs", ".img.sig"),
		"rootfs.img.sig":    fmt.Sprintf(initrd, fedoraVersion, arch, "rootfs", ".img.sig"),
		"fedora.gpg":        "https://fedoraproject.org/fedora.gpg",
	}
}

func (prov *FedoraProvider) ExtractDownloads(ctx context.Context, cacheDir map[string]io.Reader) (map[string]io.Reader, error) {

	// initramfs, ok := cacheDir["initramfs.img"]
	// if !ok {
	// 	return nil, errors.New("initramfs.img not found")
	// }

	// // extract the image
	// diskf, err :=

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
	return ""
}

func (prov *FedoraProvider) BootProvisioners() []vmm.BootProvisioner {
	cfg := &types_exp.Config{}
	return []vmm.BootProvisioner{ignition.NewIgnitionBootConfigProvider(cfg)}
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
