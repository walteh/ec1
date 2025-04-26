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

const alpineVersion = "3.21"
const alpineRelease = "2"

var _ OsProvider = &AlpineProvider{}

type AlpineProvider struct {
	diskImage            string
	efiVariableStorePath string
	createVariableStore  bool
	socketPath           string
}

func NewAlpineProvider() *AlpineProvider {
	return &AlpineProvider{}
}

func (prov *AlpineProvider) Name() string {
	return "alpine"
}

func (prov *AlpineProvider) Version() string {
	return semver.Canonical(fmt.Sprintf("v%s.%s", alpineVersion, alpineRelease))
}

func (prov *AlpineProvider) URL() string {
	arch := kernelArch()
	// GCE doesn't work
	// https://download.alpineproject.org/pub/alpine/linux/releases/42/Cloud/aarch64/images/Alpine-Cloud-Base-GCE-42-1.1.aarch64.tar.gz
	return fmt.Sprintf("https://dl-cdn.alpinelinux.org/alpine/v%[1]s/releases/cloud/nocloud_alpine-%[1]s.%[2]s-%[3]s-uefi-cloudinit-r0.qcow2", alpineVersion, alpineRelease, arch)
}

func (prov *AlpineProvider) Initialize(ctx context.Context, cacheDir string) error {

	diskImage, err := findFirstFileWithExtension(cacheDir, ".qcow2")
	if err != nil {
		return errors.Errorf("could not find disk image: %w", err)
	}

	diskImage, err = convertFileToRaw(ctx, diskImage)
	if err != nil {
		return errors.Errorf("could not convert disk image: %w", err)
	}
	prov.diskImage = diskImage
	prov.efiVariableStorePath = filepath.Join(cacheDir, "efi-variable-store")
	prov.createVariableStore = true
	prov.socketPath = filepath.Join(cacheDir, "vf.sock")
	return nil
}

func (alpine *AlpineProvider) ToVirtualMachine(ctx context.Context) (*config.VirtualMachine, error) {
	bootloader := config.NewEFIBootloader(alpine.efiVariableStorePath, alpine.createVariableStore)

	vm := config.NewVirtualMachine(puipuiCPUs, puipuiMemoryMiB, bootloader)

	cloudInitFiles := applevf.CloudInitFiles{
		UserData: `#cloud-config
users:
  - name: vfkituser
    sudo: ALL=(ALL) NOPASSWD:ALL
    shell: /bin/sh
    groups: users, wheel
    plain_text_passwd: vfkittest
    lock_passwd: false
ssh_pwauth: true
chpasswd: { expire: false }
packages:
  - doas
write_files:
  - path: /etc/doas.conf
    content: |
      permit nopass :wheel
    permissions: '0400'
    owner: root:root
`,
		MetaData: "",
	}

	cloudInitISO, err := cloudInitFiles.GenerateISO(ctx)
	if err != nil {
		return nil, err
	}

	virtioBlkDevices := []string{
		alpine.diskImage,
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

func (alpine *AlpineProvider) ShutdownCommand() string {
	return "doas poweroff"
}

func (alpine *AlpineProvider) SSHConfig() *ssh.ClientConfig {
	return &ssh.ClientConfig{
		User: "vfkituser",
		Auth: []ssh.AuthMethod{ssh.Password("vfkittest")},
		// #nosec 106 -- the host SSH key of the VM will change each time it boots
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

}
