package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/containers/image/v5/types"

	"github.com/walteh/ec1/pkg/oci"
)

func main() {
	var (
		imageRef   = flag.String("image", "", "Container image reference (e.g., docker.io/library/alpine:latest)")
		outputDir  = flag.String("output", "./output", "Output directory for the virtio device")
		mountPoint = flag.String("mount", "", "Mount point for FUSE filesystem (default: output/rootfs)")
		platform   = flag.String("platform", "linux/amd64", "Target platform (e.g., linux/amd64, linux/arm64)")
		readOnly   = flag.Bool("readonly", false, "Create read-only virtio device")
		verbose    = flag.Bool("verbose", false, "Enable verbose logging")
	)
	flag.Parse()

	if *imageRef == "" {
		fmt.Fprintf(os.Stderr, "Error: -image flag is required\n")
		flag.Usage()
		os.Exit(1)
	}

	// Setup logging
	logLevel := slog.LevelInfo
	if *verbose {
		logLevel = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))
	slog.SetDefault(logger)

	ctx := context.Background()

	// Parse platform
	var osChoice, archChoice string
	switch *platform {
	case "linux/amd64":
		osChoice, archChoice = "linux", "amd64"
	case "linux/arm64":
		osChoice, archChoice = "linux", "arm64"
	default:
		fmt.Fprintf(os.Stderr, "Error: unsupported platform %s (supported: linux/amd64, linux/arm64)\n", *platform)
		os.Exit(1)
	}

	// Set default mount point if not specified
	if *mountPoint == "" {
		*mountPoint = filepath.Join(*outputDir, "rootfs")
	}

	// Configure options
	opts := oci.ContainerToVirtioOptions{
		ImageRef:   *imageRef,
		OutputDir:  *outputDir,
		MountPoint: *mountPoint,
		Platform: &types.SystemContext{
			OSChoice:           osChoice,
			ArchitectureChoice: archChoice,
		},
		ReadOnly: *readOnly,
	}

	slog.InfoContext(ctx, "Converting OCI container to virtio device",
		"image", *imageRef,
		"platform", *platform,
		"output_dir", *outputDir,
		"mount_point", *mountPoint,
		"read_only", *readOnly)

	// Convert container to virtio device
	device, err := oci.ContainerToVirtioDevice(ctx, opts)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to convert container to virtio device", "error", err)
		os.Exit(1)
	}

	slog.InfoContext(ctx, "Successfully created virtio device",
		"mount_point", *mountPoint,
		"device", fmt.Sprintf("%T", device))

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	fmt.Printf("✅ Container successfully mounted at: %s\n", *mountPoint)
	fmt.Printf("📁 You can now explore the container filesystem at the mount point\n")
	fmt.Printf("🔄 Press Ctrl+C to unmount and exit\n\n")

	// Wait for signal
	<-sigChan
	fmt.Printf("\n🛑 Received shutdown signal, cleaning up...\n")

	slog.InfoContext(ctx, "Shutting down gracefully")
} 