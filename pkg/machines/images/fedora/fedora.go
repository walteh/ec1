package fedora

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/walteh/ec1/pkg/hypervisors/vf/config"
	"github.com/walteh/ec1/pkg/machines"
	"github.com/walteh/ec1/pkg/machines/host"
	"gitlab.com/tozd/go/errors"
	"golang.org/x/crypto/ssh"
	"golang.org/x/mod/semver"
)

const fedoraVersion = "42"
const fedoraRelease = "1.1"

const fedoraCPUs = 2
const fedoraMemoryMiB = 2048

var _ machines.OsProvider = &FedoraProvider{}

type FedoraProvider struct {
	diskImage            string
	efiVariableStorePath string
	createVariableStore  bool
	socketPath           string
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

func (prov *FedoraProvider) URL() string {
	arch := host.CurrentKernelArch()
	// GCE doesn't work https://download.fedoraproject.org/pub/fedora/linux/releases/42/Cloud/aarch64/images/Fedora-Cloud-Base-GCE-42-1.1.aarch64.tar.gz
	buildString := fmt.Sprintf("%s-%s.%s", fedoraVersion, fedoraRelease, arch)
	return fmt.Sprintf("https://download.fedoraproject.org/pub/fedora/linux/releases/%s/Cloud/%s/images/Fedora-Cloud-Base-AmazonEC2-%s.raw.xz", fedoraVersion, arch, buildString)
}

func (prov *FedoraProvider) Initialize(ctx context.Context, cacheDir string) error {
	diskImage, err := host.FindFirstFileWithExtension(cacheDir, ".raw")
	if err != nil {
		return errors.Errorf("could not find disk image: %w", err)
	}

	prov.diskImage = diskImage
	prov.efiVariableStorePath = filepath.Join(cacheDir, "efi-variable-store")
	prov.createVariableStore = true
	prov.socketPath = filepath.Join(cacheDir, "vf.sock")
	return nil
}

func (fedora *FedoraProvider) ToVirtualMachine(ctx context.Context) (*config.VirtualMachine, error) {
	bootloader := config.NewEFIBootloader(fedora.efiVariableStorePath, fedora.createVariableStore)

	vm := config.NewVirtualMachine(fedoraCPUs, fedoraMemoryMiB, bootloader)

	virtioBlkDevices := []string{
		fedora.diskImage,
	}

	for _, diskImage := range virtioBlkDevices {
		dev, err := config.VirtioBlkNew(diskImage)
		if err != nil {
			return nil, errors.Errorf("creating virtio-blk device %s: %w", filepath.Base(diskImage), err)
		}
		err = vm.AddDevice(dev)
		if err != nil {
			return nil, errors.Errorf("adding virtio-blk device %s: %w", filepath.Base(diskImage), err)
		}
		slog.InfoContext(ctx, "shared disk", "name", dev.DevName)
	}

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
