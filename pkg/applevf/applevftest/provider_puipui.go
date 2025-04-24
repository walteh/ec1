package applevftest

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/crc-org/vfkit/pkg/config"
	"gitlab.com/tozd/go/errors"
	"golang.org/x/crypto/ssh"
)

const puipuiMemoryMiB = 1 * 1024
const puipuiCPUs = 2

const puipuiVersion = "v1.0.3"

var _ OsProvider = &PuiPuiProvider{}

type PuiPuiProvider struct {
	vmlinuz    string
	initramfs  string
	kernelArgs string
}

func NewPuipuiProvider() *PuiPuiProvider {
	return &PuiPuiProvider{}
}

func (prov *PuiPuiProvider) URL() string {
	return fmt.Sprintf("https://github.com/Code-Hex/puipui-linux/releases/download/%s/puipui_linux_%s_%s.tar.gz", puipuiVersion, puipuiVersion, kernelArch())
}

func (prov *PuiPuiProvider) Initialize(ctx context.Context, cacheDir string) error {
	filez, err := os.ReadDir(cacheDir)
	if err != nil {
		return err
	}
	files := []string{}
	for _, file := range filez {
		files = append(files, filepath.Join(cacheDir, file.Name()))
	}
	prov.vmlinuz, err = findKernel(ctx, files)
	if err != nil {
		return err
	}
	prov.initramfs, err = findFile(files, "initramfs.cpio.gz")
	if err != nil {
		return err
	}
	prov.kernelArgs = "quiet"
	return nil
}

func (puipui *PuiPuiProvider) ToVirtualMachine() (*config.VirtualMachine, error) {
	bootloader := config.NewLinuxBootloader(puipui.vmlinuz, puipui.kernelArgs, puipui.initramfs)
	vm := config.NewVirtualMachine(puipuiCPUs, puipuiMemoryMiB, bootloader)

	return vm, nil
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

func (puipui *PuiPuiProvider) SSHAccessMethods() []SSHAccessMethod {
	return []SSHAccessMethod{
		{
			network: "tcp",
			port:    22,
		},
		{
			network: "vsock",
			port:    2222,
		},
	}
}

func findKernel(ctx context.Context, files []string) (string, error) {
	switch runtime.GOARCH {
	case "amd64":
		return findFile(files, "bzImage")
	case "arm64":
		compressed, err := findFile(files, "Image.gz")
		if err != nil {
			return "", err
		}

		dir := filepath.Dir(compressed)

		err = extractIntoDir(ctx, compressed, dir)
		if err != nil {
			return "", err
		}

		expectedOutFile := filepath.Join(dir, "Image")

		// make sure the file is in the dir
		if _, err := os.Stat(expectedOutFile); err != nil {
			return "", errors.Errorf("expected file %s not found", expectedOutFile)
		}

		return expectedOutFile, nil
	default:
		return "", fmt.Errorf("unsupported architecture '%s'", runtime.GOARCH)
	}
}
