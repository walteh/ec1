package vf_test

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/containers/common/pkg/strongunits"
	"github.com/stretchr/testify/require"
	"github.com/walteh/ec1/pkg/hypervisors"
	"github.com/walteh/ec1/pkg/hypervisors/vf"
	"github.com/walteh/ec1/pkg/machines/images/puipui"
	"github.com/walteh/ec1/pkg/testutils"
)

// MockObjcRuntime allows mocking of objc interactions

// Create a real VM for testing
func createTestVMWithSSH(t *testing.T, ctx context.Context) (*vf.VirtualMachine, hypervisors.VMIProvider) {
	hv := vf.NewHypervisor()
	pp := puipui.NewPuipuiProvider()

	problemch := make(chan error)

	go func() {
		err := hypervisors.RunVirtualMachine(ctx, hv, pp, 2, strongunits.B(1*1024*1024*1024))
		if err != nil {
			problemch <- err
			return
		}
	}()

	timeout := time.After(30 * time.Second)
	slog.DebugContext(ctx, "waiting for test VM")
	select {
	case <-timeout:
		t.Fatalf("Timed out waiting for test VM")
		return nil, nil
	case vm := <-hv.OnCreate():

		t.Cleanup(func() {
			err := vm.HardStop(ctx)
			if err != nil {
				t.Logf("problem hard stopping vm: %v", err)
			}
		})
		return vm.(*vf.VirtualMachine), pp
	case err := <-problemch:
		t.Fatalf("problem running vm: %v", err)
		return nil, nil
	}
}

// Mock bootloader for testing
type mockBootloader struct{}

func (m *mockBootloader) GetKernel() ([]byte, error) {
	return []byte{}, nil
}

func (m *mockBootloader) GetInitRD() ([]byte, error) {
	return []byte{}, nil
}

func (m *mockBootloader) GetCmdLine() (string, error) {
	return "console=hvc0", nil
}

func TestSSH(t *testing.T) {
	ctx := t.Context()
	ctx = testutils.SetupSlog(t, ctx)

	// // Skip on non-macOS platforms
	// if virtualizationFramework == 0 {
	// 	t.Skip("Skipping test as Virtualization framework is not available")
	// }

	// Create a real VM for testing
	vm, vmi := createTestVMWithSSH(t, ctx)
	if vm == nil {
		t.Skip("Could not create test VM")
		return
	}

	slog.DebugContext(ctx, "waiting for test VM to be running")

	if err := hypervisors.WaitForVMState(ctx, vm, hypervisors.VirtualMachineStateTypeRunning, time.After(30*time.Second)); err != nil {
		t.Fatalf("timeout waiting for vm to be running: %v", err)
	}

	opts := vm.Opts()
	gvnetProvisioner, ok := hypervisors.ProvisionerForType[*hypervisors.GvproxyProvisioner](opts)
	if !ok {
		t.Fatalf("gvnet provisioner not found")
	}

	port, err := gvnetProvisioner.SSHURL(ctx)
	if err != nil {
		t.Fatalf("error getting port: %v", err)
	}

	sshClient, err := hypervisors.ObtainSSHConnectionWithGuest(ctx, port, vmi.SSHConfig(), time.After(10*time.Second))
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
