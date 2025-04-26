package applevftest

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/crc-org/vfkit/pkg/config"
	"github.com/walteh/ec1/pkg/applevf"
	"gitlab.com/tozd/go/errors"
	"golang.org/x/crypto/ssh"
	"golang.org/x/mod/semver"
)

const fedoraVersion = "42"
const fedoraRelease = "1.1"

var _ OsProvider = &FedoraProvider{}

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
	return "fedora-ec2"
}

func (prov *FedoraProvider) Version() string {
	return semver.Canonical(fmt.Sprintf("v%s.%s", fedoraVersion, fedoraRelease))
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
	prov.efiVariableStorePath = filepath.Join(cacheDir, "efi-variable-store")
	prov.createVariableStore = true
	prov.socketPath = filepath.Join(cacheDir, "vf.sock")
	return nil
}

func (fedora *FedoraProvider) ToVirtualMachine(ctx context.Context) (*config.VirtualMachine, error) {
	bootloader := config.NewEFIBootloader(fedora.efiVariableStorePath, fedora.createVariableStore)

	vm := config.NewVirtualMachine(puipuiCPUs, puipuiMemoryMiB, bootloader)

	cloudInitFiles := applevf.CloudInitFiles{
		UserData: `#cloud-config
users:
  - name: vfkituser
    sudo: ALL=(ALL) NOPASSWD:ALL
    shell: /bin/bash
    groups: users
    plain_text_passwd: vfkittest
    lock_passwd: false
ssh_pwauth: true
chpasswd: { expire: false }`,
		MetaData: "",
	}

	cloudInitISO, err := cloudInitFiles.GenerateISO(ctx)
	if err != nil {
		return nil, err
	}

	virtioBlkDevices := []string{
		fedora.diskImage,
		cloudInitISO,
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
