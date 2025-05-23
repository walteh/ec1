package puipui

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"golang.org/x/crypto/ssh"
	"golang.org/x/mod/semver"

	"github.com/mholt/archives"
	"gitlab.com/tozd/go/errors"

	"github.com/walteh/ec1/pkg/ext/iox"
	"github.com/walteh/ec1/pkg/guest"
	"github.com/walteh/ec1/pkg/host"
	"github.com/walteh/ec1/pkg/vmm"
)

// const puipuiMemoryMiB = 1 * 1024
// const puipuiCPUs = 2

const puipuiVersion = "v1.0.3"

const (
	extractedCpioName   = "initramfs.ec1-extract.cpio"
	extractedKernelName = "kernel.ec1-extract"
)

var (
	_ vmm.VMIProvider             = &PuiPuiProvider{}
	_ vmm.LinuxVMIProvider        = &PuiPuiProvider{}
	_ vmm.DownloadableVMIProvider = &PuiPuiProvider{}
)

type PuiPuiProvider struct {
	// vmlinuz    string
	// initramfs  string
	// kernelArgs string
}

// BootLoaderConfig implements vmm.LinuxVMIProvider.

// BootProvisioners implements vmm.VMIProvider.
func (prov *PuiPuiProvider) BootProvisioners() []vmm.BootProvisioner {
	return []vmm.BootProvisioner{}
}

// GuestKernelType implements vmm.VMIProvider.
func (prov *PuiPuiProvider) GuestKernelType() guest.GuestKernelType {
	return guest.GuestKernelTypeLinux
}

// RuntimeProvisioners implements vmm.VMIProvider.
func (prov *PuiPuiProvider) RuntimeProvisioners() []vmm.RuntimeProvisioner {
	return []vmm.RuntimeProvisioner{}
}

// SupportsEFI implements vmm.VMIProvider.
func (prov *PuiPuiProvider) SupportsEFI() bool {
	return false
}

func NewPuipuiProvider() *PuiPuiProvider {
	return &PuiPuiProvider{}
}

func (prov *PuiPuiProvider) Name() string {
	return "puipui"
}

func (prov *PuiPuiProvider) Version() string {
	return semver.Canonical(puipuiVersion)
}

func (prov *PuiPuiProvider) InitramfsPath() (path string) {
	return extractedCpioName
}

func (prov *PuiPuiProvider) KernelPath() (path string) {
	return extractedKernelName
}

func (prov *PuiPuiProvider) KernelArgs() (args string) {
	return ""
}

func (prov *PuiPuiProvider) RootfsPath() (path string) {
	return "" // no rootfs
}

func (prov *PuiPuiProvider) InitScript(ctx context.Context) (string, error) {
	script := `
#!/bin/sh

echo "Hello, world!"
`

	return script, nil
}

func (prov *PuiPuiProvider) Downloads() map[string]string {
	return map[string]string{
		"root": fmt.Sprintf("https://github.com/Code-Hex/puipui-linux/releases/download/%s/puipui_linux_%s_%s.tar.gz", puipuiVersion, puipuiVersion, host.CurrentKernelArch()),
	}
}

func (prov *PuiPuiProvider) ExtractDownloads(ctx context.Context, cacheDir map[string]io.Reader) (map[string]io.Reader, error) {
	out := make(map[string]io.Reader)

	_, cpioExists := cacheDir[extractedCpioName]
	_, kernelExists := cacheDir[extractedKernelName]

	if cpioExists && kernelExists {
		out[extractedCpioName] = iox.PreservedNopCloser(cacheDir[extractedCpioName])
		out[extractedKernelName] = iox.PreservedNopCloser(cacheDir[extractedKernelName])
		return out, nil
	}

	// extract the files
	fmtz := archives.CompressedArchive{
		Archival:    archives.Tar{},
		Compression: archives.Gz{},
		Extraction:  archives.Tar{},
	}

	files := make(map[string]io.ReadCloser)

	err := fmtz.Extract(ctx, cacheDir["root"], func(ctx context.Context, info archives.FileInfo) error {
		rdr, err := info.Open()
		if err != nil {
			return errors.Errorf("opening file: %w", err)
		}
		defer rdr.Close()
		data, err := io.ReadAll(rdr)
		if err != nil {
			return errors.Errorf("reading file: %w", err)
		}
		files[info.Name()] = iox.PreservedNopCloser(bytes.NewReader(data))
		return nil
	})
	if err != nil {
		return nil, errors.Errorf("extracting files: %w", err)
	}

	for name, file := range files {
		switch name {
		case "initramfs.cpio.gz":
			rdr, err := (&archives.Gz{}).OpenReader(file)
			if err != nil {
				return nil, errors.Errorf("opening initramfs: %w", err)
			}
			out[extractedCpioName] = rdr
		case "Image.gz": // only for arm64
			out[extractedKernelName], err = (&archives.Gz{}).OpenReader(file)
			if err != nil {
				return nil, errors.Errorf("ungzipping kernel: %w", err)
			}
		case "bzImage": // only for amd64
			out[extractedKernelName] = file
		}
	}
	if len(out) != 2 {
		return nil, errors.Errorf("expected 2 files, got %d", len(out))
	}

	return out, nil
}

func (puipui *PuiPuiProvider) ShutdownCommand() string {
	return "poweroff"
}

func (puipui *PuiPuiProvider) SSHConfig() *ssh.ClientConfig {
	return &ssh.ClientConfig{
		User: "root",
		Auth: []ssh.AuthMethod{ssh.Password("passwd")},
		// #nosec 106 -- the host SSH key of the VM will change each time it boots
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
}

// func findKernel(ctx context.Context, files []string) (string, error) {
// 	switch runtime.GOARCH {
// 	case "amd64":
// 		return findFile(files, "bzImage")
// 	case "arm64":
// 		compressed, err := findFile(files, "Image.gz")
// 		if err != nil {
// 			return "", err
// 		}

// 		dir := filepath.Dir(compressed)

// 		err = extractIntoDir(ctx, compressed, dir)
// 		if err != nil {
// 			return "", err
// 		}

// 		expectedOutFile := filepath.Join(dir, "Image")

// 		// make sure the file is in the dir
// 		if _, err := os.Stat(expectedOutFile); err != nil {
// 			return "", errors.Errorf("expected file %s not found", expectedOutFile)
// 		}

// 		return expectedOutFile, nil
// 	default:
// 		return "", fmt.Errorf("unsupported architecture '%s'", runtime.GOARCH)
// 	}
// }
