# OCI Container to Virtio Device Converter

This package provides functionality to convert OCI container images into virtio block devices that can be used as rootfs in virtual machines.

## Overview

The `oci` package allows you to:

1. **Pull OCI container images** from registries (Docker Hub, etc.)
2. **Extract container layers** to a filesystem
3. **Create bootable disk images** with various filesystem types
4. **Generate virtio devices** ready for VM use

This is particularly useful for running containers as VMs, providing better isolation while maintaining the convenience of container images.

## Features

-   ✅ **Multiple Registry Support**: Docker Hub, Quay, Harbor, etc.
-   ✅ **Cross-Platform**: Support for different architectures (arm64, amd64)
-   ✅ **Multiple Filesystems**: ext4, ext3, XFS support
-   ✅ **Configurable Size**: Specify rootfs image size
-   ✅ **Read-Only Support**: Create immutable rootfs images
-   ✅ **Caching**: Efficient layer extraction and caching

## Usage

### Basic Example

```go
package main

import (
    "context"
    "github.com/walteh/ec1/pkg/oci"
    "github.com/containers/image/v5/types"
)

func main() {
    ctx := context.Background()

    opts := oci.ContainerToVirtioOptions{
        ImageRef:       "docker.io/library/alpine:latest",
        OutputPath:     "./alpine-rootfs.img",
        FilesystemType: "ext4",
        Size:           1024 * 1024 * 1024, // 1GB
        Platform: &types.SystemContext{
            OSChoice:           "linux",
            ArchitectureChoice: "arm64",
        },
    }

    device, err := oci.ContainerToVirtioDevice(ctx, opts)
    if err != nil {
        panic(err)
    }

    // Use device in your VM configuration
    _ = device
}
```

### Command Line Tool

Use the included command-line tool for quick conversions:

```bash
# Convert Alpine Linux to a 512MB ext4 rootfs
./gow run ./cmd/oci-to-virtio -image=alpine:latest -size=512 -output=./alpine.img

# Convert Ubuntu with XFS filesystem
./gow run ./cmd/oci-to-virtio \
    -image=ubuntu:22.04 \
    -fs=xfs \
    -size=2048 \
    -platform=linux/amd64 \
    -output=./ubuntu.img

# Create read-only rootfs
./gow run ./cmd/oci-to-virtio \
    -image=nginx:alpine \
    -readonly=true \
    -output=./nginx-readonly.img
```

## Configuration Options

### ContainerToVirtioOptions

| Field            | Type                   | Description               | Default       |
| ---------------- | ---------------------- | ------------------------- | ------------- |
| `ImageRef`       | `string`               | Container image reference | Required      |
| `Platform`       | `*types.SystemContext` | Target platform (OS/arch) | `linux/arm64` |
| `OutputPath`     | `string`               | Output file path          | Required      |
| `FilesystemType` | `string`               | Filesystem type           | `ext4`        |
| `Size`           | `int64`                | Image size in bytes       | `1GB`         |
| `ReadOnly`       | `bool`                 | Create read-only device   | `false`       |

### Supported Filesystems

-   **ext4** (default): Modern Linux filesystem with journaling
-   **ext3**: Legacy Linux filesystem with journaling
-   **xfs**: High-performance filesystem for large files

### Platform Support

Specify target platform using the standard format:

```go
Platform: &types.SystemContext{
    OSChoice:           "linux",
    ArchitectureChoice: "arm64", // or "amd64"
}
```

## System Requirements

### Required Tools

The package requires these system tools to be available:

-   `mke2fs` (for ext3/ext4 filesystems)
-   `mkfs.xfs` (for XFS filesystems)
-   `mount`/`umount` (for filesystem population)
-   `cp` (for file copying)

### Permissions

Some operations may require elevated privileges:

-   **Mounting filesystems**: Requires `sudo` access
-   **Loop device access**: May need permissions for `/dev/loop*`

For production use, consider running in a container with appropriate capabilities.

## Integration with EC1

This package integrates seamlessly with EC1's VM management:

```go
// Convert container to virtio device
device, err := oci.ContainerToVirtioDevice(ctx, opts)
if err != nil {
    return err
}

// Add to VM configuration
vmOpts := vmm.NewVMOptions{
    Vcpus:   2,
    Memory:  strongunits.GiB(4).ToBytes(),
    Devices: []virtio.VirtioDevice{device},
}

// Create and run VM
vm, err := vmm.RunVirtualMachine(ctx, hypervisor, vmiProvider, 2, memoryBytes)
```

## Performance Considerations

### Image Size

-   **Smaller images boot faster**: Use minimal base images like Alpine
-   **Size vs. functionality**: Balance image size with required tools
-   **Layer optimization**: Multi-stage builds reduce final image size

### Filesystem Choice

-   **ext4**: Best general-purpose choice, good performance
-   **xfs**: Better for large files and high I/O workloads
-   **ext3**: Use only for legacy compatibility

### Caching

The package automatically caches:

-   **Downloaded layers**: Avoid re-downloading unchanged layers
-   **Extracted content**: Speed up repeated conversions

## Error Handling

Common errors and solutions:

### Permission Denied

```
Error: permission denied mounting filesystem
Solution: Run with sudo or in privileged container
```

### Missing Tools

```
Error: command failed: mke2fs
Solution: Install e2fsprogs package (apt-get install e2fsprogs)
```

### Network Issues

```
Error: failed to pull image
Solution: Check network connectivity and registry access
```

## Examples

### Converting Different Base Images

```bash
# Minimal Alpine (5MB base)
./gow run ./cmd/oci-to-virtio -image=alpine:latest -size=256

# Ubuntu LTS (30MB base)
./gow run ./cmd/oci-to-virtio -image=ubuntu:22.04 -size=1024

# Distroless (2MB base)
./gow run ./cmd/oci-to-virtio -image=gcr.io/distroless/static -size=128
```

### Application Containers

```bash
# Web server
./gow run ./cmd/oci-to-virtio -image=nginx:alpine -size=512

# Database
./gow run ./cmd/oci-to-virtio -image=postgres:alpine -size=2048

# Custom application
./gow run ./cmd/oci-to-virtio -image=myregistry.com/myapp:v1.0 -size=1024
```

## Testing

Run the test suite:

```bash
# Run all tests
./gow test ./pkg/oci/...

# Run with coverage
./gow test -function-coverage ./pkg/oci/...

# Skip integration tests (no network/sudo required)
CI=1 ./gow test ./pkg/oci/...
```

## Contributing

When contributing to this package:

1. **Add tests** for new functionality
2. **Update documentation** for API changes
3. **Consider security** implications of filesystem operations
4. **Test cross-platform** compatibility

## Security Considerations

-   **Image verification**: Consider implementing signature verification
-   **Privilege escalation**: Minimize sudo usage in production
-   **Resource limits**: Set appropriate size limits for images
-   **Network security**: Use private registries for sensitive images

## Future Enhancements

Planned improvements:

-   [ ] **Signature verification** for image authenticity
-   [ ] **Incremental updates** for faster rebuilds
-   [ ] **Compression support** for smaller disk images
-   [ ] **Direct integration** with containerd/CRI-O
-   [ ] **Windows container** support
-   [ ] **Custom init systems** for container-to-VM transition
