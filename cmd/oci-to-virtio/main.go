package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/containers/image/v5/types"

	"github.com/walteh/ec1/pkg/logging"
	"github.com/walteh/ec1/pkg/oci"
)

func main() {
	var (
		imageRef   = flag.String("image", "docker.io/library/alpine:latest", "OCI container image reference")
		outputPath = flag.String("output", "", "Output path for the virtio device image (default: ./container-rootfs.img)")
		fsType     = flag.String("fs", "ext4", "Filesystem type (currently only ext4 supported)")
		sizeMB     = flag.Int64("size", 1024, "Size of the rootfs image in MB")
		platform   = flag.String("platform", "", "Target platform (e.g., linux/arm64, linux/amd64). If empty, host's platform is used.")
		readOnly   = flag.Bool("readonly", false, "Create a read-only virtio device")
	)
	flag.Parse()

	ctx := logging.SetupSlogSimple(context.Background())

	actProcessedOutputPath := *outputPath
	if actProcessedOutputPath == "" {
		wd, err := os.Getwd()
		if err != nil {
			slog.ErrorContext(ctx, "failed to get current working directory", "error", err)
			os.Exit(1)
		}
		actProcessedOutputPath = filepath.Join(wd, "container-rootfs.img")
	}

	slog.InfoContext(ctx, "OCI to Virtio Device Converter",
		"image", *imageRef,
		"output", actProcessedOutputPath,
		"fsType", *fsType,
		"sizeMB", *sizeMB,
		"platform", *platform,
		"readOnly", *readOnly,
	)

	if strings.ToLower(*fsType) != "ext4" {
		slog.ErrorContext(ctx, "unsupported filesystem type", "type", *fsType, "supported", "ext4")
		os.Exit(1)
	}

	var sysCtx *types.SystemContext
	if *platform != "" {
		parts := strings.SplitN(*platform, "/", 2)
		if len(parts) != 2 {
			slog.ErrorContext(ctx, "invalid platform format, expected os/arch (e.g., linux/arm64)", "platform", *platform)
			os.Exit(1)
		}
		sysCtx = &types.SystemContext{
			OSChoice:           parts[0],
			ArchitectureChoice: parts[1],
		}
	} // else, containers/image will use the host default

	opts := oci.ContainerToVirtioOptions{
		ImageRef:       *imageRef,
		Platform:       sysCtx,
		OutputPath:     actProcessedOutputPath,
		FilesystemType: *fsType,
		Size:           *sizeMB * 1024 * 1024, // Convert MB to bytes
		ReadOnly:       *readOnly,
	}

	_, err := oci.ContainerToVirtioDevice(ctx, opts)
	if err != nil {
		slog.ErrorContext(ctx, "failed to convert container to virtio device", "error", err)
		os.Exit(1)
	}

	slog.InfoContext(ctx, "Successfully created virtio device", "path", actProcessedOutputPath)
} 