package coreos

import (
	"context"
	"fmt"
	"io"
	"log/slog"

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
		"kernel":            fmt.Sprintf(coreos, coreosVersion, arch, "kernel", ""),
		"initramfs.img":     fmt.Sprintf(coreos, coreosVersion, arch, "initramfs", ".img"),
		"rootfs.img":        fmt.Sprintf(coreos, coreosVersion, arch, "rootfs", ".img"),
		"kernel.sig":        fmt.Sprintf(coreos, coreosVersion, arch, "kernel", ".sig"),
		"initramfs.img.sig": fmt.Sprintf(coreos, coreosVersion, arch, "initramfs", ".img.sig"),
		"rootfs.img.sig":    fmt.Sprintf(coreos, coreosVersion, arch, "rootfs", ".img.sig"),
		"fedora.gpg":        "https://fedoraproject.org/fedora.gpg",
	}
}

func (prov *CoreOSProvider) ExtractDownloads(ctx context.Context, cacheDir map[string]io.Reader) (map[string]io.Reader, error) {

	wrk := map[string]io.Reader{}

	var err error
	if kernel, cached := cacheDir["kernel.coreos-extract"]; !cached {
		wrk["kernel.coreos-extract"], err = prov.Kernel(ctx, cacheDir)
		if err != nil {
			return nil, errors.Errorf("processing kernel: %w", err)
		}
	} else {
		wrk["kernel.coreos-extract"] = io.NopCloser(kernel)
	}

	if initramfs, cached := cacheDir["initramfs.coreos-extract.cpio"]; !cached {
		wrk["initramfs.coreos-extract.cpio"], err = prov.Initramfs(ctx, cacheDir)
		if err != nil {
			return nil, errors.Errorf("processing initramfs: %w", err)
		}
	} else {
		wrk["initramfs.coreos-extract.cpio"] = io.NopCloser(initramfs)
	}

	if rootfs, cached := cacheDir["rootfs.coreos-extract.img"]; !cached {

		wrk["rootfs.coreos-extract.img"], err = prov.Rootfs(ctx, cacheDir)
		if err != nil {
			return nil, errors.Errorf("processing rootfs: %w", err)
		}
	} else {
		wrk["rootfs.coreos-extract.img"] = io.NopCloser(rootfs)
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

func (prov *CoreOSProvider) Rootfs(ctx context.Context, mem map[string]io.Reader) (io.Reader, error) {
	return io.NopCloser(mem["rootfs.img"]), nil
}

func (prov *CoreOSProvider) Kernel(ctx context.Context, mem map[string]io.Reader) (io.Reader, error) {

	kernelReader, err := unzbootgo.ProcessKernel(ctx, mem["kernel"])
	if err != nil {
		return nil, errors.Errorf("processing kernel: %w", err)
	}

	return kernelReader, nil
}

func (prov *CoreOSProvider) Initramfs(ctx context.Context, mem map[string]io.Reader) (io.Reader, error) {

	// just decompress the initramfs, it is either gz or xz
	read, err := (archives.Zstd{}).OpenReader(mem["initramfs.img"])
	if err != nil {
		return nil, errors.Errorf("decompressing initramfs: %w", err)
	}

	slog.InfoContext(ctx, "decompressed initramfs")

	return read, nil
}

func (prov *CoreOSProvider) KernelArgs() (args string) {
	return "coreos.live.rootfs_url=/dev/nvme0n1p1"
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
