package oci

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"golang.org/x/crypto/ssh"
	"golang.org/x/mod/semver"

	"gitlab.com/tozd/go/errors"

	v1 "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/walteh/ec1/pkg/guest"
	"github.com/walteh/ec1/pkg/host"
	"github.com/walteh/ec1/pkg/vmm"
)

var (
	_ vmm.VMIProvider             = &OCIProvider{}
	_ vmm.DownloadableVMIProvider = &OCIProvider{}
	_ vmm.LinuxVMIProvider        = &OCIProvider{}
)

// OCIProvider implements VMI provider for OCI container images
type OCIProvider struct {
	imageRef     string
	kernelImage  string // Separate kernel image reference (e.g., "ghcr.io/linuxkit/kernel:6.6.13")
	architecture string
	platform     *v1.Platform
}

// NewOCIProvider creates a new OCI container VMI provider
func NewOCIProvider(imageRef, kernelImage string) *OCIProvider {
	arch := host.CurrentKernelArch()

	// Convert to OCI platform format
	var ociArch string
	switch arch {
	case "aarch64":
		ociArch = "arm64"
	case "x86_64":
		ociArch = "amd64"
	default:
		ociArch = arch
	}

	return &OCIProvider{
		imageRef:     imageRef,
		kernelImage:  kernelImage,
		architecture: arch,
		platform: &v1.Platform{
			Architecture: ociArch,
			OS:           "linux",
		},
	}
}

func (p *OCIProvider) Name() string {
	return "oci"
}

func (p *OCIProvider) Version() string {
	// Extract tag from image reference, default to latest
	parts := strings.Split(p.imageRef, ":")
	if len(parts) > 1 {
		return semver.Canonical("v" + parts[len(parts)-1])
	}
	return semver.Canonical("v1.0.0")
}

func (p *OCIProvider) GuestKernelType() guest.GuestKernelType {
	return guest.GuestKernelTypeLinux
}

func (p *OCIProvider) Downloads() map[string]string {
	downloads := make(map[string]string)

	// For now, we'll use a simple approach where we expect:
	// 1. A container image that will be converted to rootfs
	// 2. A separate kernel image (could be from LinuxKit or similar)

	// The actual implementation would use containerd/docker APIs
	// For proof of concept, we'll simulate this
	downloads["container-image"] = p.imageRef
	if p.kernelImage != "" {
		downloads["kernel-image"] = p.kernelImage
	}

	return downloads
}

func (p *OCIProvider) ExtractDownloads(ctx context.Context, files map[string]io.Reader) (map[string]io.Reader, error) {
	slog.InfoContext(ctx, "extracting OCI container image", "image", p.imageRef)

	extracted := make(map[string]io.Reader)

	// Check cache first
	if rootfs, cached := files["rootfs.oci-extract.img"]; cached {
		extracted["rootfs.oci-extract.img"] = rootfs
	} else {
		// Extract container image to rootfs
		if containerReader, ok := files["container-image"]; ok {
			rootfsReader, err := p.extractContainerToRootfs(ctx, containerReader)
			if err != nil {
				return nil, errors.Errorf("extracting container to rootfs: %w", err)
			}
			extracted["rootfs.oci-extract.img"] = rootfsReader
		}
	}

	// Handle kernel extraction
	if kernel, cached := files["kernel.oci-extract"]; cached {
		extracted["kernel.oci-extract"] = kernel
	} else {
		if kernelReader, ok := files["kernel-image"]; ok {
			extracted["kernel.oci-extract"] = kernelReader
		} else {
			// Use a default kernel - in practice this would be more sophisticated
			return nil, errors.New("no kernel image provided for OCI container")
		}
	}

	// Create a minimal initramfs for container boot
	if initramfs, cached := files["initramfs.oci-extract.cpio"]; cached {
		extracted["initramfs.oci-extract.cpio"] = initramfs
	} else {
		initramfsReader, err := p.createContainerInitramfs(ctx)
		if err != nil {
			return nil, errors.Errorf("creating container initramfs: %w", err)
		}
		extracted["initramfs.oci-extract.cpio"] = initramfsReader
	}

	return extracted, nil
}

func (p *OCIProvider) extractContainerToRootfs(ctx context.Context, containerReader io.Reader) (io.Reader, error) {
	// This is a simplified implementation
	// In practice, this would:
	// 1. Use containerd/docker APIs to pull the image
	// 2. Extract all layers to create a filesystem
	// 3. Convert to a disk image format suitable for VM boot

	slog.InfoContext(ctx, "converting OCI container to rootfs", "image", p.imageRef)

	// For proof of concept, we'll create a simple ext4 filesystem
	// In reality, this would involve:
	// - Pulling the container image using containerd client
	// - Extracting layers in order
	// - Creating a bootable filesystem

	// Placeholder: return the container reader as-is for now
	// This would need to be replaced with actual container extraction logic
	return containerReader, nil
}

func (p *OCIProvider) createContainerInitramfs(ctx context.Context) (io.Reader, error) {
	// Create a minimal initramfs that can:
	// 1. Mount the rootfs
	// 2. Switch to the container's entrypoint
	// 3. Handle container-specific initialization

	slog.InfoContext(ctx, "creating container-aware initramfs")

	// This would create a custom initramfs that understands container semantics
	// For now, return an empty reader as placeholder
	return strings.NewReader(""), nil
}

func (p *OCIProvider) RootfsPath() string {
	return "rootfs.oci-extract.img"
}

func (p *OCIProvider) KernelPath() string {
	return "kernel.oci-extract"
}

func (p *OCIProvider) InitramfsPath() string {
	return "initramfs.oci-extract.cpio"
}

func (p *OCIProvider) KernelArgs() string {
	// Container-specific kernel arguments
	// These would be derived from the container image metadata
	args := []string{
		"console=hvc0",
		"root=/dev/nvme0n1p1", // Assuming rootfs is on first NVMe partition
		"init=/sbin/init",     // Or container's entrypoint
		"rw",
		"modules_load=vsock,vmw_vsock_virtio_transport",
	}

	return strings.Join(args, " ")
}

func (p *OCIProvider) BootProvisioners() []vmm.BootProvisioner {
	// Could include container-specific boot provisioners
	// e.g., for setting up container networking, volumes, etc.
	return []vmm.BootProvisioner{}
}

func (p *OCIProvider) RuntimeProvisioners() []vmm.RuntimeProvisioner {
	// Could include container runtime provisioners
	// e.g., for container lifecycle management
	return []vmm.RuntimeProvisioner{}
}

func (p *OCIProvider) ShutdownCommand() string {
	// Use container's stop signal or default
	return "poweroff"
}

func (p *OCIProvider) SSHConfig() *ssh.ClientConfig {
	// Container images might not have SSH by default
	// This would need to be configured based on the container
	return &ssh.ClientConfig{
		User:            "root",
		Auth:            []ssh.AuthMethod{ssh.Password("container")},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
}

func (p *OCIProvider) InitScript(ctx context.Context) (string, error) {
	// Generate an init script that:
	// 1. Sets up the container environment
	// 2. Runs the container's entrypoint/cmd

	script := fmt.Sprintf(`#!/bin/sh

# Container initialization script for %s
echo "Initializing OCI container: %s"

# Set up container environment
export PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin

# Mount essential filesystems
mount -t proc proc /proc
mount -t sysfs sysfs /sys
mount -t devtmpfs devtmpfs /dev

# Load necessary modules
modprobe vsock
modprobe vmw_vsock_virtio_transport

echo "Container %s ready"

# In a real implementation, this would:
# 1. Read the container's entrypoint and cmd from image metadata
# 2. Set up the container's working directory
# 3. Apply container environment variables
# 4. Execute the container's main process

# For now, just start a shell
exec /bin/sh
`, p.imageRef, p.imageRef, p.imageRef)

	return script, nil
}
