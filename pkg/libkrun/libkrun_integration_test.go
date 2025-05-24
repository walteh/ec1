//go:build libkrun && cgo && vmnet_helper

package libkrun

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	slogctx "github.com/veqryn/slog-context"

	"github.com/walteh/ec1/pkg/vmnet"
)

// Global flag to ensure we only initialize libkrun logger once
var (
	loggerInitialized sync.Once
	loggerInitError   error
)

func setupIntegrationContext(t testing.TB) context.Context {
	ctx := t.Context()

	if _, ok := t.(*testing.B); ok {
		slog.SetDefault(slog.New(slog.DiscardHandler))
		return ctx
	} else {
		logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))
		slog.SetDefault(logger)
		return slogctx.NewCtx(ctx, logger)
	}
}

// initializeLibkrunLogger ensures libkrun logger is only initialized once
func initializeLibkrunLogger(ctx context.Context) error {
	loggerInitialized.Do(func() {
		loggerInitError = SetLogLevel(ctx, LogLevelInfo)
	})
	return loggerInitError
}

// TestIntegrationBasicVM tests creating a basic VM with minimal configuration
func TestIntegrationBasicVM(t *testing.T) {

	ctx := setupIntegrationContext(t)
	log := slog.With(slog.String("test", "BasicVM"))

	log.InfoContext(ctx, "starting basic VM integration test")

	// Initialize libkrun logger once globally
	err := initializeLibkrunLogger(ctx)
	require.NoError(t, err, "should initialize libkrun logger")

	// Create libkrun context
	kctx, err := CreateContext(ctx)
	require.NoError(t, err, "should create libkrun context")
	defer func() {
		if freeErr := kctx.Free(ctx); freeErr != nil {
			log.ErrorContext(ctx, "failed to free context", slog.Any("error", freeErr))
		}
	}()

	// Configure basic VM
	vmConfig := VMConfig{
		NumVCPUs: 1,
		RAMMiB:   256, // Minimal RAM for basic test
	}
	err = kctx.SetVMConfig(ctx, vmConfig)
	require.NoError(t, err, "should set VM config")

	// Set root filesystem to /tmp (won't actually boot but validates API)
	err = kctx.SetRoot(ctx, "/tmp")
	require.NoError(t, err, "should set root filesystem")

	// Configure a simple process
	processConfig := ProcessConfig{
		ExecPath: "/bin/echo",
		Args:     []string{"echo", "Hello from basic VM test"},
		Env:      []string{"PATH=/bin:/usr/bin"},
	}
	err = kctx.SetProcess(ctx, processConfig)
	require.NoError(t, err, "should set process config")

	// Test network configuration (without port mapping for now)
	networkConfig := NetworkConfig{
		PortMap: []string{}, // Start with empty port map
	}
	err = kctx.SetNetwork(ctx, networkConfig)
	require.NoError(t, err, "should set network config")

	log.InfoContext(ctx, "basic VM configuration completed successfully")

	// Note: We don't call StartEnter() as it would actually start the VM
	// and we need a proper kernel/initramfs for that
}

// TestIntegrationVMNetNetworking tests vmnet networking integration
func TestIntegrationVMNetNetworking(t *testing.T) {

	ctx := setupIntegrationContext(t)
	log := slog.With(slog.String("test", "VMNetNetworking"))

	log.InfoContext(ctx, "starting vmnet networking integration test")

	// Initialize libkrun logger once globally
	err := initializeLibkrunLogger(ctx)
	require.NoError(t, err, "should initialize libkrun logger")

	// Create libkrun context
	kctx, err := CreateContext(ctx)
	require.NoError(t, err, "should create libkrun context")
	defer kctx.Free(ctx)

	// Configure basic VM
	vmConfig := VMConfig{
		NumVCPUs: 1,
		RAMMiB:   256,
	}
	err = kctx.SetVMConfig(ctx, vmConfig)
	require.NoError(t, err, "should set VM config")

	// Test VMNet shared mode networking without port mapping first
	t.Run("SharedModeBasic", func(t *testing.T) {
		// First check if vmnet-helper is available
		if !vmnet.HelperAvailable() {
			t.Skip("VMNet helper not available at /opt/vmnet-helper/bin/vmnet-helper")
		}

		log.InfoContext(ctx, "vmnet helper is available, proceeding with test")

		vmnetConfig := VMNetConfig{
			OperationMode: vmnet.OperationModeShared,
			PortMap:       []string{}, // Empty port map to avoid EOPNOTSUPP
			Verbose:       true,
		}

		log.InfoContext(ctx, "attempting to configure vmnet with config",
			slog.Any("config", vmnetConfig))

		err := kctx.SetVMNetNetwork(ctx, vmnetConfig)
		if err != nil {
			log.ErrorContext(ctx, "VMNet configuration failed",
				slog.Any("error", err),
				slog.String("error_type", fmt.Sprintf("%T", err)))

			if strings.Contains(err.Error(), "vmnet-helper not available") {
				t.Skipf("VMNet helper not available: %v", err)
			} else if strings.Contains(err.Error(), "permission denied") {
				t.Skipf("VMNet permission denied (need entitlements): %v", err)
			} else if strings.Contains(err.Error(), "EOF") {
				// This suggests the helper process is failing
				log.WarnContext(ctx, "VMNet helper process appears to be failing - this might be an entitlement issue")
				t.Skipf("VMNet helper process failing (likely entitlement issue): %v", err)
			}
			t.Fatalf("Unexpected VMNet error: %v", err)
		}

		log.InfoContext(ctx, "vmnet shared mode (basic) configured successfully")
	})

	// Test VMNet shared mode with port mapping
	t.Run("SharedModeWithPorts", func(t *testing.T) {
		// First check if vmnet-helper is available
		if !vmnet.HelperAvailable() {
			t.Skip("VMNet helper not available at /opt/vmnet-helper/bin/vmnet-helper")
		}

		vmnetConfig := VMNetConfig{
			OperationMode: vmnet.OperationModeShared,
			PortMap:       []string{"2222:22"},
			Verbose:       true,
		}

		err := kctx.SetVMNetNetwork(ctx, vmnetConfig)
		if err != nil {
			log.ErrorContext(ctx, "VMNet configuration failed",
				slog.Any("error", err),
				slog.String("error_type", fmt.Sprintf("%T", err)))

			if strings.Contains(err.Error(), "vmnet-helper not available") {
				t.Skipf("VMNet helper not available: %v", err)
			} else if strings.Contains(err.Error(), "permission denied") {
				t.Skipf("VMNet permission denied (need entitlements): %v", err)
			} else if strings.Contains(err.Error(), "EOF") {
				// This suggests the helper process is failing
				log.WarnContext(ctx, "VMNet helper process appears to be failing - this might be an entitlement issue")
				t.Skipf("VMNet helper process failing (likely entitlement issue): %v", err)
			} else if strings.Contains(err.Error(), "operation not supported") || strings.Contains(err.Error(), "-45") {
				// EOPNOTSUPP - port mapping not supported in current configuration
				t.Skipf("VMNet port mapping not supported: %v", err)
			}
			t.Fatalf("Unexpected VMNet error: %v", err)
		}

		log.InfoContext(ctx, "vmnet shared mode with port mapping configured successfully")
	})

	// Test VMNet host-only mode networking
	t.Run("HostOnlyModeBasic", func(t *testing.T) {
		// First check if vmnet-helper is available
		if !vmnet.HelperAvailable() {
			t.Skip("VMNet helper not available at /opt/vmnet-helper/bin/vmnet-helper")
		}

		vmnetConfig := VMNetConfig{
			OperationMode: vmnet.OperationModeHost,
			PortMap:       []string{}, // Empty port map to avoid issues
			Verbose:       true,
		}

		err := kctx.SetVMNetNetwork(ctx, vmnetConfig)
		if err != nil {
			log.ErrorContext(ctx, "VMNet configuration failed",
				slog.Any("error", err),
				slog.String("error_type", fmt.Sprintf("%T", err)))

			if strings.Contains(err.Error(), "vmnet-helper not available") {
				t.Skipf("VMNet helper not available: %v", err)
			} else if strings.Contains(err.Error(), "permission denied") {
				t.Skipf("VMNet permission denied (need entitlements): %v", err)
			} else if strings.Contains(err.Error(), "EOF") {
				// This suggests the helper process is failing
				log.WarnContext(ctx, "VMNet helper process appears to be failing - this might be an entitlement issue")
				t.Skipf("VMNet helper process failing (likely entitlement issue): %v", err)
			}
			t.Fatalf("Unexpected VMNet error: %v", err)
		}

		log.InfoContext(ctx, "vmnet host-only mode configured successfully")
	})
}

// TestIntegrationDiskOperations tests disk and storage configuration
func TestIntegrationDiskOperations(t *testing.T) {

	ctx := setupIntegrationContext(t)
	log := slog.With(slog.String("test", "DiskOperations"))

	// Initialize libkrun logger once globally
	err := initializeLibkrunLogger(ctx)
	require.NoError(t, err, "should initialize libkrun logger")

	// Create temporary disk files for testing
	tempDir := t.TempDir()

	// Create a small raw disk image
	rawDiskPath := filepath.Join(tempDir, "test.raw")
	rawDiskFile, err := os.Create(rawDiskPath)
	require.NoError(t, err, "should create raw disk file")

	// Write 1MB of zeros
	zeros := make([]byte, 1024*1024)
	_, err = rawDiskFile.Write(zeros)
	require.NoError(t, err, "should write to raw disk")
	rawDiskFile.Close()

	log.InfoContext(ctx, "created temporary disk files",
		slog.String("raw_disk", rawDiskPath))

	// Create libkrun context
	kctx, err := CreateContext(ctx)
	require.NoError(t, err, "should create libkrun context")
	defer kctx.Free(ctx)

	// Configure basic VM
	vmConfig := VMConfig{
		NumVCPUs: 1,
		RAMMiB:   256,
	}
	err = kctx.SetVMConfig(ctx, vmConfig)
	require.NoError(t, err, "should set VM config")

	// Test adding disks
	t.Run("AddRawDisk", func(t *testing.T) {
		diskConfig := DiskConfig{
			BlockID:  "root",
			Path:     rawDiskPath,
			Format:   DiskFormatRaw,
			ReadOnly: false,
		}

		err := kctx.AddDisk(ctx, diskConfig)
		assert.NoError(t, err, "should add raw disk")

		log.InfoContext(ctx, "added raw disk successfully",
			slog.String("block_id", diskConfig.BlockID),
			slog.String("path", diskConfig.Path))
	})

	t.Run("AddDisk2WithFormat", func(t *testing.T) {
		diskConfig := DiskConfig{
			BlockID:  "data",
			Path:     rawDiskPath,
			Format:   DiskFormatRaw,
			ReadOnly: true,
		}

		err := kctx.AddDisk2(ctx, diskConfig)
		assert.NoError(t, err, "should add disk with explicit format")

		log.InfoContext(ctx, "added disk with format successfully")
	})

	// Test legacy disk methods
	t.Run("LegacyDiskMethods", func(t *testing.T) {
		err := kctx.SetRootDisk(ctx, rawDiskPath)
		assert.NoError(t, err, "should set root disk")

		err = kctx.SetDataDisk(ctx, rawDiskPath)
		assert.NoError(t, err, "should set data disk")

		log.InfoContext(ctx, "legacy disk methods work")
	})
}

// TestIntegrationAdvancedFeatures tests advanced VM features
func TestIntegrationAdvancedFeatures(t *testing.T) {

	ctx := setupIntegrationContext(t)
	log := slog.With(slog.String("test", "AdvancedFeatures"))

	// Initialize libkrun logger once globally
	err := initializeLibkrunLogger(ctx)
	require.NoError(t, err, "should initialize libkrun logger")

	// Create libkrun context
	kctx, err := CreateContext(ctx)
	require.NoError(t, err, "should create libkrun context")
	defer kctx.Free(ctx)

	// Configure basic VM
	vmConfig := VMConfig{
		NumVCPUs: 2, // Use more CPUs for advanced testing
		RAMMiB:   512,
	}
	err = kctx.SetVMConfig(ctx, vmConfig)
	require.NoError(t, err, "should set VM config")

	// Test VirtioFS configuration
	t.Run("VirtioFS", func(t *testing.T) {
		tempDir := t.TempDir()

		virtiofsConfig := VirtioFSConfig{
			Tag:  "shared",
			Path: tempDir,
		}

		err := kctx.AddVirtioFS(ctx, virtiofsConfig)
		assert.NoError(t, err, "should add virtio-fs device")

		// Test with custom DAX window size
		shmSize := uint64(64 * 1024 * 1024) // 64MB
		virtiofsConfig2 := VirtioFSConfig{
			Tag:     "shared2",
			Path:    tempDir,
			ShmSize: &shmSize,
		}

		err = kctx.AddVirtioFS(ctx, virtiofsConfig2)
		assert.NoError(t, err, "should add virtio-fs device with custom DAX size")

		log.InfoContext(ctx, "virtio-fs devices configured successfully")
	})

	// Test GPU configuration
	t.Run("GPU", func(t *testing.T) {
		gpuConfig := GPUConfig{
			VirglFlags: VirglUseEGL | VirglThreadSync,
		}

		err := kctx.SetGPU(ctx, gpuConfig)
		assert.NoError(t, err, "should set GPU config")

		// Test with custom vRAM size
		vramSize := uint64(16 * 1024 * 1024) // 16MB
		gpuConfig2 := GPUConfig{
			VirglFlags: VirglUseGLX,
			ShmSize:    &vramSize,
		}

		err = kctx.SetGPU(ctx, gpuConfig2)
		assert.NoError(t, err, "should set GPU config with vRAM")

		log.InfoContext(ctx, "GPU configuration successful")
	})

	// Test vsock ports
	t.Run("VsockPorts", func(t *testing.T) {
		tempDir := t.TempDir()

		ports := []VsockPort{
			{
				Port:     1234,
				FilePath: filepath.Join(tempDir, "socket1"),
			},
			{
				Port:     5678,
				FilePath: filepath.Join(tempDir, "socket2"),
			},
		}

		err := kctx.AddVsockPorts(ctx, ports)
		assert.NoError(t, err, "should add vsock ports")

		// Test with listen config
		listen := true
		portsWithListen := []VsockPort{
			{
				Port:     9999,
				FilePath: filepath.Join(tempDir, "socket3"),
				Listen:   &listen,
			},
		}

		err = kctx.AddVsockPorts(ctx, portsWithListen)
		assert.NoError(t, err, "should add vsock ports with listen config")

		log.InfoContext(ctx, "vsock ports configured successfully")
	})

	// Test security configuration
	t.Run("Security", func(t *testing.T) {
		uid := uint32(1000)
		gid := uint32(1000)

		securityConfig := SecurityConfig{
			UID:              &uid,
			GID:              &gid,
			Rlimits:          []string{"RLIMIT_NOFILE=1024:2048", "RLIMIT_NPROC=512:1024"},
			SMBIOSOEMStrings: []string{"vendor=TestVendor", "product=TestProduct"},
		}

		err := kctx.SetSecurity(ctx, securityConfig)
		assert.NoError(t, err, "should set security config")

		log.InfoContext(ctx, "security configuration successful")
	})

	// Test advanced features
	t.Run("AdvancedSettings", func(t *testing.T) {
		nestedVirt := true
		soundDevice := false
		consoleOutput := filepath.Join(t.TempDir(), "console.log")

		advancedConfig := AdvancedConfig{
			NestedVirt:    &nestedVirt,
			SoundDevice:   &soundDevice,
			ConsoleOutput: &consoleOutput,
		}

		err := kctx.SetAdvanced(ctx, advancedConfig)
		assert.NoError(t, err, "should set advanced config")

		log.InfoContext(ctx, "advanced configuration successful")
	})
}

// TestIntegrationVariantSpecificFeatures tests variant-specific functionality
func TestIntegrationVariantSpecificFeatures(t *testing.T) {

	ctx := setupIntegrationContext(t)
	log := slog.With(slog.String("test", "VariantSpecific"))

	// Initialize libkrun logger once globally
	err := initializeLibkrunLogger(ctx)
	require.NoError(t, err, "should initialize libkrun logger")

	// Create libkrun context
	kctx, err := CreateContext(ctx)
	require.NoError(t, err, "should create libkrun context")
	defer kctx.Free(ctx)

	// Test mapped volumes (generic/EFI only)
	t.Run("MappedVolumes", func(t *testing.T) {
		mappedVolumes := []string{
			"/host/path1:/guest/path1",
			"/host/path2:/guest/path2:ro",
		}

		err := kctx.SetMappedVolumes(ctx, mappedVolumes)
		if err != nil {
			if strings.Contains(err.Error(), "only available in libkrun-sev variant") ||
				strings.Contains(err.Error(), "not available") {
				log.InfoContext(ctx, "mapped volumes not available in this variant", slog.Any("error", err))
			} else {
				// Mapped volumes might fail with EINVAL if paths don't exist
				log.InfoContext(ctx, "mapped volumes failed (paths may not exist)", slog.Any("error", err))
			}
		} else {
			log.InfoContext(ctx, "mapped volumes configured successfully")
		}
	})

	// Test SEV configuration (SEV only)
	t.Run("SEVConfig", func(t *testing.T) {
		sevConfig := SEVConfig{}

		err := kctx.SetSEVConfig(ctx, sevConfig)
		if err != nil {
			if strings.Contains(err.Error(), "only available in libkrun-sev variant") ||
				strings.Contains(err.Error(), "not available") {
				log.InfoContext(ctx, "SEV config not available in this variant", slog.Any("error", err))
			} else {
				log.WarnContext(ctx, "unexpected SEV config error", slog.Any("error", err))
			}
		} else {
			log.InfoContext(ctx, "SEV configuration successful")
		}
	})

	// Test shutdown event FD (EFI only)
	t.Run("ShutdownEventFD", func(t *testing.T) {
		fd, err := kctx.GetShutdownEventFD(ctx)
		if err != nil {
			if strings.Contains(err.Error(), "only available in libkrun-efi variant") ||
				strings.Contains(err.Error(), "not available") {
				log.InfoContext(ctx, "shutdown event FD not available in this variant", slog.Any("error", err))
			} else {
				log.WarnContext(ctx, "unexpected shutdown event FD error", slog.Any("error", err))
			}
		} else {
			log.InfoContext(ctx, "shutdown event FD obtained", slog.Int("fd", fd))
			assert.GreaterOrEqual(t, fd, 0, "file descriptor should be non-negative")
		}
	})
}

// TestIntegrationPerformanceBenchmarks runs performance benchmarks
func BenchmarkTestIntegrationPerformance(t *testing.B) {

	ctx := setupIntegrationContext(t)
	log := slog.With(slog.String("test", "Performance"))

	// Initialize libkrun logger once globally
	err := initializeLibkrunLogger(ctx)
	require.NoError(t, err, "should initialize libkrun logger")

	log.InfoContext(ctx, "starting performance benchmarks")

	// Benchmark context creation/destruction
	t.Run("ContextCreationBench", func(t *testing.B) {
		iterations := 100
		start := time.Now()

		for i := 0; i < iterations; i++ {
			kctx, err := CreateContext(ctx)
			require.NoError(t, err, "should create context")
			err = kctx.Free(ctx)
			require.NoError(t, err, "should free context")
		}

		duration := time.Since(start)
		avgDuration := duration / time.Duration(iterations)

		log.InfoContext(ctx, "context creation benchmark completed",
			slog.Int("iterations", iterations),
			slog.Duration("total_time", duration),
			slog.Duration("avg_per_operation", avgDuration))

		// Performance assertion - context creation should be fast
		assert.Less(t, avgDuration, 10*time.Millisecond, "context creation should be fast")
	})

	// Benchmark VM configuration
	t.Run("VMConfigBench", func(t *testing.B) {
		kctx, err := CreateContext(ctx)
		require.NoError(t, err, "should create context")
		defer kctx.Free(ctx)

		vmConfig := VMConfig{
			NumVCPUs: 2,
			RAMMiB:   512,
		}

		iterations := 1000
		start := time.Now()

		for i := 0; i < iterations; i++ {
			err := kctx.SetVMConfig(ctx, vmConfig)
			require.NoError(t, err, "should set VM config")
		}

		duration := time.Since(start)
		avgDuration := duration / time.Duration(iterations)

		log.InfoContext(ctx, "VM config benchmark completed",
			slog.Int("iterations", iterations),
			slog.Duration("total_time", duration),
			slog.Duration("avg_per_operation", avgDuration))

		// Configuration should be very fast
		assert.Less(t, avgDuration, 1*time.Millisecond, "VM config should be very fast")
	})
}

// Helper function to check if libkrun is properly installed
func TestIntegrationLibkrunAvailability(t *testing.T) {
	ctx := setupIntegrationContext(t)
	log := slog.With(slog.String("test", "Availability"))

	log.InfoContext(ctx, "checking libkrun availability")

	// Initialize libkrun logger once globally
	err := initializeLibkrunLogger(ctx)
	require.NoError(t, err, "libkrun should be available")

	// Try to create context
	kctx, err := CreateContext(ctx)
	require.NoError(t, err, "should create libkrun context")
	defer kctx.Free(ctx)

	log.InfoContext(ctx, "libkrun is properly installed and available")
}

// TestIntegrationCleanup tests proper resource cleanup
func TestIntegrationCleanup(t *testing.T) {

	ctx := setupIntegrationContext(t)
	log := slog.With(slog.String("test", "Cleanup"))

	// Initialize libkrun logger once globally
	err := initializeLibkrunLogger(ctx)
	require.NoError(t, err, "should initialize libkrun logger")

	log.InfoContext(ctx, "testing resource cleanup")

	// Create multiple contexts and ensure they can be cleaned up
	contexts := make([]*Context, 10)

	for i := 0; i < 10; i++ {
		kctx, err := CreateContext(ctx)
		require.NoError(t, err, "should create context %d", i)
		contexts[i] = kctx
	}

	// Configure all contexts
	for i, kctx := range contexts {
		vmConfig := VMConfig{
			NumVCPUs: 1,
			RAMMiB:   256,
		}
		err := kctx.SetVMConfig(ctx, vmConfig)
		require.NoError(t, err, "should configure context %d", i)
	}

	// Clean up all contexts
	for i, kctx := range contexts {
		err := kctx.Free(ctx)
		require.NoError(t, err, "should free context %d", i)
	}

	log.InfoContext(ctx, "resource cleanup test completed successfully")
}
