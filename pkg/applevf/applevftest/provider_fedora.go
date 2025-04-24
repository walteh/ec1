package applevftest

import (
	"context"
	"fmt"

	"github.com/crc-org/vfkit/pkg/config"
	"gitlab.com/tozd/go/errors"
	"golang.org/x/crypto/ssh"
)

const fedoraVersion = "42"
const fedoraRelease = "1.1"

var _ OsProvider = &FedoraProvider{}

type FedoraProvider struct {
	diskImage            string
	efiVariableStorePath string
	createVariableStore  bool
}

func NewFedoraProvider() *FedoraProvider {
	return &FedoraProvider{}
}

func (prov *FedoraProvider) URL() string {
	arch := kernelArch()
	// https://download.fedoraproject.org/pub/fedora/linux/releases/42/Cloud/aarch64/images/Fedora-Cloud-Base-GCE-42-1.1.aarch64.tar.gz
	buildString := fmt.Sprintf("%s-%s.%s", fedoraVersion, fedoraRelease, arch)
	return fmt.Sprintf("https://download.fedoraproject.org/pub/fedora/linux/releases/%s/Cloud/%s/images/Fedora-Cloud-Base-AmazonEC2-%s.raw.xz", fedoraVersion, arch, buildString)
}

func (prov *FedoraProvider) Initialize(ctx context.Context, cacheDir string) error {
	diskImage, err := findFirstFileWithExtension(cacheDir, ".raw")
	if err != nil {
		return errors.Errorf("could not find disk image: %w", err)
	}
	prov.diskImage = diskImage
	// xzCutName, _ := strings.CutSuffix(filepath.Base(prov.URL()), "tar.gz")
	// prov.efiVariableStorePath = filepath.Join(cacheDir, "efivars.img")
	// prov.createVariableStore = true
	return nil
}

func (fedora *FedoraProvider) ToVirtualMachine() (*config.VirtualMachine, error) {
	bootloader := config.NewEFIBootloader(fedora.efiVariableStorePath, fedora.createVariableStore)
	vm := config.NewVirtualMachine(puipuiCPUs, puipuiMemoryMiB, bootloader)

	return vm, nil
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

func (fedora *FedoraProvider) SSHAccessMethods() []SSHAccessMethod {
	return []SSHAccessMethod{
		{
			network: "tcp",
			port:    22,
		},
		{
			network: "vsock",
			port:    2222,
		},
	}
}
