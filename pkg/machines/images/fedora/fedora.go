package fedora

import (
	"fmt"
	"path/filepath"
	"strings"

	types_exp "github.com/coreos/ignition/v2/config/v3_6_experimental/types"
	"github.com/walteh/ec1/pkg/hypervisors"
	"github.com/walteh/ec1/pkg/machines/guest"
	"github.com/walteh/ec1/pkg/machines/host"
	"github.com/walteh/ec1/pkg/machines/provisioner/ignition"
	"golang.org/x/crypto/ssh"
	"golang.org/x/mod/semver"
)

const fedoraVersion = "42"
const fedoraRelease = "1.1"

func (prov *FedoraProvider) SupportsEFI() bool {
	return true
}

func (prov *FedoraProvider) GuestKernelType() guest.GuestKernelType {
	return guest.GuestKernelTypeLinux
}

var (
	_ hypervisors.VMIProvider                     = &FedoraProvider{}
	_ hypervisors.DiskImageRawFileNameVMIProvider = &FedoraProvider{}
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
	return semver.Canonical(fmt.Sprintf("v%s.%s", fedoraVersion, fedoraRelease))
}

func (prov *FedoraProvider) DiskImageURL() string {
	arch := host.CurrentKernelArch()
	// GCE doesn't work https://download.fedoraproject.org/pub/fedora/linux/releases/42/Cloud/aarch64/images/Fedora-Cloud-Base-GCE-42-1.1.aarch64.tar.gz
	buildString := fmt.Sprintf("%s-%s.%s", fedoraVersion, fedoraRelease, arch)
	return fmt.Sprintf("https://download.fedoraproject.org/pub/fedora/linux/releases/%s/Cloud/%s/images/Fedora-Cloud-Base-AmazonEC2-%s.raw.xz", fedoraVersion, arch, buildString)
}

func (prov *FedoraProvider) DiskImageRawFileName() string {
	diskImageURL := prov.DiskImageURL()
	return strings.TrimSuffix(filepath.Base(diskImageURL), filepath.Ext(diskImageURL))
}

func (prov *FedoraProvider) BootProvisioners() []hypervisors.BootProvisioner {
	cfg := &types_exp.Config{}
	return []hypervisors.BootProvisioner{ignition.NewIgnitionBootConfigProvider(cfg)}
}

func (fedora *FedoraProvider) RuntimeProvisioners() []hypervisors.RuntimeProvisioner {
	return []hypervisors.RuntimeProvisioner{}
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
