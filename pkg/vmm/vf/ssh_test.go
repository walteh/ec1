package vf_test

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/stretchr/testify/require"

	"github.com/walteh/ec1/pkg/bootloader"
	"github.com/walteh/ec1/pkg/logging"
	"github.com/walteh/ec1/pkg/testing/tlog"
	"github.com/walteh/ec1/pkg/vmm"
)

func init() {
	_, _ = bootloader.UncompressInitBin(context.Background())
}

func TestSSH(t *testing.T) {
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

func TestGuestInitWrapperVSock(t *testing.T) {
	ctx := tlog.SetupSlogForTest(t)

	// Create a real VM for testing
	rvm, _ := setupPuipuiVM(t, ctx, 1024)
	if rvm == nil {
		t.Skip("Could not create test VM")
		return
	}

	slog.DebugContext(ctx, "waiting for test VM to be running")

	err := vmm.WaitForVMState(ctx, rvm.VM(), vmm.VirtualMachineStateTypeRunning, time.After(30*time.Second))
	require.NoError(t, err, "timeout waiting for vm to be running: %v", err)

	<-time.After(100 * time.Millisecond)

	mi, err := vmm.ProcMemInfo(ctx, rvm.VM())
	require.NoError(t, err, "Failed to get meminfo")

	slog.InfoContext(ctx, "meminfo", "meminfo", logging.NewSlogRawJSONValue(mi))

	require.NotNil(t, mi)

}
