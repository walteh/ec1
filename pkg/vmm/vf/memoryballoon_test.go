package vf_test

import (
	"log/slog"
	"testing"

	"github.com/Code-Hex/vz/v3"
	"github.com/containers/common/pkg/strongunits"
	"github.com/dustin/go-humanize"
	"github.com/stretchr/testify/require"

	"github.com/walteh/ec1/pkg/testing/tlog"
	"github.com/walteh/ec1/pkg/testing/toci"
	"github.com/walteh/ec1/pkg/testing/tvmm"
	"github.com/walteh/ec1/pkg/units"
	"github.com/walteh/ec1/pkg/vmm"
	"github.com/walteh/ec1/pkg/vmm/vf"
)

func TestMemoryBalloonDevices(t *testing.T) {
	ctx := tlog.SetupSlogForTest(t)

	cache := toci.TestSimpleCache(t)

	// // Skip on non-macOS platforms
	// if virtualizationFramework == 0 {
	// 	t.Skip("Skipping test as Virtualization framework is not available")
	// }

	// Create a real VM for testing
	rvm := tvmm.RunHarpoonVM(t, ctx, vf.NewHypervisor(), vmm.ConatinerImageConfig{
		ImageRef: "docker.io/library/alpine:latest",
		Platform: units.PlatformLinuxARM64,
		Memory:   strongunits.MiB(1024).ToBytes(),
		VCPUs:    1,
	}, cache)
	require.NotNil(t, rvm)

	// Now we can call the actual method
	devices := rvm.VM().VZ().MemoryBalloonDevices()

	// Just check that the call completes - results will depend on the actual environment
	if len(devices) == 0 {
		t.Fatalf("No memory balloon devices found")
	} else {
		t.Logf("Found %d memory balloon devices", len(devices))
	}
}

func TestSetTargetVirtualMachineMemorySize(t *testing.T) {
	ctx := tlog.SetupSlogForTest(t)

	// Skip on non-macOS platforms
	// if virtualizationFramework == 0 {
	// 	t.Skip("Skipping test as Virtualization framework is not available")
	// }

	cache := toci.TestSimpleCache(t)

	startingMemory := strongunits.MiB(512)
	targetMemory := strongunits.MiB(300)

	rvm := tvmm.RunHarpoonVM(t, ctx, vf.NewHypervisor(), vmm.ConatinerImageConfig{
		ImageRef: "docker.io/library/alpine:latest",
		Platform: units.PlatformLinuxARM64,
		Memory:   startingMemory.ToBytes(),
		VCPUs:    1,
	}, cache)
	require.NotNil(t, rvm)

	// Get devices
	devices := rvm.VM().VZ().MemoryBalloonDevices()

	require.NotNil(t, devices)
	require.Equal(t, len(devices), 1)

	// Try to set memory size on the first device
	device := devices[0]

	trad, ok := device.(*vz.VirtioTraditionalMemoryBalloonDevice)
	require.True(t, ok)

	// Get the current target memory size
	sizeBefore := trad.GetTargetVirtualMachineMemorySize()
	slog.DebugContext(ctx, "sizeBefore", "sizeBefore", humanize.Bytes(sizeBefore), "startingMemory", humanize.Bytes(uint64(startingMemory.ToBytes())))
	require.Equal(t, sizeBefore, uint64(startingMemory.ToBytes()))

	trad.SetTargetVirtualMachineMemorySize(uint64(targetMemory.ToBytes())) // 100 MB

	sizeAfter := trad.GetTargetVirtualMachineMemorySize()

	require.Equal(t, sizeAfter, uint64(targetMemory.ToBytes()))

}
