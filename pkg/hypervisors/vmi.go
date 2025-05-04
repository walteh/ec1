package hypervisors

import (
	"context"

	"github.com/walteh/ec1/pkg/machines/bootloader"
	"github.com/walteh/ec1/pkg/machines/guest"
	"golang.org/x/crypto/ssh"
)

type VMIProvider interface {
	DiskImageURL() string
	BootProvisioners() []BootProvisioner
	RuntimeProvisioners() []RuntimeProvisioner
	SSHConfig() *ssh.ClientConfig
	ShutdownCommand() string
	Name() string
	Version() string
	SupportsEFI() bool
	GuestKernelType() guest.GuestKernelType
}

type CustomExtractorVMIProvider interface {
	CustomExtraction(ctx context.Context, cacheDir string) error
}

type DiskImageRawFileNameVMIProvider interface {
	DiskImageRawFileName() string
}

type MacOSVMIProvider interface {
	BootLoaderConfig() *bootloader.MacOSBootloader
}

type LinuxVMIProvider interface {
	BootLoaderConfig(cacheDir string) *bootloader.LinuxBootloader
}
