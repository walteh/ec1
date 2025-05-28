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
	v1 "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/walteh/ec1/pkg/oci"
	"github.com/walteh/ec1/pkg/virtio"
)

func main() {
	var (
		imageRef   = flag.String("image", "", "Container image reference (e.g., docker.io/library/alpine:latest)")
		outputDir  = flag.String("output", "./output", "Output directory for the virtio device")
		mountPoint = flag.String("mount", "", "Mount point for FUSE filesystem (default: output/rootfs)")
		platform   = flag.String("platform", "linux/amd64", "Target platform (e.g., linux/amd64, linux/arm64)")
		readOnly   = flag.Bool("readonly", false, "Create read-only virtio device")
		verbose    = flag.Bool("verbose", false, "Enable verbose logging")
		noCache    = flag.Bool("no-cache", false, "Disable caching (always download fresh)")
		clearCache = flag.Bool("clear-cache", false, "Clear all cached images and exit")
		listCache  = flag.Bool("list-cache", false, "List all cached images and exit")
	)
	flag.Parse()

	ctx := context.Background()

	// Handle cache management commands first
	if *clearCache {
		if err := oci.ClearCache(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "Error clearing cache: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("✅ Cache cleared successfully")
		return
	}

	if *listCache {
		cachedImages, err := oci.ListCachedImages(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error listing cached images: %v\n", err)
			os.Exit(1)
		}
		
		if len(cachedImages) == 0 {
			fmt.Println("No cached images found")
		} else {
			fmt.Printf("Found %d cached images:\n", len(cachedImages))
			for i, entry := range cachedImages {
				fmt.Printf("  %d. %s (%s)\n", i+1, entry.ImageRef, entry.Platform)
				fmt.Printf("     Cached: %s, Expires: %s\n", 
					entry.CachedAt.Format("2006-01-02 15:04:05"),
					entry.ExpiresAt.Format("2006-01-02 15:04:05"))
				fmt.Printf("     Size: %.1f MB\n", float64(entry.Size)/1024/1024)
				fmt.Println()
			}
		}
		return
	}

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
	var device virtio.VirtioDevice
	var metadata *v1.Image
	var err error
	
	if *noCache {
		slog.InfoContext(ctx, "Using non-cached conversion (--no-cache specified)")
		device, metadata, err = oci.ContainerToVirtioDevice(ctx, opts)
	} else {
		slog.InfoContext(ctx, "Using cached conversion")
		device, metadata, err = oci.ContainerToVirtioDeviceCached(ctx, opts)
	}
	
	if err != nil {
		slog.ErrorContext(ctx, "Failed to convert container to virtio device", "error", err)
		os.Exit(1)
	}

	slog.InfoContext(ctx, "Successfully created virtio device",
		"mount_point", *mountPoint,
		"device", fmt.Sprintf("%T", device))

	// Log container metadata
	if metadata != nil {
		slog.InfoContext(ctx, "Container metadata extracted",
			"entrypoint", metadata.Config.Entrypoint,
			"cmd", metadata.Config.Cmd,
			"working_dir", metadata.Config.WorkingDir,
			"user", metadata.Config.User,
			"env_count", len(metadata.Config.Env))
	}

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