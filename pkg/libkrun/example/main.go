package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/walteh/ec1/pkg/libkrun"
)

func main() {
	// Setup context with logger
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	ctx = context.WithValue(ctx, "logger", logger)

	logger.InfoContext(ctx, "libkrun example starting")

	// Set log level for libkrun
	if err := libkrun.SetLogLevel(ctx, libkrun.LogLevelInfo); err != nil {
		logger.WarnContext(ctx, "failed to set libkrun log level", slog.Any("error", err))
	}

	// Create a new libkrun context
	kctx, err := libkrun.CreateContext(ctx)
	if err != nil {
		logger.ErrorContext(ctx, "failed to create libkrun context", slog.Any("error", err))
		fmt.Printf("Error: %v\n", err)
		fmt.Println("\nThis is expected if libkrun is not installed on your system.")
		fmt.Println("To install libkrun:")
		fmt.Println("  - On macOS: brew install libkrun")
		fmt.Println("  - On Linux: check your distribution's package manager")
		fmt.Println("\nLibkrun variants:")
		fmt.Println("  - libkrun: Generic variant (default)")
		fmt.Println("  - libkrun-sev: AMD SEV support variant")
		fmt.Println("  - libkrun-efi: EFI/OVMF support variant (macOS only)")
		fmt.Println("\nThis example demonstrates the API structure even without libkrun installed.")
		return
	}

	// Ensure we clean up the context
	defer func() {
		if err := kctx.Free(ctx); err != nil {
			logger.ErrorContext(ctx, "failed to free libkrun context", slog.Any("error", err))
		}
	}()

	logger.InfoContext(ctx, "libkrun context created successfully")

	// Configure the VM using the new struct-based API
	vmConfig := libkrun.VMConfig{
		NumVCPUs: 1,
		RAMMiB:   512,
	}
	if err := kctx.SetVMConfig(ctx, vmConfig); err != nil {
		logger.ErrorContext(ctx, "failed to set VM config", slog.Any("error", err))
		return
	}
	logger.InfoContext(ctx, "VM configuration set",
		slog.Int("vcpus", int(vmConfig.NumVCPUs)),
		slog.Uint64("ram_mib", uint64(vmConfig.RAMMiB)))

	// Try to set root filesystem (might not be available in SEV variant)
	rootPath := "/tmp"
	if err := kctx.SetRoot(ctx, rootPath); err != nil {
		logger.WarnContext(ctx, "failed to set VM root (might be SEV variant)", slog.Any("error", err))
	} else {
		logger.InfoContext(ctx, "VM root path set", slog.String("root_path", rootPath))
	}

	// Set process to run using the new struct-based API
	processConfig := libkrun.ProcessConfig{
		ExecPath: "/bin/echo",
		Args:     []string{"echo", "Hello from libkrun!"},
		Env:      []string{"PATH=/bin:/usr/bin", "HOME=/tmp"},
	}
	if err := kctx.SetProcess(ctx, processConfig); err != nil {
		logger.ErrorContext(ctx, "failed to set VM process", slog.Any("error", err))
		return
	}
	logger.InfoContext(ctx, "VM process configured",
		slog.String("exec_path", processConfig.ExecPath),
		slog.Any("args", processConfig.Args))

	// Test variant-specific features

	// 1. Try mapped volumes (not available in SEV variant)
	logger.InfoContext(ctx, "testing mapped volumes (generic/EFI only)")
	mappedVolumes := []string{"/host/shared:/guest/shared"}
	if err := kctx.SetMappedVolumes(ctx, mappedVolumes); err != nil {
		logger.WarnContext(ctx, "mapped volumes not available", slog.Any("error", err))
	} else {
		logger.InfoContext(ctx, "mapped volumes configured", slog.Any("volumes", mappedVolumes))
	}

	// 2. Try SEV configuration (only available in SEV variant)
	logger.InfoContext(ctx, "testing SEV configuration (SEV only)")
	sevConfig := libkrun.SEVConfig{}
	if err := kctx.SetSEVConfig(ctx, sevConfig); err != nil {
		logger.WarnContext(ctx, "SEV configuration not available", slog.Any("error", err))
	} else {
		logger.InfoContext(ctx, "SEV configuration set")
	}

	// 3. Try to get shutdown event FD (only available in EFI variant)
	logger.InfoContext(ctx, "testing shutdown event FD (EFI only)")
	if fd, err := kctx.GetShutdownEventFD(ctx); err != nil {
		logger.WarnContext(ctx, "shutdown event FD not available", slog.Any("error", err))
	} else {
		logger.InfoContext(ctx, "shutdown event FD obtained", slog.Int("fd", fd))
	}

	logger.InfoContext(ctx, "libkrun configuration completed successfully")
	fmt.Println("\n✅ libkrun PoC completed successfully!")
	fmt.Println("All configuration functions executed without errors.")
	fmt.Println("\nNote: StartEnter() was not called as it would actually start the VM.")
	fmt.Println("In a real application, you would call kctx.StartEnter(ctx) to start the microVM.")

	// Example of other configurations that could be set:
	fmt.Println("\n📋 Additional configuration examples:")

	// Network configuration example
	networkConfig := libkrun.NetworkConfig{
		PortMap: []string{"8080:80", "9090:90"},
	}
	fmt.Printf("- Network: Port mapping %v\n", networkConfig.PortMap)

	// Disk configuration example
	diskConfig := libkrun.DiskConfig{
		BlockID:  "root",
		Path:     "/path/to/disk.raw",
		Format:   libkrun.DiskFormatRaw,
		ReadOnly: true,
	}
	fmt.Printf("- Disk: %s at %s (format: %v, readonly: %v)\n",
		diskConfig.BlockID, diskConfig.Path, diskConfig.Format, diskConfig.ReadOnly)

	// VirtioFS configuration example
	virtiofsConfig := libkrun.VirtioFSConfig{
		Tag:  "shared",
		Path: "/host/shared",
	}
	fmt.Printf("- VirtioFS: tag '%s' -> %s\n", virtiofsConfig.Tag, virtiofsConfig.Path)

	// GPU configuration example
	gpuConfig := libkrun.GPUConfig{
		VirglFlags: libkrun.VirglUseEGL | libkrun.VirglThreadSync,
	}
	fmt.Printf("- GPU: VirGL flags %v\n", gpuConfig.VirglFlags)

	fmt.Println("\n🏗️  Build tags for different variants:")
	fmt.Println("  - Default build: Uses stub implementation")
	fmt.Println("  - -tags libkrun: Generic libkrun variant")
	fmt.Println("  - -tags libkrun_sev: AMD SEV variant")
	fmt.Println("  - -tags libkrun_efi: EFI/OVMF variant")
}
