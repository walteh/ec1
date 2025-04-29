package hypervisors

import (
	"context"

	"github.com/rs/xid"
	"github.com/walteh/ec1/pkg/machines/bootloader"
	"github.com/walteh/ec1/pkg/machines/guest"
	"github.com/walteh/ec1/pkg/machines/host"
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

func RunVMI(ctx context.Context, vmi VMIProvider) error {
	id := "vm-" + xid.New().String()

	workingDir, err := host.EmphiricalVMCacheDir(ctx, id)
	if err != nil {
		return err
	}

	err = host.DownloadAndExtractVMI(ctx, vmi.DiskImageURL(), workingDir)
	if err != nil {
		return err
	}

	return nil
}
