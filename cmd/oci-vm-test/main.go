package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/containers/common/pkg/strongunits"

	"github.com/walteh/ec1/pkg/images/oci"
	"github.com/walteh/ec1/pkg/logging"
	"github.com/walteh/ec1/pkg/vmm"
	"github.com/walteh/ec1/pkg/vmm/vf"
)

func parseMemory(memStr string) (strongunits.B, error) {
	// Simple memory parsing - convert common formats to bytes
	memStr = strings.ToUpper(strings.TrimSpace(memStr))
	
	if strings.HasSuffix(memStr, "GIB") || strings.HasSuffix(memStr, "GB") {
		numStr := strings.TrimSuffix(strings.TrimSuffix(memStr, "GIB"), "GB")
		num, err := strconv.ParseFloat(numStr, 64)
		if err != nil {
			return 0, err
		}
		return strongunits.GiB(num).ToBytes(), nil
	}
	
	if strings.HasSuffix(memStr, "MIB") || strings.HasSuffix(memStr, "MB") {
		numStr := strings.TrimSuffix(strings.TrimSuffix(memStr, "MIB"), "MB")
		num, err := strconv.ParseFloat(numStr, 64)
		if err != nil {
			return 0, err
		}
		return strongunits.MiB(num).ToBytes(), nil
	}
	
	// Default to bytes
	num, err := strconv.ParseUint(memStr, 10, 64)
	if err != nil {
		return 0, err
	}
	return strongunits.B(num), nil
}

func main() {
	var (
		imageRef    = flag.String("image", "docker.io/library/alpine:latest", "OCI container image reference")
		kernelImage = flag.String("kernel", "", "Kernel image reference (optional)")
		vcpus       = flag.Uint("vcpus", 2, "Number of virtual CPUs")
		memory      = flag.String("memory", "2GiB", "Amount of memory")
		timeout     = flag.Duration("timeout", 5*time.Minute, "VM run timeout")
	)
	flag.Parse()

	// Parse memory
	memoryBytes, err := parseMemory(*memory)
	if err != nil {
		slog.Error("invalid memory format", "memory", *memory, "error", err)
		os.Exit(1)
	}

	// Set up logging
	ctx := context.Background()
	ctx = logging.SetupSlogSimple(ctx)

	slog.InfoContext(ctx, "EC1 OCI Container VM Test", 
		"image", *imageRef,
		"kernel", *kernelImage,
		"vcpus", *vcpus,
		"memory", memoryBytes.String(),
	)

	// Create OCI provider
	ociProvider := oci.NewOCIProvider(*imageRef, *kernelImage)

	// Create hypervisor (using VF for macOS)
	hypervisor := vf.NewHypervisor()

	// Run the VM with OCI container as rootfs
	slog.InfoContext(ctx, "starting VM with OCI container rootfs")
	
	runCtx, cancel := context.WithTimeout(ctx, *timeout)
	defer cancel()

	runningVM, err := vmm.RunVirtualMachine(runCtx, hypervisor, ociProvider, *vcpus, memoryBytes)
	if err != nil {
		slog.ErrorContext(ctx, "failed to run VM", "error", err)
		os.Exit(1)
	}

	slog.InfoContext(ctx, "VM started successfully", "vm_id", runningVM.VM().ID())

	// Wait for VM to complete or timeout
	select {
	case err := <-runningVM.WaitOnVmStopped():
		if err != nil {
			slog.ErrorContext(ctx, "VM exited with error", "error", err)
			os.Exit(1)
		}
		slog.InfoContext(ctx, "VM completed successfully")
	case <-runCtx.Done():
		slog.WarnContext(ctx, "VM test timed out")
		os.Exit(1)
	}
} 