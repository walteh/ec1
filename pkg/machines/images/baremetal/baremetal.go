package baremetal

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"

	"golang.org/x/crypto/ssh"
	"golang.org/x/mod/semver"

	"github.com/kdomanski/iso9660"
	"gitlab.com/tozd/go/errors"

	"github.com/walteh/ec1/pkg/hypervisors"
	"github.com/walteh/ec1/pkg/machines/bootloader"
	"github.com/walteh/ec1/pkg/machines/guest"
	"github.com/walteh/ec1/pkg/machines/host"
)

const baremetalVersion = "v1.0.0"

var (
	_ hypervisors.VMIProvider                = &BareMetalProvider{}
	_ hypervisors.LinuxVMIProvider           = &BareMetalProvider{}
	_ hypervisors.CustomExtractorVMIProvider = &BareMetalProvider{}
)

type BareMetalProvider struct {
}

// BootLoaderConfig implements hypervisors.LinuxVMIProvider.
func (prov *BareMetalProvider) BootLoaderConfig(cacheDir string) *bootloader.LinuxBootloader {
	var kernelFile, kernelCmdLine string

	if host.CurrentKernelArch() == "aarch64" {
		kernelFile = "vmlinuz-virt-uncompressed"
		kernelCmdLine = "console=ttyAMA0 modloop=modloop-virt modules=loop,squashfs,sd-mod,usb-storage"
	} else {
		kernelFile = "vmlinuz-virt"
		kernelCmdLine = "console=ttyS0 modloop=modloop-virt modules=loop,squashfs,sd-mod,usb-storage"
	}

	return &bootloader.LinuxBootloader{
		VmlinuzPath:   filepath.Join(cacheDir, kernelFile),
		KernelCmdLine: kernelCmdLine,
		InitrdPath:    filepath.Join(cacheDir, "initramfs-virt"),
	}
}

// BootProvisioners implements hypervisors.VMIProvider.
func (prov *BareMetalProvider) BootProvisioners() []hypervisors.BootProvisioner {
	return []hypervisors.BootProvisioner{}
}

// DiskImageURL implements hypervisors.VMIProvider.
func (prov *BareMetalProvider) DiskImageURL() string {
	arch := host.CurrentKernelArch()

	// Use Alpine Linux ISO which contains both kernel and initrd
	var alpineArch string
	if arch == "aarch64" {
		alpineArch = "aarch64"
	} else {
		alpineArch = "x86_64"
	}

	alpineVersion := "3.21.1"
	return fmt.Sprintf("https://dl-cdn.alpinelinux.org/alpine/v3.21/releases/%s/alpine-virt-%s-%s.iso",
		alpineArch, alpineVersion, alpineArch)
}

// CustomExtraction implements hypervisors.CustomExtractorVMIProvider
func (prov *BareMetalProvider) CustomExtraction(ctx context.Context, cacheDir string) error {
	// for debnigging copy recusivly the cache dir to a new temp dir and log the name

	// Find the ISO file
	isoFile := filepath.Join(cacheDir, "alpine-virt.iso")
	entries, err := os.ReadDir(cacheDir)
	if err != nil {
		return errors.Errorf("reading cache dir: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".iso" {
			isoFile = filepath.Join(cacheDir, entry.Name())
			break
		}
	}

	// Open the ISO file
	file, err := os.Open(isoFile)
	if err != nil {
		return errors.Errorf("opening ISO file: %w", err)
	}
	defer file.Close()

	// Read the ISO file
	image, err := iso9660.OpenImage(file)
	if err != nil {
		return errors.Errorf("reading ISO image: %w", err)
	}

	// Find and extract the kernel and initrd files
	root, err := image.RootDir()
	if err != nil {
		return errors.Errorf("getting ISO root directory: %w", err)
	}

	// Get boot directory
	bootFiles, err := findDirectory(root, "boot")
	if err != nil {
		return errors.Errorf("finding boot directory: %w", err)
	}

	// Extract kernel and initramfs
	kernelFile, err := findFile(bootFiles, "vmlinuz-virt")
	if err != nil {
		return errors.Errorf("finding kernel file: %w", err)
	}

	kernelPath := filepath.Join(cacheDir, "vmlinuz-virt")
	if err := extractFile(kernelFile, kernelPath); err != nil {
		return errors.Errorf("extracting kernel: %w", err)
	}

	tempDir, err := os.MkdirTemp("", "alpine-virt")
	if err != nil {
		return errors.Errorf("creating temp dir: %w", err)
	}

	bootDir := filepath.Join(tempDir, "boot")
	err = os.MkdirAll(bootDir, 0755)
	if err != nil {
		return errors.Errorf("creating boot dir: %w", err)
	}

	for _, file := range bootFiles {
		if file.Reader() == nil {
			continue
		}

		dat, err := io.ReadAll(file.Reader())
		if err != nil {
			return errors.Errorf("reading file: %w", err)
		}

		os.WriteFile(filepath.Join(tempDir, "boot", file.Name()), dat, 0644)
	}

	cmd := exec.CommandContext(ctx, "cp", "-r", cacheDir, tempDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		return errors.Errorf("copying cache dir: %w", err)
	}

	slog.InfoContext(ctx, "copied cache dir to temp dir", "tempDir", tempDir)

	// For ARM64, we need to decompress the kernel for Apple Virtualization Framework
	if host.CurrentKernelArch() == "aarch64" {
		uncompressedKernelPath := filepath.Join(cacheDir, "vmlinuz-virt-uncompressed")
		if err := decompressKernel(ctx, kernelPath, uncompressedKernelPath); err != nil {
			return errors.Errorf("decompressing kernel: %w", err)
		}
	}

	initrdFile, err := findFile(bootFiles, "initramfs-virt")
	if err != nil {
		return errors.Errorf("finding initramfs file: %w", err)
	}

	if err := extractFile(initrdFile, filepath.Join(cacheDir, "initramfs-virt")); err != nil {
		return errors.Errorf("extracting initramfs: %w", err)
	}

	return nil
}

// Helper function to decompress a kernel
func decompressKernel(ctx context.Context, inputPath, outputPath string) error {
	// Read the compressed kernel
	compressedData, err := os.ReadFile(inputPath)
	if err != nil {
		return errors.Errorf("reading compressed kernel: %w", err)
	}

	rdr, err := host.DecompressUnknown(ctx, bytes.NewReader(compressedData))
	if err != nil {
		return errors.Errorf("decompressing kernel: %w", err)
	}

	compressedData, err = io.ReadAll(rdr)
	if err != nil {
		return errors.Errorf("reading decompressed kernel: %w", err)
	}

	err = os.WriteFile(outputPath, compressedData, 0644)
	if err != nil {
		return errors.Errorf("writing decompressed kernel: %w", err)
	}

	return nil
}

// Helper functions for ISO extraction
func findDirectory(root *iso9660.File, name string) ([]*iso9660.File, error) {
	children, err := root.GetChildren()
	if err != nil {
		return nil, err
	}

	for _, child := range children {
		if child.IsDir() && child.Name() == name {
			return child.GetChildren()
		}
	}

	return nil, errors.New("directory not found")
}

func findFile(files []*iso9660.File, name string) (*iso9660.File, error) {
	for _, file := range files {
		if !file.IsDir() && file.Name() == name {
			return file, nil
		}
	}

	return nil, errors.New("file not found")
}

func extractFile(file *iso9660.File, destPath string) error {
	reader := file.Reader()

	dest, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer dest.Close()

	_, err = io.Copy(dest, reader)
	return err
}

// GuestKernelType implements hypervisors.VMIProvider.
func (prov *BareMetalProvider) GuestKernelType() guest.GuestKernelType {
	return guest.GuestKernelTypeLinux
}

// RuntimeProvisioners implements hypervisors.VMIProvider.
func (prov *BareMetalProvider) RuntimeProvisioners() []hypervisors.RuntimeProvisioner {
	return []hypervisors.RuntimeProvisioner{}
}

// SupportsEFI implements hypervisors.VMIProvider.
func (prov *BareMetalProvider) SupportsEFI() bool {
	return false
}

func NewBareMetalProvider() *BareMetalProvider {
	return &BareMetalProvider{}
}

func (prov *BareMetalProvider) Name() string {
	return "baremetal"
}

func (prov *BareMetalProvider) Version() string {
	return semver.Canonical(baremetalVersion)
}

func (prov *BareMetalProvider) ShutdownCommand() string {
	return "poweroff"
}

func (prov *BareMetalProvider) SSHConfig() *ssh.ClientConfig {
	return &ssh.ClientConfig{
		User: "root",
		Auth: []ssh.AuthMethod{ssh.Password("")}, // Alpine has empty root password by default
		// #nosec 106 -- the host SSH key of the VM will change each time it boots
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
}
