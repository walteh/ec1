package applevftest

import (
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/crc-org/vfkit/pkg/config"
	"github.com/xi2/xz"

	"github.com/cavaliergopher/grab/v3"
	"github.com/crc-org/crc/v2/pkg/extract"
	"gitlab.com/tozd/go/errors"
	"golang.org/x/crypto/ssh"
)

const fedoraVersion = "42"
const fedoraRelease = "1.1"
const puipuiVersion = "v1.0.3"

type OsProvider interface {
	URL() string
	Uncompress(ctx context.Context, cacheFile string, destDir string) error
	Initialize(ctx context.Context, cacheDir string) error
	// Fetch(ctx context.Context, cacheFile string, destDir string) error
	ToVirtualMachine() (*config.VirtualMachine, error)
	SSHConfig() *ssh.ClientConfig
	SSHAccessMethods() []SSHAccessMethod
}

func cacheDir(urld string) (string, error) {
	hrlHasher := sha256.New()
	hrlHasher.Write([]byte(urld))
	hrlHash := hex.EncodeToString(hrlHasher.Sum(nil))

	// parse the url and get the filename
	parsedURL, err := url.Parse(urld)
	if err != nil {
		return "", err
	}
	filename := filepath.Base(parsedURL.Path)
	hostname := parsedURL.Host

	filename = fmt.Sprintf("%s_%s_%s", hostname, hrlHash, filename)
	userCacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(userCacheDir, "vfkit-testing", "cache", filename), nil
}

func SetupOS(t *testing.T, prov OsProvider) error {
	ctx := t.Context()
	tmpDir := t.TempDir()
	url := prov.URL()

	cacheDir, err := cacheDir(url)
	if err != nil {
		return err
	}

	cacheFile := filepath.Join(cacheDir, filepath.Base(url))

	if _, err := os.Stat(cacheFile); err != nil {
		grab.DefaultClient.UserAgent = "vfkit"
		resp, err := grab.Get(tmpDir, url)
		if err != nil {
			return err
		}
		// move the file to the cache
		err = os.MkdirAll(cacheDir, 0755)
		if err != nil {
			return err
		}
		err = os.Rename(resp.Filename, cacheFile)
		if err != nil {
			return err
		}
	}

	extractedDir := filepath.Join(tmpDir, "extracted")
	err = os.MkdirAll(extractedDir, 0755)
	if err != nil {
		return err
	}

	err = prov.Uncompress(ctx, cacheFile, extractedDir)
	if err != nil {
		return err
	}

	err = prov.Initialize(ctx, extractedDir)
	if err != nil {
		return err
	}

	return nil
}

func kernelArch() string {
	switch runtime.GOARCH {
	case "amd64":
		return "x86_64"
	case "arm64":
		return "aarch64"
	default:
		return "invalid"
	}
}

func (prov *PuiPuiProvider) URL() string {
	return fmt.Sprintf("https://github.com/Code-Hex/puipui-linux/releases/download/%s/puipui_linux_%s_%s.tar.gz", puipuiVersion, puipuiVersion, kernelArch())
}

func (prov *FedoraProvider) URL() string {
	arch := kernelArch()
	buildString := fmt.Sprintf("%s-%s.%s", fedoraVersion, fedoraRelease, arch)
	return fmt.Sprintf("https://download.fedoraproject.org/pub/fedora/linux/releases/%s/Cloud/%s/images/Fedora-Cloud-Base-AmazonEC2-%s.raw.xz", fedoraVersion, arch, buildString)
}

func (prov *PuiPuiProvider) Uncompress(ctx context.Context, cacheFile string, destDir string) error {
	_, err := extract.Uncompress(ctx, cacheFile, destDir)
	if err != nil {
		return errors.Errorf("uncompressing pui pui: %w", err)
	}
	return nil
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
	prov.vmlinuz, err = findKernel(files)
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

func (prov *FedoraProvider) Uncompress(ctx context.Context, cacheFile string, destDir string) error {
	file, err := os.Open(filepath.Clean(cacheFile))
	if err != nil {
		return err
	}
	defer file.Close()

	reader, err := xz.NewReader(file, 0)
	if err != nil {
		return err
	}

	xzCutName, _ := strings.CutSuffix(filepath.Base(prov.URL()), ".xz")
	outPath := filepath.Join(destDir, xzCutName)
	out, err := os.Create(outPath)
	if err != nil {
		return err
	}

	_, err = io.Copy(out, reader)
	if err != nil {
		return err
	}

	return err
}

func (prov *FedoraProvider) Initialize(ctx context.Context, cacheDir string) error {
	xzCutName, _ := strings.CutSuffix(filepath.Base(prov.URL()), ".xz")
	prov.diskImage = filepath.Join(cacheDir, xzCutName)
	// prov.efiVariableStorePath = filepath.Join(cacheDir, "efivars.img")
	// prov.createVariableStore = true
	return nil
}

var _ OsProvider = &PuiPuiProvider{}
var _ OsProvider = &FedoraProvider{}

type SSHAccessMethod struct {
	network string
	port    uint
}

type PuiPuiProvider struct {
	vmlinuz    string
	initramfs  string
	kernelArgs string
}

func NewPuipuiProvider() *PuiPuiProvider {
	return &PuiPuiProvider{}
}

type FedoraProvider struct {
	diskImage            string
	efiVariableStorePath string
	createVariableStore  bool
}

func NewFedoraProvider() *FedoraProvider {
	return &FedoraProvider{}
}

func findFile(files []string, filename string) (string, error) {
	for _, f := range files {
		if filepath.Base(f) == filename {
			return f, nil
		}
	}

	return "", fmt.Errorf("could not find %s", filename)
}

func uncompressPuiPuiKernel(gzFile string) (string, error) {
	reader, err := os.Open(gzFile)
	if err != nil {
		return "", err
	}
	defer reader.Close()
	gzReader, err := gzip.NewReader(reader)
	if err != nil {
		return "", err
	}
	defer gzReader.Close()
	destFile, _ := strings.CutSuffix(gzFile, ".gz")
	writer, err := os.OpenFile(destFile, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
	if err != nil {
		return "", err
	}
	defer writer.Close()

	// https://stackoverflow.com/questions/67327323/g110-potential-dos-vulnerability-via-decompression-bomb-gosec
	for {
		_, err = io.CopyN(writer, gzReader, 1024*1024)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return "", err
		}
	}
	return destFile, nil
}

func findKernel(files []string) (string, error) {
	switch runtime.GOARCH {
	case "amd64":
		return findFile(files, "bzImage")
	case "arm64":
		compressed, err := findFile(files, "Image.gz")
		if err != nil {
			return "", err
		}
		return uncompressPuiPuiKernel(compressed)
	default:
		return "", fmt.Errorf("unsupported architecture '%s'", runtime.GOARCH)
	}
}

const puipuiMemoryMiB = 1 * 1024
const puipuiCPUs = 2

func (puipui *PuiPuiProvider) ToVirtualMachine() (*config.VirtualMachine, error) {
	bootloader := config.NewLinuxBootloader(puipui.vmlinuz, puipui.kernelArgs, puipui.initramfs)
	vm := config.NewVirtualMachine(puipuiCPUs, puipuiMemoryMiB, bootloader)

	return vm, nil
}

func (fedora *FedoraProvider) ToVirtualMachine() (*config.VirtualMachine, error) {
	bootloader := config.NewEFIBootloader(fedora.efiVariableStorePath, fedora.createVariableStore)
	vm := config.NewVirtualMachine(puipuiCPUs, puipuiMemoryMiB, bootloader)

	return vm, nil
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

func (fedora *FedoraProvider) SSHConfig() *ssh.ClientConfig {
	return &ssh.ClientConfig{
		User: "vfkituser",
		Auth: []ssh.AuthMethod{ssh.Password("vfkittest")},
		// #nosec 106 -- the host SSH key of the VM will change each time it boots
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

}

func (fedora *FedoraProvider) SSHAccessMethods() []SSHAccessMethod {
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
