package coreos

import (
	"context"
	"fmt"
	"io"

	"golang.org/x/crypto/ssh"
	"golang.org/x/mod/semver"

	"github.com/coreos/stream-metadata-go/fedoracoreos"
	"github.com/mholt/archives"
	"gitlab.com/tozd/go/errors"

	"github.com/walteh/ec1/pkg/guest"
	"github.com/walteh/ec1/pkg/host"
	"github.com/walteh/ec1/pkg/unzbootgo"
	"github.com/walteh/ec1/pkg/vmm"
)

const coreosVersion = "42.20250427.3.0"
const coreosReleaseStream = fedoracoreos.StreamStable

const (
	coreosKernelExtractPath     = "kernel.coreos-extract"
	coreosInitramfsExtractPath  = "initramfs.coreos-extract.cpio"
	coreosRootfsExtractPath     = "rootfs.coreos-extract.img"
	coreosKernelDownloadPath    = "kernel"
	coreosInitramfsDownloadPath = "initramfs.img"
	coreosRootfsDownloadPath    = "rootfs.img"
)

// stream, err := fedoracoreos.FetchStream(fedoraReleaseStream)
// if err != nil {
// 	return nil, errors.Errorf("fetching stream: %w", err)
// }

// archInfo, err := stream.GetArchitecture(arch)
// if err != nil {
// 	return nil, errors.Errorf("getting architecture info: %w", err)
// }

// root := archInfo.Artifacts["metal"].Formats["pxe"]

func (prov *CoreOSProvider) GuestKernelType() guest.GuestKernelType {
	return guest.GuestKernelTypeLinux
}

var (
	_ vmm.VMIProvider             = &CoreOSProvider{}
	_ vmm.DownloadableVMIProvider = &CoreOSProvider{}
	_ vmm.LinuxVMIProvider        = &CoreOSProvider{}
)

type CoreOSProvider struct {
}

func NewProvider() *CoreOSProvider {
	return &CoreOSProvider{}
}

func (prov *CoreOSProvider) Name() string {
	return "coreos"
}

func (prov *CoreOSProvider) Version() string {
	return semver.Canonical(fmt.Sprintf("v%s", coreosVersion))
}

func (prov *CoreOSProvider) Downloads() map[string]string {

	coreos := `https://builds.coreos.fedoraproject.org/prod/streams/stable/builds/%[1]s/%[2]s/fedora-coreos-%[1]s-live-%[3]s.%[2]s%[4]s`
	// rawFedora := ` https://download.fedoraproject.org/pub/fedora/linux/releases/<release>/Everything/<architecture>/os/images/pxeboot/%[1]`

	arch := host.CurrentKernelArch()

	return map[string]string{
		coreosKernelDownloadPath:             fmt.Sprintf(coreos, coreosVersion, arch, "kernel", ""),
		coreosInitramfsDownloadPath:          fmt.Sprintf(coreos, coreosVersion, arch, "initramfs", ".img"),
		coreosRootfsDownloadPath:             fmt.Sprintf(coreos, coreosVersion, arch, "rootfs", ".img"),
		coreosKernelDownloadPath + ".sig":    fmt.Sprintf(coreos, coreosVersion, arch, "kernel", ".sig"),
		coreosInitramfsDownloadPath + ".sig": fmt.Sprintf(coreos, coreosVersion, arch, "initramfs", ".img.sig"),
		coreosRootfsDownloadPath + ".sig":    fmt.Sprintf(coreos, coreosVersion, arch, "rootfs", ".img.sig"),
		"fedora.gpg":                         "https://fedoraproject.org/fedora.gpg",
	}
}

func (prov *CoreOSProvider) ExtractDownloads(ctx context.Context, mem map[string]io.Reader) (map[string]io.Reader, error) {

	wrk := map[string]io.Reader{}

	if kernel, cached := mem[coreosKernelExtractPath]; cached {
		wrk[coreosKernelExtractPath] = kernel
	} else {
		kernelReader, err := unzbootgo.ProcessKernel(ctx, mem[coreosKernelDownloadPath])
		if err != nil {
			return nil, errors.Errorf("processing kernel: %w", err)
		}
		wrk[coreosKernelExtractPath] = kernelReader
	}

	if initramfs, cached := mem[coreosInitramfsExtractPath]; cached {
		wrk[coreosInitramfsExtractPath] = initramfs
	} else {
		read, err := (archives.Zstd{}).OpenReader(mem[coreosInitramfsDownloadPath])
		if err != nil {
			return nil, errors.Errorf("decompressing initramfs: %w", err)
		}
		wrk[coreosInitramfsExtractPath] = read
	}

	if rootfs, cached := mem[coreosRootfsExtractPath]; cached {
		wrk[coreosRootfsExtractPath] = rootfs
	} else {
		wrk[coreosRootfsExtractPath] = mem[coreosRootfsDownloadPath]
	}

	return wrk, nil
}

func (prov *CoreOSProvider) RootfsPath() (path string) {
	return "rootfs.coreos-extract.img"
}

func (prov *CoreOSProvider) KernelPath() (path string) {
	return "kernel.coreos-extract"
}

func (prov *CoreOSProvider) InitramfsPath() (path string) {
	return "initramfs.coreos-extract.cpio"
}

func (prov *CoreOSProvider) InitScript(ctx context.Context) (string, error) {
	script := `
#!/bin/sh

echo "Hello, world!"
`
	return script, nil
}

func (prov *CoreOSProvider) KernelArgs() (args string) {
	// EXPERIMENTAL: For maximum boot speed, we could use initramfs-only mode
	// return "rdinit=/init rd.driver.blacklist=floppy,pcspkr modules_load=vsock,vmw_vsock_virtio_transport"

	return "coreos.live.rootfs_url=/dev/nvme0n1p1 rd.driver.blacklist=floppy,pcspkr modules_load=vsock,vmw_vsock_virtio_transport"
}

func (prov *CoreOSProvider) BootProvisioners() []vmm.BootProvisioner {
	return []vmm.BootProvisioner{
		// ignition.NewIgnitionBootConfigProvider(cfg),
	}
}

func (fedora *CoreOSProvider) RuntimeProvisioners() []vmm.RuntimeProvisioner {
	return []vmm.RuntimeProvisioner{}
}

func (fedora *CoreOSProvider) ShutdownCommand() string {
	return "sudo shutdown -h now"
}

func (fedora *CoreOSProvider) SSHConfig() *ssh.ClientConfig {
	return &ssh.ClientConfig{
		User: "vfkituser",
		Auth: []ssh.AuthMethod{ssh.Password("vfkittest")},
		// #nosec 106 -- the host SSH key of the VM will change each time it boots
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
}
