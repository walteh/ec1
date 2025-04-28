package hypervisors

import (
	"context"

	"github.com/rs/xid"
	"github.com/walteh/ec1/pkg/hypervisors/vf/config"
	"github.com/walteh/ec1/pkg/machines/guest"
	"github.com/walteh/ec1/pkg/machines/host"
	"golang.org/x/crypto/ssh"
)

type VMIProvider interface {
	DiskImageURL() string
	BootProvisioners() []BootProvisioner
	RuntimeProvisioners() []RuntimeProvisioner
	DiskImageRawFileName() string
	SSHConfig() *ssh.ClientConfig
	ShutdownCommand() string
	Name() string
	Version() string
	SupportsEFI() bool
	GuestKernelType() guest.GuestKernelType
}

type MacOSVMIProvider interface {
	VMIProvider
	BootLoaderConfig() *config.MacOSBootloader
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
