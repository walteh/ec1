package vf_test

import (
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/walteh/ec1/pkg/hypervisors"
	"github.com/walteh/ec1/pkg/machines/virtio"
	"github.com/walteh/ec1/pkg/testutils"
)

// MockObjcRuntime allows mocking of objc interactions

// Create a real VM for testing
// func createTestVMWithSSH(t *testing.T, ctx context.Context) (*vf.VirtualMachine, hypervisors.VMIProvider) {
// 	hv := vf.NewHypervisor()
// 	pp := puipui.NewPuipuiProvider()

// 	problemch := make(chan error)

// 	go func() {
// 		err := hypervisors.RunVirtualMachine(ctx, hv, pp, 2, strongunits.B(1*1024*1024*1024))
// 		if err != nil {
// 			problemch <- err
// 			return
// 		}
// 	}()

// 	timeout := time.After(30 * time.Second)
// 	slog.DebugContext(ctx, "waiting for test VM")
// 	select {
// 	case <-timeout:
// 		t.Fatalf("Timed out waiting for test VM")
// 		return nil, nil
// 	case vm := <-hv.OnCreate():

// 		t.Cleanup(func() {
// 			slog.DebugContext(ctx, "hard stopping vm")
// 			err := vm.HardStop(ctx)
// 			if err != nil {
// 				t.Logf("problem hard stopping vm: %v", err)
// 			}
// 		})
// 		return vm.(*vf.VirtualMachine), pp
// 	case err := <-problemch:
// 		t.Fatalf("problem running vm: %v", err)
// 		return nil, nil
// 	}
// }

func TestSSH(t *testing.T) {
	ctx := t.Context()
	ctx = testutils.SetupSlog(t, ctx)

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

	if err := hypervisors.WaitForVMState(ctx, rvm.VM(), hypervisors.VirtualMachineStateTypeRunning, time.After(30*time.Second)); err != nil {
		t.Fatalf("timeout waiting for vm to be running: %v", err)
	}

	// opts := rvm.VM().Opts()

	sshUrl := fmt.Sprintf("tcp://%s:%d", "127.0.0.1", rvm.PortOnHostIP())

	if err := hypervisors.WaitForVMState(ctx, rvm.VM(), hypervisors.VirtualMachineStateTypeRunning, time.After(30*time.Second)); err != nil {
		t.Fatalf("timeout waiting for vm to be running: %v", err)
	}

	sshClient, err := hypervisors.ObtainSSHConnectionWithGuest(ctx, sshUrl, pp.SSHConfig(), time.After(30*time.Second))
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

func TestVSock(t *testing.T) {
	ctx := t.Context()
	ctx = testutils.SetupSlog(t, ctx)

	// Create a real VM for testing
	rvm, pp := setupPuipuiVM(t, ctx, 1024)
	if rvm == nil {
		t.Skip("Could not create test VM")
		return
	}

	slog.DebugContext(ctx, "waiting for test VM to be running")

	err := hypervisors.WaitForVMState(ctx, rvm.VM(), hypervisors.VirtualMachineStateTypeRunning, time.After(30*time.Second))
	require.NoError(t, err, "timeout waiting for vm to be running: %v", err)

	// Setup SSH to run commands in the guest
	sshUrl := fmt.Sprintf("tcp://%s:%d", "127.0.0.1", rvm.PortOnHostIP())
	sshClient, err := hypervisors.ObtainSSHConnectionWithGuest(ctx, sshUrl, pp.SSHConfig(), time.After(30*time.Second))
	require.NoError(t, err, "error obtaining ssh connection: %v", err)
	defer sshClient.Close()

	// --- Test Vsock ---
	guestListenPort := uint32(7890) // Arbitrary vsock port for the guest to listen on

	slog.DebugContext(ctx, "Exposing vsock port", "guestPort", guestListenPort)
	// Expose the guest's vsock port. The host will connect to the guest's server.
	fd, hostConn, closer, err := hypervisors.ExposeVsock(ctx, rvm.VM(), guestListenPort, virtio.VirtioVsockDirectionGuestListensAsServer)
	require.NoError(t, err, "Failed to expose vsock port")
	require.NotNil(t, hostConn, "Host connection should not be nil")
	require.NotNil(t, closer, "Closer should not be nil")
	defer closer.Close()
	defer hostConn.Close()
	slog.DebugContext(ctx, "Vsock exposed", "hostFd", fd)

	// Start a vsock echo server in the guest via SSH
	// Using socat: listens on VSOCK port guestListenPort, forks a new process for each connection,
	// and echoes input (STDIO) back to the client.
	// Runs in the background (&) so SSH command returns.
	// We use "setsid" to ensure socat is not killed when the SSH session closes.
	serverCmd := fmt.Sprintf("setsid socat VSOCK-LISTEN:%d,fork STDOUT > /dev/null 2>&1 &", guestListenPort)
	slog.DebugContext(ctx, "Starting vsock server in guest", "command", serverCmd)

	sshSession, err := sshClient.NewSession() // Declare sshSession here
	require.NoError(t, err, "error creating ssh session for server start")

	err = sshSession.Start(serverCmd) // Use Start for background commands
	if err != nil {
		// Reading output can be helpful for debugging if Start fails immediately
		// Try to get combined output, but don't let it hang if the command is truly backgrounded
		// This is a bit tricky with Start(); if Start fails, it might be before the command detaches.
		// For a truly detached command, CombinedOutput after a failed Start might not give much.
		// output, _ := sshSession.CombinedOutput(serverCmd) // This might hang or not be useful
		slog.ErrorContext(ctx, "Failed to start guest server", "error", err /*, "output", string(output)*/)
		sshSession.Close() // Attempt to close the session even if start failed
		t.Fatalf("Failed to start guest server: %v", err)
	}
	// It's important to close the session that started the background command.
	err = sshSession.Close()
	require.NoError(t, err, "error closing ssh session for server start")

	slog.DebugContext(ctx, "Guest vsock server started, waiting for it to be ready...")
	// Give the server a moment to start up.
	// A more robust way would be to try connecting in a loop.
	time.Sleep(2 * time.Second)

	// Send data from host to guest via the proxied connection
	message := "hello vsock from host"
	slog.DebugContext(ctx, "Writing to host connection", "message", message)
	_, err = hostConn.Write([]byte(message))
	require.NoError(t, err, "Failed to write to host connection")

	// Read the echoed data back from the guest
	buffer := make([]byte, 1024)
	slog.DebugContext(ctx, "Reading from host connection")
	n, err := hostConn.Read(buffer)
	require.NoError(t, err, "Failed to read from host connection")

	receivedMessage := string(buffer[:n])
	slog.DebugContext(ctx, "Received from guest", "message", receivedMessage)

	// Verify the echoed message
	require.Equal(t, message, receivedMessage, "Expected echoed message to match sent message")

	slog.InfoContext(ctx, "Vsock connectivity test successful")

}
