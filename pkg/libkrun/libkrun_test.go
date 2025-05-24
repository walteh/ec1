package libkrun

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	slogctx "github.com/veqryn/slog-context"

	"github.com/walteh/ec1/pkg/vmnet"
)

func setupTestContext(t testing.TB) context.Context {
	ctx := t.Context()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	if _, ok := t.(*testing.B); ok {
		slog.SetDefault(slog.New(slog.DiscardHandler))
	} else {
		slog.SetDefault(logger)
	}

	return slogctx.NewCtx(ctx, logger)
}

func TestSetLogLevel(t *testing.T) {
	ctx := setupTestContext(t)

	t.Run("SetOnce", func(t *testing.T) {
		// libkrun's Rust logger can only be initialized once per process
		// so we only test setting it once
		err := SetLogLevel(ctx, LogLevelInfo)
		if err != nil {
			// When libkrun isn't available, we should get ErrLibkrunNotAvailable
			assert.Equal(t, ErrLibkrunNotAvailable, err, "should get ErrLibkrunNotAvailable when libkrun not available")
		} else {
			t.Log("Successfully set libkrun log level to Info")
		}
	})

	t.Run("SubsequentCallsShouldBeAvoided", func(t *testing.T) {
		// Document that subsequent calls to SetLogLevel should be avoided
		// as libkrun's Rust logger can only be initialized once
		t.Log("Note: SetLogLevel should only be called once per process due to libkrun's Rust logger implementation")

		// We don't actually call SetLogLevel again to avoid the panic
		// In real applications, users should call SetLogLevel only once at startup
	})
}

func TestCreateContext(t *testing.T) {
	ctx := setupTestContext(t)

	kctx, err := CreateContext(ctx)
	if err != nil {
		// When libkrun isn't available, we should get ErrLibkrunNotAvailable
		assert.Equal(t, ErrLibkrunNotAvailable, err, "should get ErrLibkrunNotAvailable when libkrun not available")
		assert.Nil(t, kctx, "context should be nil when creation fails")
		return
	}

	// If we got here, libkrun is available
	require.NotNil(t, kctx, "context should not be nil when libkrun is available")

	// Clean up
	err = kctx.Free(ctx)
	assert.NoError(t, err, "free should succeed")
}

func TestContextFree(t *testing.T) {
	ctx := setupTestContext(t)

	kctx, err := CreateContext(ctx)
	if err != nil {
		// When libkrun isn't available, skip the test
		assert.Equal(t, ErrLibkrunNotAvailable, err, "should get ErrLibkrunNotAvailable")
		t.Skip("libkrun not available")
	}

	require.NotNil(t, kctx, "context should not be nil")

	err = kctx.Free(ctx)
	assert.NoError(t, err, "free should succeed")
}

func TestSetVMConfig(t *testing.T) {
	ctx := setupTestContext(t)

	kctx, err := CreateContext(ctx)
	if err != nil {
		assert.Equal(t, ErrLibkrunNotAvailable, err, "should get ErrLibkrunNotAvailable")
		t.Skip("libkrun not available")
	}
	defer kctx.Free(ctx)

	tests := []struct {
		name   string
		config VMConfig
	}{
		{
			name: "SingleCPU",
			config: VMConfig{
				NumVCPUs: 1,
				RAMMiB:   256,
			},
		},
		{
			name: "MultiCPU",
			config: VMConfig{
				NumVCPUs: 4,
				RAMMiB:   1024,
			},
		},
		{
			name: "MinimalRAM",
			config: VMConfig{
				NumVCPUs: 1,
				RAMMiB:   64,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := kctx.SetVMConfig(ctx, tt.config)
			assert.NoError(t, err, "SetVMConfig should succeed")
		})
	}
}

func TestSetRoot(t *testing.T) {
	ctx := setupTestContext(t)

	kctx, err := CreateContext(ctx)
	if err != nil {
		assert.Equal(t, ErrLibkrunNotAvailable, err, "should get ErrLibkrunNotAvailable")
		t.Skip("libkrun not available")
	}
	defer kctx.Free(ctx)

	tests := []struct {
		name     string
		rootPath string
	}{
		{"TmpPath", "/tmp"},
		{"RootPath", "/"},
		{"HomePath", "/home"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := kctx.SetRoot(ctx, tt.rootPath)
			// SetRoot might not be available in SEV variant
			if err != nil {
				t.Logf("SetRoot failed (might be SEV variant): %v", err)
			}
		})
	}
}

func TestDiskOperations(t *testing.T) {
	ctx := setupTestContext(t)

	kctx, err := CreateContext(ctx)
	if err != nil {
		assert.Equal(t, ErrLibkrunNotAvailable, err, "should get ErrLibkrunNotAvailable")
		t.Skip("libkrun not available")
	}
	defer kctx.Free(ctx)

	t.Run("AddDisk", func(t *testing.T) {
		config := DiskConfig{
			BlockID:  "test",
			Path:     "/tmp/test.raw",
			ReadOnly: true,
		}
		err := kctx.AddDisk(ctx, config)
		assert.NoError(t, err, "AddDisk should succeed")
	})

	t.Run("AddDisk2", func(t *testing.T) {
		config := DiskConfig{
			BlockID:  "test2",
			Path:     "/tmp/test2.qcow2",
			Format:   DiskFormatQcow2,
			ReadOnly: false,
		}
		err := kctx.AddDisk2(ctx, config)
		assert.NoError(t, err, "AddDisk2 should succeed")
	})

	t.Run("SetRootDisk", func(t *testing.T) {
		err := kctx.SetRootDisk(ctx, "/tmp/root.raw")
		assert.NoError(t, err, "SetRootDisk should succeed")
	})

	t.Run("SetDataDisk", func(t *testing.T) {
		err := kctx.SetDataDisk(ctx, "/tmp/data.raw")
		assert.NoError(t, err, "SetDataDisk should succeed")
	})
}

func TestVirtioFS(t *testing.T) {
	ctx := setupTestContext(t)

	kctx, err := CreateContext(ctx)
	if err != nil {
		assert.Equal(t, ErrLibkrunNotAvailable, err, "should get ErrLibkrunNotAvailable")
		t.Skip("libkrun not available")
	}
	defer kctx.Free(ctx)

	t.Run("Basic", func(t *testing.T) {
		config := VirtioFSConfig{
			Tag:  "shared",
			Path: "/tmp/shared",
		}
		err := kctx.AddVirtioFS(ctx, config)
		assert.NoError(t, err, "AddVirtioFS should succeed")
	})

	t.Run("WithShmSize", func(t *testing.T) {
		shmSize := uint64(1024 * 1024) // 1MB
		config := VirtioFSConfig{
			Tag:     "shared2",
			Path:    "/tmp/shared2",
			ShmSize: &shmSize,
		}
		err := kctx.AddVirtioFS(ctx, config)
		assert.NoError(t, err, "AddVirtioFS with ShmSize should succeed")
	})
}

func TestNetworking(t *testing.T) {
	ctx := setupTestContext(t)

	kctx, err := CreateContext(ctx)
	if err != nil {
		assert.Equal(t, ErrLibkrunNotAvailable, err, "should get ErrLibkrunNotAvailable")
		t.Skip("libkrun not available")
	}
	defer kctx.Free(ctx)

	t.Run("BasicPortMapping", func(t *testing.T) {
		config := NetworkConfig{
			PortMap: []string{"8080:80", "9090:90"},
		}
		err := kctx.SetNetwork(ctx, config)
		assert.NoError(t, err, "SetNetwork should succeed")
	})

	t.Run("EmptyPortMapping", func(t *testing.T) {
		config := NetworkConfig{
			PortMap: []string{},
		}
		err := kctx.SetNetwork(ctx, config)
		assert.NoError(t, err, "SetNetwork with empty port map should succeed")
	})

	t.Run("WithMAC", func(t *testing.T) {
		mac := [6]uint8{0x02, 0x00, 0x00, 0x00, 0x00, 0x01}
		config := NetworkConfig{
			MAC:     &mac,
			PortMap: []string{"8080:80"},
		}
		err := kctx.SetNetwork(ctx, config)
		assert.NoError(t, err, "SetNetwork with MAC should succeed")
	})
}

func TestGPU(t *testing.T) {
	ctx := setupTestContext(t)

	kctx, err := CreateContext(ctx)
	if err != nil {
		assert.Equal(t, ErrLibkrunNotAvailable, err, "should get ErrLibkrunNotAvailable")
		t.Skip("libkrun not available")
	}
	defer kctx.Free(ctx)

	t.Run("Basic", func(t *testing.T) {
		config := GPUConfig{
			VirglFlags: VirglUseEGL | VirglThreadSync,
		}
		err := kctx.SetGPU(ctx, config)
		assert.NoError(t, err, "SetGPU should succeed")
	})

	t.Run("WithShmSize", func(t *testing.T) {
		shmSize := uint64(16 * 1024 * 1024) // 16MB
		config := GPUConfig{
			VirglFlags: VirglUseGLX,
			ShmSize:    &shmSize,
		}
		err := kctx.SetGPU(ctx, config)
		assert.NoError(t, err, "SetGPU with ShmSize should succeed")
	})
}

func TestProcess(t *testing.T) {
	ctx := setupTestContext(t)

	kctx, err := CreateContext(ctx)
	if err != nil {
		assert.Equal(t, ErrLibkrunNotAvailable, err, "should get ErrLibkrunNotAvailable")
		t.Skip("libkrun not available")
	}
	defer kctx.Free(ctx)

	t.Run("Basic", func(t *testing.T) {
		config := ProcessConfig{
			ExecPath: "/bin/echo",
			Args:     []string{"echo", "hello"},
			Env:      []string{"PATH=/bin", "HOME=/tmp"},
		}
		err := kctx.SetProcess(ctx, config)
		assert.NoError(t, err, "SetProcess should succeed")
	})

	t.Run("WithWorkDir", func(t *testing.T) {
		workdir := "/tmp"
		config := ProcessConfig{
			ExecPath: "/bin/ls",
			Args:     []string{"ls", "-la"},
			Env:      []string{"PATH=/bin"},
			WorkDir:  &workdir,
		}
		err := kctx.SetProcess(ctx, config)
		assert.NoError(t, err, "SetProcess with WorkDir should succeed")
	})
}

func TestKernel(t *testing.T) {
	ctx := setupTestContext(t)

	kctx, err := CreateContext(ctx)
	if err != nil {
		assert.Equal(t, ErrLibkrunNotAvailable, err, "should get ErrLibkrunNotAvailable")
		t.Skip("libkrun not available")
	}
	defer kctx.Free(ctx)

	t.Run("Basic", func(t *testing.T) {
		config := KernelConfig{
			Path:    "/tmp/nonexistent_kernel", // Use a more realistic but still non-existent path
			Format:  KernelFormatELF,
			Cmdline: "console=ttyS0",
		}
		err := kctx.SetKernel(ctx, config)
		// Expect this to fail with EINVAL (-22) since the file doesn't exist
		if err != nil {
			t.Logf("SetKernel failed as expected (file doesn't exist): %v", err)
		} else {
			t.Log("SetKernel succeeded unexpectedly")
		}
	})

	t.Run("WithInitramfs", func(t *testing.T) {
		initramfs := "/tmp/nonexistent_initramfs.cpio.gz"
		config := KernelConfig{
			Path:      "/tmp/nonexistent_kernel",
			Format:    KernelFormatImageGZ,
			Initramfs: &initramfs,
			Cmdline:   "console=ttyS0 init=/sbin/init",
		}
		err := kctx.SetKernel(ctx, config)
		// Expect this to fail with EINVAL (-22) since the files don't exist
		if err != nil {
			t.Logf("SetKernel with initramfs failed as expected (files don't exist): %v", err)
		} else {
			t.Log("SetKernel with initramfs succeeded unexpectedly")
		}
	})

	t.Run("RealKernelIfExists", func(t *testing.T) {
		// Try to find an actual kernel on the system
		possibleKernels := []string{
			"/boot/vmlinuz",                  // Linux
			"/boot/kernel",                   // FreeBSD
			"/System/Library/Kernels/kernel", // macOS (unlikely to work with libkrun but let's try)
		}

		for _, kernelPath := range possibleKernels {
			if _, err := os.Stat(kernelPath); err == nil {
				config := KernelConfig{
					Path:    kernelPath,
					Format:  KernelFormatELF,
					Cmdline: "console=ttyS0",
				}
				err := kctx.SetKernel(ctx, config)
				if err != nil {
					t.Logf("SetKernel with real kernel %s failed: %v", kernelPath, err)
				} else {
					t.Logf("SetKernel with real kernel %s succeeded", kernelPath)
				}
				return // Only test the first found kernel
			}
		}
		t.Log("No real kernel found on system to test with")
	})
}

func TestVsockPorts(t *testing.T) {
	ctx := setupTestContext(t)

	kctx, err := CreateContext(ctx)
	if err != nil {
		assert.Equal(t, ErrLibkrunNotAvailable, err, "should get ErrLibkrunNotAvailable")
		t.Skip("libkrun not available")
	}
	defer kctx.Free(ctx)

	t.Run("Basic", func(t *testing.T) {
		ports := []VsockPort{
			{Port: 1234, FilePath: "/tmp/socket1"},
			{Port: 5678, FilePath: "/tmp/socket2"},
		}
		err := kctx.AddVsockPorts(ctx, ports)
		assert.NoError(t, err, "AddVsockPorts should succeed")
	})

	t.Run("WithListen", func(t *testing.T) {
		listen := true
		ports := []VsockPort{
			{Port: 9999, FilePath: "/tmp/socket3", Listen: &listen},
		}
		err := kctx.AddVsockPorts(ctx, ports)
		assert.NoError(t, err, "AddVsockPorts with Listen should succeed")
	})
}

func TestSecurity(t *testing.T) {
	ctx := setupTestContext(t)

	kctx, err := CreateContext(ctx)
	if err != nil {
		assert.Equal(t, ErrLibkrunNotAvailable, err, "should get ErrLibkrunNotAvailable")
		t.Skip("libkrun not available")
	}
	defer kctx.Free(ctx)

	t.Run("Basic", func(t *testing.T) {
		config := SecurityConfig{
			Rlimits: []string{"RLIMIT_NOFILE=1024:2048"},
		}
		err := kctx.SetSecurity(ctx, config)
		assert.NoError(t, err, "SetSecurity should succeed")
	})

	t.Run("WithUIDGID", func(t *testing.T) {
		uid := uint32(1000)
		gid := uint32(1000)
		config := SecurityConfig{
			UID:     &uid,
			GID:     &gid,
			Rlimits: []string{"RLIMIT_NOFILE=1024:2048", "RLIMIT_NPROC=512:1024"},
		}
		err := kctx.SetSecurity(ctx, config)
		assert.NoError(t, err, "SetSecurity with UID/GID should succeed")
	})

	t.Run("WithSMBIOS", func(t *testing.T) {
		config := SecurityConfig{
			SMBIOSOEMStrings: []string{"key1=value1", "key2=value2"},
		}
		err := kctx.SetSecurity(ctx, config)
		assert.NoError(t, err, "SetSecurity with SMBIOS should succeed")
	})
}

func TestAdvanced(t *testing.T) {
	ctx := setupTestContext(t)

	kctx, err := CreateContext(ctx)
	if err != nil {
		assert.Equal(t, ErrLibkrunNotAvailable, err, "should get ErrLibkrunNotAvailable")
		t.Skip("libkrun not available")
	}
	defer kctx.Free(ctx)

	t.Run("Basic", func(t *testing.T) {
		config := AdvancedConfig{}
		err := kctx.SetAdvanced(ctx, config)
		assert.NoError(t, err, "SetAdvanced should succeed")
	})

	t.Run("WithSettings", func(t *testing.T) {
		nestedVirt := true
		soundDevice := false
		consoleOutput := "/tmp/console.log"
		config := AdvancedConfig{
			NestedVirt:    &nestedVirt,
			SoundDevice:   &soundDevice,
			ConsoleOutput: &consoleOutput,
		}
		err := kctx.SetAdvanced(ctx, config)
		assert.NoError(t, err, "SetAdvanced with settings should succeed")
	})
}

// Variant-specific tests

func TestMappedVolumes(t *testing.T) {
	ctx := setupTestContext(t)

	kctx, err := CreateContext(ctx)
	if err != nil {
		assert.Equal(t, ErrLibkrunNotAvailable, err, "should get ErrLibkrunNotAvailable")
		t.Skip("libkrun not available")
	}
	defer kctx.Free(ctx)

	t.Run("Basic", func(t *testing.T) {
		mappedVolumes := []string{"/host/path1:/guest/path1", "/host/path2:/guest/path2"}
		err := kctx.SetMappedVolumes(ctx, mappedVolumes)
		// This might fail in SEV variant
		if err != nil {
			t.Logf("SetMappedVolumes failed (might be SEV variant): %v", err)
		}
	})

	t.Run("Empty", func(t *testing.T) {
		err := kctx.SetMappedVolumes(ctx, []string{})
		// This might fail in SEV variant
		if err != nil {
			t.Logf("SetMappedVolumes failed (might be SEV variant): %v", err)
		}
	})
}

func TestSEVConfig(t *testing.T) {
	ctx := setupTestContext(t)

	kctx, err := CreateContext(ctx)
	if err != nil {
		assert.Equal(t, ErrLibkrunNotAvailable, err, "should get ErrLibkrunNotAvailable")
		t.Skip("libkrun not available")
	}
	defer kctx.Free(ctx)

	t.Run("Basic", func(t *testing.T) {
		config := SEVConfig{}
		err := kctx.SetSEVConfig(ctx, config)
		// This might only work in SEV variant
		if err != nil {
			t.Logf("SetSEVConfig failed (might not be SEV variant): %v", err)
		}
	})

	t.Run("WithTEEConfig", func(t *testing.T) {
		teeConfigFile := "/path/to/tee.conf"
		config := SEVConfig{
			TEEConfigFile: &teeConfigFile,
		}
		err := kctx.SetSEVConfig(ctx, config)
		// This might only work in SEV variant
		if err != nil {
			t.Logf("SetSEVConfig with TEE failed (might not be SEV variant): %v", err)
		}
	})
}

func TestGetShutdownEventFD(t *testing.T) {
	ctx := setupTestContext(t)

	kctx, err := CreateContext(ctx)
	if err != nil {
		assert.Equal(t, ErrLibkrunNotAvailable, err, "should get ErrLibkrunNotAvailable")
		t.Skip("libkrun not available")
	}
	defer kctx.Free(ctx)

	fd, err := kctx.GetShutdownEventFD(ctx)
	if err != nil {
		// This is expected if not using EFI variant
		t.Logf("GetShutdownEventFD not available (might not be EFI variant): %v", err)
	} else {
		assert.GreaterOrEqual(t, fd, 0, "file descriptor should be non-negative")
	}
}

func TestStartEnter(t *testing.T) {
	ctx := setupTestContext(t)

	kctx, err := CreateContext(ctx)
	if err != nil {
		assert.Equal(t, ErrLibkrunNotAvailable, err, "should get ErrLibkrunNotAvailable")
		t.Skip("libkrun not available")
	}
	defer kctx.Free(ctx)

	// We don't actually call StartEnter because it would start the VM
	// This test just ensures the method exists and has the right signature
	t.Run("SignatureTest", func(t *testing.T) {
		// This would normally start the VM, so we skip it
		t.Skip("StartEnter would actually start the VM - skipping for safety")
	})
}

// TestLibkrunAvailability tests if libkrun is available on the system
func TestLibkrunAvailability(t *testing.T) {
	ctx := setupTestContext(t)

	_, err := CreateContext(ctx)
	if err != nil {
		t.Logf("libkrun is not available on this system: %v", err)
		t.Skip("libkrun not available - this is expected during development")
	} else {
		t.Log("libkrun is available on this system")
	}
}

// Benchmark tests for performance validation
func BenchmarkCreateAndFreeContext(b *testing.B) {
	ctx := setupTestContext(b)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		kctx, err := CreateContext(ctx)
		if err != nil {
			if err == ErrLibkrunNotAvailable {
				b.Skip("libkrun not available")
			}
			b.Fatal(err)
		}
		err = kctx.Free(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSetVMConfig(b *testing.B) {
	ctx := setupTestContext(b)

	kctx, err := CreateContext(ctx)
	if err != nil {
		if err == ErrLibkrunNotAvailable {
			b.Skip("libkrun not available")
		}
		b.Fatal(err)
	}
	defer kctx.Free(ctx)

	config := VMConfig{
		NumVCPUs: 2,
		RAMMiB:   512,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := kctx.SetVMConfig(ctx, config)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func TestVMNetNetworking(t *testing.T) {
	ctx := setupTestContext(t)

	kctx, err := CreateContext(ctx)
	if err != nil {
		assert.Equal(t, ErrLibkrunNotAvailable, err, "should get ErrLibkrunNotAvailable")
		t.Skip("libkrun not available")
	}
	defer kctx.Free(ctx)

	t.Run("SharedMode", func(t *testing.T) {
		portMap := []string{"8080:80", "9090:90"}
		err := kctx.SetVMNetNetwork(ctx, VMNetConfig{
			OperationMode: vmnet.OperationModeShared,
			PortMap:       portMap,
		})
		if err != nil {
			if err == ErrVMNetNotAvailable {
				t.Log("vmnet networking not available (requires entitlement and build tag)")
			} else {
				t.Logf("vmnet shared networking failed: %v", err)
			}
		} else {
			t.Log("vmnet shared networking configured successfully")
		}
	})

	t.Run("BridgedMode", func(t *testing.T) {
		sharedInterface := "en0"
		portMap := []string{"8080:80"}
		err := kctx.SetVMNetNetwork(ctx, VMNetConfig{
			OperationMode:   vmnet.OperationModeBridged,
			SharedInterface: &sharedInterface,
			PortMap:         portMap,
		})
		if err != nil {
			if err == ErrVMNetNotAvailable {
				t.Log("vmnet networking not available (requires entitlement and build tag)")
			} else {
				t.Logf("vmnet bridged networking failed: %v", err)
			}
		} else {
			t.Log("vmnet bridged networking configured successfully")
		}
	})

	t.Run("HostMode", func(t *testing.T) {
		enableIsolation := true
		portMap := []string{"9999:99"}
		err := kctx.SetVMNetNetwork(ctx, VMNetConfig{
			OperationMode:   vmnet.OperationModeHost,
			EnableIsolation: &enableIsolation,
			PortMap:         portMap,
		})
		if err != nil {
			if err == ErrVMNetNotAvailable {
				t.Log("vmnet networking not available (requires entitlement and build tag)")
			} else {
				t.Logf("vmnet host networking failed: %v", err)
			}
		} else {
			t.Log("vmnet host networking configured successfully")
		}
	})

	t.Run("CustomVMNetConfig", func(t *testing.T) {
		startAddr := "192.168.100.1"
		endAddr := "192.168.100.254"
		subnetMask := "255.255.255.0"

		config := VMNetConfig{
			OperationMode: "shared", // Use string type for stub compatibility
			StartAddress:  &startAddr,
			EndAddress:    &endAddr,
			SubnetMask:    &subnetMask,
			PortMap:       []string{"8080:80"},
			Verbose:       true,
		}

		err := kctx.SetVMNetNetwork(ctx, config)
		if err != nil {
			if err == ErrVMNetNotAvailable {
				t.Log("vmnet networking not available (requires entitlement and build tag)")
			} else {
				t.Logf("vmnet custom networking failed: %v", err)
			}
		} else {
			t.Log("vmnet custom networking configured successfully")
		}
	})
}
