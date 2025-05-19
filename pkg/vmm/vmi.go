package vmm

import (
	"context"
	"io"

	"golang.org/x/crypto/ssh"

	"github.com/walteh/ec1/pkg/bootloader"
	"github.com/walteh/ec1/pkg/guest"
)

type VMIProvider interface {
	BootProvisioners() []BootProvisioner
	RuntimeProvisioners() []RuntimeProvisioner
	SSHConfig() *ssh.ClientConfig
	ShutdownCommand() string
	Name() string
	Version() string
	// SupportsEFI() bool
	GuestKernelType() guest.GuestKernelType
	InitScript(ctx context.Context) (string, error)
}

type DownloadableVMIProvider interface {
	Downloads() map[string]string
	ExtractDownloads(ctx context.Context, cacheDir map[string]io.Reader) (map[string]io.Reader, error)
}

type RootFSProvider interface {
	RelativeRootFSPath() string
}

type MacOSVMIProvider interface {
	BootLoaderConfig() *bootloader.MacOSBootloader
}

type LinuxVMIProvider interface {
	RootfsPath() (path string)
	KernelPath() (path string)
	InitramfsPath() (path string)
	KernelArgs() (args string)
}
