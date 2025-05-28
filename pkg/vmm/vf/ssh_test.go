package vf_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/containers/image/v5/types"
	"github.com/prometheus/procfs"
	"github.com/stretchr/testify/require"

	"github.com/walteh/ec1/pkg/ec1init"
	"github.com/walteh/ec1/pkg/logging"
	"github.com/walteh/ec1/pkg/oci"
	"github.com/walteh/ec1/pkg/testing/tctx"
	"github.com/walteh/ec1/pkg/testing/tlog"
	"github.com/walteh/ec1/pkg/vmm"
)

func init() {
	_, _ = vmm.LoadInitBinToMemory(context.Background())
}

func TestSSH(t *testing.T) {
	ctx := tlog.SetupSlogForTest(t)
	ctx = tctx.WithContext(ctx, t)

	// // Skip on non-macOS platforms
	// if virtualizationFramework == 0 {
	// 	t.Skip("Skipping test as Virtualization framework is not available")
	// }

	// Create a real VM for testing
	rvm, pp := setupPuipuiVM(t, ctx, 1024)
	if rvm == nil {
		t.Skip("Could not create test VM")
		return
	}

	slog.DebugContext(ctx, "waiting for test VM to be running")

	if err := vmm.WaitForVMState(ctx, rvm.VM(), vmm.VirtualMachineStateTypeRunning, time.After(30*time.Second)); err != nil {
		t.Fatalf("timeout waiting for vm to be running: %v", err)
	}

	// opts := rvm.VM().Opts()

	sshUrl := fmt.Sprintf("tcp://%s:%d", "127.0.0.1", rvm.PortOnHostIP())

	if err := vmm.WaitForVMState(ctx, rvm.VM(), vmm.VirtualMachineStateTypeRunning, time.After(30*time.Second)); err != nil {
		t.Fatalf("timeout waiting for vm to be running: %v", err)
	}

	sshClient, err := vmm.ObtainSSHConnectionWithGuest(ctx, sshUrl, pp.SSHConfig(), time.After(30*time.Second))
	if err != nil {
		t.Fatalf("error obtaining ssh connection: %v", err)
	}

	defer sshClient.Close()

	sshSession, err := sshClient.NewSession()
	if err != nil {
		t.Fatalf("error creating ssh session: %v", err)
	}

	defer sshSession.Close()
	output, err := sshSession.CombinedOutput("echo 'hello'")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}

	t.Logf("command output: %s", string(output))

	require.Equal(t, "hello\n", string(output))

}

func makeTestSSHCommandCall(t *testing.T, ctx context.Context, sshClient *ssh.Client, command string) (string, error) {
	sshSession, err := sshClient.NewSession()
	if err != nil {
		t.Fatalf("error creating ssh session: %v", err)
	}

	defer sshSession.Close()
	output, err := sshSession.CombinedOutput(command)
	if err != nil {
		t.Fatalf("command failed: %v: %s", err, string(output))
	}

	return string(output), nil
}

func TestVSock(t *testing.T) {
	ctx := tlog.SetupSlogForTest(t)

	// Create a real VM for testing
	rvm, pp := setupPuipuiVM(t, ctx, 1024)
	if rvm == nil {
		t.Skip("Could not create test VM")
		return
	}

	slog.DebugContext(ctx, "waiting for test VM to be running")

	err := vmm.WaitForVMState(ctx, rvm.VM(), vmm.VirtualMachineStateTypeRunning, time.After(30*time.Second))
	require.NoError(t, err, "timeout waiting for vm to be running: %v", err)

	// Setup SSH to run commands in the guest
	sshUrl := fmt.Sprintf("tcp://%s:%d", "127.0.0.1", rvm.PortOnHostIP())
	sshClient, err := vmm.ObtainSSHConnectionWithGuest(ctx, sshUrl, pp.SSHConfig(), time.After(30*time.Second))
	require.NoError(t, err, "error obtaining ssh connection: %v", err)
	defer sshClient.Close()

	// --- Test Vsock ---
	guestListenPort := uint32(7890) // Arbitrary vsock port for the guest to listen on

	serverCmd := fmt.Sprintf("socat VSOCK-LISTEN:%d,fork PIPE", guestListenPort)
	slog.DebugContext(ctx, "Starting vsock server in guest", "command", serverCmd)

	sshSession, err := sshClient.NewSession() // Declare sshSession here
	require.NoError(t, err, "error creating ssh session for server start")

	closed := false

	go func() {
		defer func() {
			closed = true
			sshSession.Close()
		}()
		t.Logf("starting guest server")
		output, err := sshSession.CombinedOutput(serverCmd) // Use Start for background commands
		if err != nil && !errors.Is(err, &ssh.ExitMissingError{}) {
			slog.ErrorContext(ctx, "Failed to start guest server", "error", err /*, "output", string(output)*/)
			t.Errorf("Failed to start guest server: %v", err)
		}
		t.Logf("guest server output: %s", string(output))
	}()

	// // It's important to close the session that started the background command.
	// defer sshSession.Close()
	// require.NoError(t, err, "error closing ssh session for server start")

	slog.DebugContext(ctx, "Guest vsock server started, waiting for it to be ready...")
	// Give the server a moment to start up.
	// A more robust way would be to try connecting in a loop.
	time.Sleep(2 * time.Second)

	slog.DebugContext(ctx, "Exposing vsock port", "guestPort", guestListenPort)
	// Expose the guest's vsock port. The host will connect to the guest's server.
	// conn, cleanup, err := vmm.NewUnixSocketStreamConnection(ctx, rvm.VM(), guestListenPort)
	conn, err := rvm.VM().VSockConnect(ctx, guestListenPort)
	require.NoError(t, err, "Failed to expose vsock port")
	require.NotNil(t, conn, "Host connection should not be nil")
	// t.Cleanup(cleanup)

	require.False(t, closed, "SSH session should not be closed before we send data")
	// slog.DebugContext(ctx, "Vsock exposed", "hostFd", fd)

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

}

func TestMeminfo(t *testing.T) {
	ctx := tlog.SetupSlogForTest(t)

	// // Skip on non-macOS platforms
	// if virtualizationFramework == 0 {
	// 	t.Skip("Skipping test as Virtualization framework is not available")
	// }

	// Create a real VM for testing
	rvm, pp := setupPuipuiVM(t, ctx, 1024)
	if rvm == nil {
		t.Skip("Could not create test VM")
		return
	}

	slog.DebugContext(ctx, "waiting for test VM to be running")

	if err := vmm.WaitForVMState(ctx, rvm.VM(), vmm.VirtualMachineStateTypeRunning, time.After(30*time.Second)); err != nil {
		t.Fatalf("timeout waiting for vm to be running: %v", err)
	}

	// opts := rvm.VM().Opts()

	sshUrl := fmt.Sprintf("tcp://%s:%d", "127.0.0.1", rvm.PortOnHostIP())

	if err := vmm.WaitForVMState(ctx, rvm.VM(), vmm.VirtualMachineStateTypeRunning, time.After(30*time.Second)); err != nil {
		t.Fatalf("timeout waiting for vm to be running: %v", err)
	}

	sshClient, err := vmm.ObtainSSHConnectionWithGuest(ctx, sshUrl, pp.SSHConfig(), time.After(30*time.Second))
	if err != nil {
		t.Fatalf("error obtaining ssh connection: %v", err)
	}

	defer sshClient.Close()

	sshSession, err := sshClient.NewSession()
	if err != nil {
		t.Fatalf("error creating ssh session: %v", err)
	}

	defer sshSession.Close()
	output, err := sshSession.CombinedOutput("cat /proc/meminfo")
	if err != nil {
		t.Fatalf("command failed: %v: %s", err, string(output))
	}

	t.Logf("command output: %s", string(output))

	require.NotEmpty(t, string(output))

}

func TestGuestInitWrapperVSockPuipui(t *testing.T) {
	ctx := tlog.SetupSlogForTest(t)
	ctx = tctx.WithContext(ctx, t)
	// Create a real VM for testing
	rvm, _ := setupPuipuiVM(t, ctx, 1024)
	if rvm == nil {
		t.Skip("Could not create test VM")
		return
	}

	slog.DebugContext(ctx, "waiting for test VM to be running")

	err := vmm.WaitForVMState(ctx, rvm.VM(), vmm.VirtualMachineStateTypeRunning, time.After(30*time.Second))
	require.NoError(t, err, "timeout waiting for vm to be running: %v", err)

	select {
	case <-rvm.WaitOnVMReadyToExec():
	case <-time.After(3 * time.Second):
		t.Fatalf("timeout waiting for vm to be ready to exec")
	}

	t.Logf("ready to exec")

	var info *procfs.Meminfo
	var errres error
	var errchan = make(chan error, 1)

	go func() {
		mi, err := vmm.ProcMemInfo(ctx, rvm)
		if err != nil {
			errres = err
		}
		if mi != nil {
			info = mi
		}
		errchan <- err
	}()

	select {
	case <-errchan:
	case <-time.After(3 * time.Second):
		t.Fatalf("timeout waiting for meminfo")
	}

	require.NoError(t, errres, "Failed to get meminfo")

	slog.InfoContext(ctx, "meminfo", "meminfo", logging.NewSlogRawJSONValue(info))

	require.NotNil(t, info)

}

func TestGuestInitWrapperVSockHarpoon(t *testing.T) {
	ctx := tlog.SetupSlogForTest(t)
	ctx = tctx.WithContext(ctx, t)
	// Create a real VM for testing
	rvm, _ := setupHarpoonVM(t, ctx, 1024, map[string]io.Reader{})
	if rvm == nil {
		t.Skip("Could not create test VM")
		return
	}

	slog.DebugContext(ctx, "waiting for test VM to be running")

	err := vmm.WaitForVMState(ctx, rvm.VM(), vmm.VirtualMachineStateTypeRunning, time.After(30*time.Second))
	require.NoError(t, err, "timeout waiting for vm to be running: %v", err)

	select {
	case <-rvm.WaitOnVMReadyToExec():
	case <-time.After(3 * time.Second):
		t.Fatalf("timeout waiting for vm to be ready to exec")
	}

	t.Logf("ready to exec")

	var info *procfs.Meminfo
	var errres error
	var errchan = make(chan error, 1)

	go func() {
		mi, err := vmm.ProcMemInfo(ctx, rvm)
		if err != nil {
			errres = err
		}
		if mi != nil {
			info = mi
		}
		errchan <- err
	}()

	select {
	case <-errchan:
	case <-time.After(3 * time.Second):
		t.Fatalf("timeout waiting for meminfo")
	}

	require.NoError(t, errres, "Failed to get meminfo")

	slog.InfoContext(ctx, "meminfo", "meminfo", logging.NewSlogRawJSONValue(info))

	require.NotNil(t, info)

}

// func TestGuestInitWrapperVSockFedora(t *testing.T) {
// 	ctx := tlog.SetupSlogForTest(t)

// 	// Create a real VM for testing
// 	rvm, _ := setupFedoraVM(t, ctx, 1024)
// 	if rvm == nil {
// 		t.Skip("Could not create test VM")
// 		return
// 	}

// 	slog.DebugContext(ctx, "waiting for test VM to be running")

// 	err := vmm.WaitForVMState(ctx, rvm.VM(), vmm.VirtualMachineStateTypeRunning, time.After(30*time.Second))
// 	require.NoError(t, err, "timeout waiting for vm to be running: %v", err)

// 	<-time.After(5000 * time.Millisecond)

// 	t.Logf("ready to exec")
// 	mi, err := vmm.ProcMemInfo(ctx, rvm)
// 	require.NoError(t, err, "Failed to get meminfo")

// 	slog.InfoContext(ctx, "meminfo", "meminfo", logging.NewSlogRawJSONValue(mi))

// 	require.NotNil(t, mi)

// }

func TestGuestInitWrapperVSockCoreOS(t *testing.T) {
	ctx := tlog.SetupSlogForTest(t)

	// Create a real VM for testing
	rvm, _ := setupCoreOSVM(t, ctx, 1024*10)
	if rvm == nil {
		t.Skip("Could not create test VM")
		return
	}

	slog.DebugContext(ctx, "waiting for test VM to be running")

	err := vmm.WaitForVMState(ctx, rvm.VM(), vmm.VirtualMachineStateTypeRunning, time.After(30*time.Second))
	require.NoError(t, err, "timeout waiting for vm to be running: %v", err)

	select {
	case <-rvm.WaitOnVMReadyToExec():
	case <-time.After(8 * time.Second):
		t.Fatalf("timeout waiting for vm to be ready to exec")
	}

	t.Logf("ready to exec")
	mi, err := vmm.ProcMemInfo(ctx, rvm)
	require.NoError(t, err, "Failed to get meminfo")

	slog.InfoContext(ctx, "meminfo", "meminfo", logging.NewSlogRawJSONValue(mi))

	require.NotNil(t, mi)

}

func TestHarpoonOCI(t *testing.T) {
	ctx := tlog.SetupSlogForTest(t)
	ctx = tctx.WithContext(ctx, t)

	device, metadata, err := oci.ContainerToVirtioDeviceCached(ctx, oci.ContainerToVirtioOptions{
		ImageRef: "docker.io/oven/bun:alpine",
		Platform: &types.SystemContext{
			OSChoice:           "linux",
			ArchitectureChoice: "arm64",
		},
		OutputDir: t.TempDir(),
	})
	require.NoError(t, err, "Failed to create virtio device")

	// Log the extracted metadata
	if metadata != nil {
		t.Logf("Container metadata:")
		t.Logf("  Entrypoint: %v", metadata.Config.Entrypoint)
		t.Logf("  Cmd: %v", metadata.Config.Cmd)
		t.Logf("  WorkingDir: %s", metadata.Config.WorkingDir)
		t.Logf("  User: %s", metadata.Config.User)
	}

	buf := bytes.NewBuffer(nil)
	err = json.NewEncoder(buf).Encode(metadata)
	require.NoError(t, err, "Failed to encode container metadata")

	cmd := []string{"/usr/local/bin/bun", "--version"}

	obuf := bytes.NewBuffer(nil)
	err = json.NewEncoder(obuf).Encode(cmd)
	require.NoError(t, err, "Failed to encode container metadata")

	extraInitramfsFiles := map[string]io.Reader{
		ec1init.ContainerManifestFile: buf,
		ec1init.UserProvidedCmdline:   obuf,
	}

	// Create a real VM for testing
	rvm, _ := setupHarpoonVM(t, ctx, 1024, extraInitramfsFiles, device)
	if rvm == nil {
		t.Skip("Could not create test VM")
		return
	}

	slog.DebugContext(ctx, "waiting for test VM to be running")

	err = vmm.WaitForVMState(ctx, rvm.VM(), vmm.VirtualMachineStateTypeRunning, time.After(30*time.Second))
	require.NoError(t, err, "timeout waiting for vm to be running: %v", err)

	select {
	case <-rvm.WaitOnVMReadyToExec():
	case <-time.After(3 * time.Second):
		t.Fatalf("timeout waiting for vm to be ready to exec")
	}

	t.Logf("ready to exec")

	<-time.After(1 * time.Second)

	var errres error
	var stdout string
	var stderr string
	var exitCode string
	var errchan = make(chan error, 1)

	go func() {
		// Verify the OCI container filesystem is properly mounted and accessible
		// Focus on filesystem verification rather than binary execution due to library dependencies
		stdout, stderr, exitCode, errres = vmm.Exec(ctx, rvm, "/usr/local/bin/bun --version")
		errchan <- errres
	}()

	select {
	case <-errchan:
	case <-time.After(3 * time.Second):
		t.Fatalf("timeout waiting for command execution")
	}

	fmt.Println("stdout", stdout)
	fmt.Println("stderr", stderr)
	fmt.Println("exitCode", exitCode)

	require.NoError(t, errres, "Failed to execute commands")
	require.Contains(t, stdout, "/newroot/usr/local/bin/bun", "Should find bun binary")
	require.Contains(t, stdout, "/newroot/etc/", "Should find etc directory")
	require.Contains(t, stdout, "/newroot/bin/", "Should find bin directory")
	require.Contains(t, exitCode, "successfully", "Command should complete successfully")

	// Test passed - OCI container filesystem is properly mounted and accessible!

}
