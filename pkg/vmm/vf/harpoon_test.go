package vf_test

import (
	"bufio"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/containers/common/pkg/strongunits"
	"github.com/prometheus/procfs"
	"github.com/stretchr/testify/require"

	"github.com/walteh/ec1/pkg/binembed"
	"github.com/walteh/ec1/pkg/testing/tctx"
	"github.com/walteh/ec1/pkg/testing/tlog"
	"github.com/walteh/ec1/pkg/testing/toci"
	"github.com/walteh/ec1/pkg/testing/tvmm"
	"github.com/walteh/ec1/pkg/units"
	"github.com/walteh/ec1/pkg/vmm"
	"github.com/walteh/ec1/pkg/vmm/vf"

	oci_image_cache "github.com/walteh/ec1/gen/oci-image-cache"
)

func TestHarpoon(t *testing.T) {
	tlog.SetupSlogForTest(t)

	err := binembed.PreloadAllSync()
	require.NoError(t, err)

	// Pre-load all images that will be used in the tests
	// This moves the expensive extraction work to setup time, not test execution time

	cache := toci.PreloadedImageCache(t, units.PlatformLinuxARM64, []oci_image_cache.OCICachedImage{
		oci_image_cache.OVEN_BUN_ALPINE,
		oci_image_cache.ALPINE_SOCAT_LATEST,
		oci_image_cache.ALPINE_LATEST,
		oci_image_cache.BUSYBOX_GLIBC,
	})

	t.Run("bun_version", func(t *testing.T) {
		ctx := tlog.SetupSlogForTest(t)
		ctx = tctx.WithContext(ctx, t)
		// Create a real VM for testing
		rvm := tvmm.RunHarpoonVM(t, ctx, vf.NewHypervisor(), vmm.ConatinerImageConfig{
			ImageRef: oci_image_cache.OVEN_BUN_ALPINE.String(),
			Platform: units.PlatformLinuxARM64,
			Memory:   strongunits.MiB(64).ToBytes(),
			VCPUs:    1,
		}, cache)

		var errres error
		var stdout string
		var stderr string
		var exitCode string
		var errchan = make(chan error, 1)

		go func() {
			start := time.Now()
			// Verify the OCI container filesystem is properly mounted and accessible
			// Focus on filesystem verification rather than binary execution due to library dependencies
			stdout, stderr, exitCode, errres = vmm.Exec(ctx, rvm, "/usr/local/bin/bun --version")
			if exitCode != "" {
				t.Logf("exitCode: %s", exitCode)
			}
			slog.InfoContext(ctx, "bun --version", "duration", time.Since(start))
			errchan <- errres
		}()

		select {
		case <-errchan:
		case <-time.After(3 * time.Second):
			t.Fatalf("timeout waiting for command execution")
		}

		require.NoError(t, errres, "Failed to execute commands")
		require.NotEmpty(t, stdout, "stdout should not be empty")
		require.Empty(t, stderr, "stderr should be empty")

		// make sure it is a match
		version := strings.TrimPrefix(strings.TrimSpace(stdout), "bun ")
		require.Regexp(t, `^\d+\.\d+\.\d+$`, version, "bun version should be a valid semver")
	})

	t.Run("socat_over_vsock", func(t *testing.T) {
		ctx := tlog.SetupSlogForTest(t)
		ctx = tctx.WithContext(ctx, t)

		rvm := tvmm.RunHarpoonVM(t, ctx, vf.NewHypervisor(), vmm.ConatinerImageConfig{
			ImageRef: oci_image_cache.ALPINE_SOCAT_LATEST.String(),
			Platform: units.PlatformLinuxARM64,
			Memory:   strongunits.MiB(64).ToBytes(),
			VCPUs:    1,
		}, cache)

		guestListenPort := uint32(7890) // Arbitrary vsock port for the guest to listen on

		serverCmd := fmt.Sprintf("socat VSOCK-LISTEN:%d,fork PIPE", guestListenPort)
		var errres error
		var stdout string
		var stderr string
		var exitCode string
		var errchan = make(chan error, 1)

		go func() {
			start := time.Now()
			stdout, stderr, exitCode, errres = vmm.Exec(ctx, rvm, serverCmd)
			slog.InfoContext(ctx, "socat command", "duration", time.Since(start))
			errchan <- errres
		}()

		slog.DebugContext(ctx, "Guest vsock server started, waiting for it to be ready...")

		// Give the server a moment to start up.
		// A more robust way would be to try connecting in a loop.
		select {
		case <-errchan:
			t.Logf("stdout: %s", stdout)
			t.Logf("stderr: %s", stderr)
			t.Logf("exitCode: %s", exitCode)
			t.Logf("errres: %v", errres)
			t.Fatalf("the socat command exited early, see output above")
		case <-time.After(10 * time.Millisecond):
		}

		slog.DebugContext(ctx, "Exposing vsock port", "guestPort", guestListenPort)
		// Expose the guest's vsock port. The host will connect to the guest's server.
		// conn, cleanup, err := vmm.NewUnixSocketStreamConnection(ctx, rvm.VM(), guestListenPort)
		conn, err := rvm.VM().VSockConnect(ctx, guestListenPort)
		require.NoError(t, err, "Failed to expose vsock port")
		require.NotNil(t, conn, "Host connection should not be nil")
		// t.Cleanup(cleanup)

		// Send data from host to guest via the proxied connection
		message := "hello vsock from host"
		slog.DebugContext(ctx, "Writing to host connection", "message", message)
		_, err = conn.Write([]byte(message + "\n"))
		require.NoError(t, err, "Failed to write to host connection")

		// Read the echoed data back from the guest
		buffer := bufio.NewScanner(conn)

		slog.DebugContext(ctx, "Reading from host connection")
		buffer.Scan()

		receivedMessage := buffer.Text()
		slog.DebugContext(ctx, "Received from guest", "message", receivedMessage)

		// Verify the echoed message
		require.Equal(t, message, receivedMessage, "Expected echoed message to match sent message")

		slog.InfoContext(ctx, "Vsock connectivity test successful")
	})

	t.Run("proc_mem_info", func(t *testing.T) {
		ctx := tlog.SetupSlogForTest(t)
		ctx = tctx.WithContext(ctx, t)

		// Create a real VM for testing
		rvm := tvmm.RunHarpoonVM(t, ctx, vf.NewHypervisor(), vmm.ConatinerImageConfig{
			ImageRef: oci_image_cache.BUSYBOX_GLIBC.String(),
			Platform: units.PlatformLinuxARM64,
			Memory:   strongunits.MiB(64).ToBytes(),
			VCPUs:    1,
		}, cache)

		var errres error
		var info *procfs.Meminfo
		var errchan = make(chan error, 1)

		go func() {
			info, errres = vmm.ProcMemInfo(ctx, rvm)
			errchan <- errres
		}()

		select {
		case <-errchan:
		case <-time.After(3 * time.Second):
			t.Fatalf("timeout waiting for command execution")
		}

		require.NoError(t, errres, "Failed to execute commands")
		require.NotNil(t, info, "info should not be nil")
		require.NotZero(t, info.MemTotal, "MemTotal should not be zero")
		require.NotZero(t, info.MemFree, "MemFree should not be zero")
		require.NotZero(t, info.MemAvailable, "MemAvailable should not be zero")
		require.NotZero(t, info.Buffers, "Buffers should not be zero")
		require.NotZero(t, info.Cached, "Cached should not be zero")
		require.NotZero(t, info.SwapCached, "SwapCached should not be zero")
		require.NotZero(t, info.Active, "Active should not be zero")
		require.NotZero(t, info.Inactive, "Inactive should not be zero")
	})
}
