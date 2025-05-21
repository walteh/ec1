package coreos

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"golang.org/x/crypto/ssh"
	"golang.org/x/mod/semver"

	"github.com/coreos/stream-metadata-go/fedoracoreos"
	"gitlab.com/tozd/go/errors"

	"github.com/walteh/ec1/pkg/ext/archivesx"
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

func (prov *CoreOSProvider) ExtractDownloads(ctx context.Context, cacheDir map[string]io.Reader) (map[string]io.ReadCloser, error) {

	wrk := map[string]io.ReadCloser{}

	var err error
	wrk["kernel.coreos-extract"], err = prov.Kernel(ctx, cacheDir)
	if err != nil {
		return nil, errors.Errorf("processing kernel: %w", err)
	}

	wrk["initramfs.coreos-extract"], err = prov.Initramfs(ctx, cacheDir)
	if err != nil {
		return nil, errors.Errorf("processing initramfs: %w", err)
	}

	wrk["rootfs.coreos-extract"], err = prov.Rootfs(ctx, cacheDir)
	if err != nil {
		return nil, errors.Errorf("processing rootfs: %w", err)
	}

	return wrk, nil
}

func (prov *CoreOSProvider) RootfsPath() (path string) {
	return "rootfs.coreos-extract"
}

func (prov *CoreOSProvider) KernelPath() (path string) {
	return "kernel.coreos-extract"
}

func (prov *CoreOSProvider) InitramfsPath() (path string) {
	return "initramfs.coreos-extract"
}

func (prov *CoreOSProvider) InitScript(ctx context.Context) (string, error) {
	script := `
#!/bin/sh

echo "Hello, world!"
`
	return script, nil
}

func (prov *CoreOSProvider) Rootfs(ctx context.Context, mem map[string]io.Reader) (io.ReadCloser, error) {
	return io.NopCloser(mem["rootfs.img"]), nil
}

func (prov *CoreOSProvider) Kernel(ctx context.Context, mem map[string]io.Reader) (io.ReadCloser, error) {

	kernelReader, err := unzbootgo.ProcessKernel(ctx, mem["kernel"])
	if err != nil {
		slog.Error("failed to process kernel", "error", err)
		return nil, err
	}

	return kernelReader, nil
}

func (prov *CoreOSProvider) Initramfs(ctx context.Context, mem map[string]io.Reader) (io.ReadCloser, error) {

	// just decompress the initramfs, it is either gz or xz
	read, compressed, err := archivesx.IdentifyAndDecompress(ctx, "", mem["initramfs.img"])
	if err != nil {
		return nil, errors.Errorf("decompressing initramfs: %w", err)
	}

	if !compressed {
		return nil, errors.New("initramfs is not compressed ... expected gzip or xz")
	}

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
