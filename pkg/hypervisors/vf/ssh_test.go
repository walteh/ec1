package vf_test

import (
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/walteh/ec1/pkg/hypervisors"
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
