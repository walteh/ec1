package flatcar

import (
	"fmt"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/ssh"
	"golang.org/x/mod/semver"

	"github.com/walteh/ec1/pkg/machines/guest"
	"github.com/walteh/ec1/pkg/machines/host"
	"github.com/walteh/ec1/pkg/vmm"
)

const flatcarVersion = "4230.1.1"
const flatcarChannel = "beta"

func (prov *FlatcarProvider) SupportsEFI() bool {
	return true
}

func (prov *FlatcarProvider) GuestKernelType() guest.GuestKernelType {
	return guest.GuestKernelTypeLinux
}

var (
	_ vmm.VMIProvider                     = &FlatcarProvider{}
	_ vmm.DiskImageRawFileNameVMIProvider = &FlatcarProvider{}
)

type FlatcarProvider struct {
}

func NewFlatcarProvider() *FlatcarProvider {
	return &FlatcarProvider{}
}

func (prov *FlatcarProvider) Name() string {
	return "flatcar"
}

func (prov *FlatcarProvider) Version() string {
	return semver.Canonical(fmt.Sprintf("v%s", flatcarVersion))
}

func (prov *FlatcarProvider) DiskImageURL() string {
	arch := host.CurrentKernelArch()

	// Flatcar uses different directory structures for different architectures
	var archDir string
	if arch == "aarch64" {
		archDir = "arm64-usr"
	} else {
		archDir = "amd64-usr"
	}

	return fmt.Sprintf("https://%s.release.flatcar-linux.net/%s/%s/flatcar_production_image.bin.bz2", flatcarChannel, archDir, flatcarVersion)
}

func (prov *FlatcarProvider) DiskImageRawFileName() string {
	diskImageURL := prov.DiskImageURL()
	return strings.TrimSuffix(filepath.Base(diskImageURL), filepath.Ext(diskImageURL))
}

func (prov *FlatcarProvider) BootProvisioners() []vmm.BootProvisioner {
	return []vmm.BootProvisioner{}
}

func (prov *FlatcarProvider) RuntimeProvisioners() []vmm.RuntimeProvisioner {
	return []vmm.RuntimeProvisioner{}
}

func (prov *FlatcarProvider) ShutdownCommand() string {
	return "sudo systemctl poweroff"
}

func (prov *FlatcarProvider) SSHConfig() *ssh.ClientConfig {
	return &ssh.ClientConfig{
		User: "core",
		Auth: []ssh.AuthMethod{ssh.Password("flatcar")},
		// #nosec 106 -- the host SSH key of the VM will change each time it boots
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
}
