package applevftest

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Code-Hex/vz/v3"
	"github.com/crc-org/vfkit/pkg/cmdline"
	"github.com/crc-org/vfkit/pkg/config"
	"github.com/walteh/ec1/pkg/applevf"

	vfkithelpers "github.com/crc-org/crc/v2/pkg/drivers/vfkit"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
)

func retryIPFromMAC(t *testing.T, errCh chan error, macAddress string) (string, error) {
	t.Helper()
	var (
		err error
		ip  string
	)

	tenSeconds := time.After(10 * time.Second)

	for {
		select {
		case err := <-errCh:
			slog.ErrorContext(t.Context(), "retryIPFromMAC returned", "error", err)
			return "", err
		case <-time.After(1 * time.Second):
			slog.DebugContext(t.Context(), "getting IP address by MAC address", "mac", macAddress)
			ip, err = vfkithelpers.GetIPAddressByMACAddress(macAddress)
			if err != nil {
				if strings.HasPrefix(err.Error(), "could not find an IP address for ") {
					continue
				}
				return "", err
			}
			slog.InfoContext(t.Context(), "found IP address", "ip", ip, "mac", macAddress)
			return ip, nil
		case <-tenSeconds:
			return "", fmt.Errorf("timeout getting IP from MAC: %w", err)
		}
	}
}

func retrySSHDial(t *testing.T, errCh chan error, scheme string, address string, sshConfig *ssh.ClientConfig) (*ssh.Client, error) {
	t.Helper()
	var (
		sshClient *ssh.Client
		err       error
	)

	tenSeconds := time.After(10 * time.Second)
	for {
		select {
		case err := <-errCh:
			slog.ErrorContext(t.Context(), "ssh dial returned", "error", err)
			return nil, err
		case <-time.After(1 * time.Second):
			slog.DebugContext(t.Context(), "trying ssh dial")
			sshClient, err = ssh.Dial(scheme, address, sshConfig)
			if err == nil {
				slog.InfoContext(t.Context(), "established SSH connection", "address", address, "scheme", scheme)
				return sshClient, nil
			}
			slog.DebugContext(t.Context(), "ssh failed", "error", err)
		case <-tenSeconds:
			return nil, fmt.Errorf("timeout waiting for SSH: %w", err)
		}
	}
}

type vfkitRunner struct {
	// *exec.Cmd
	errCh              chan error
	gracefullyShutdown bool
	restSocketPath     string
}

func startVfkit(t *testing.T, ctx context.Context, vm *config.VirtualMachine) *vfkitRunner {
	t.Helper()
	opts := &cmdline.Options{}

	restSocketPath := filepath.Join(t.TempDir(), "rest.sock")

	opts.RestfulURI = fmt.Sprintf("unix://%s", restSocketPath)
	opts.Bootloader.Append("efi")

	errCh := make(chan error)
	go func() {
		err := applevf.RunVFKit(t.Context(), vm, opts)
		if err != nil {
			slog.ErrorContext(t.Context(), "applevf.RunVFKit errored out", "error", err)
		}
		slog.Log(t.Context(), slog.LevelDebug, "attempting to close error channel")
		errCh <- err
		slog.Log(t.Context(), slog.LevelDebug, "applevf.RunVFKit closed errCh")
		close(errCh)
	}()

	return &vfkitRunner{
		errCh,
		false,
		restSocketPath,
	}
}

func (cmd *vfkitRunner) Wait(t *testing.T) {
	slog.InfoContext(t.Context(), "waiting for vfkit to return")
	err := <-cmd.errCh
	slog.InfoContext(t.Context(), "vfkit returned", "error", err)
	require.NoError(t, err)
	cmd.gracefullyShutdown = true
}

func (cmd *vfkitRunner) Close(t *testing.T) {
	if cmd != nil && !cmd.gracefullyShutdown {
		slog.InfoContext(t.Context(), "killing left-over vfkit process")
		// err := cmd.Cmd.Process.Kill()
		// if err != nil {
		// 	slog.WarnContext(t.Context(), "failed to kill vfkit process", "error", err)
		// }
	}
}

type testVM struct {
	provider OsProvider
	config   *config.VirtualMachine

	sshNetwork     string
	macAddress     string // for SSH over TCP
	port           uint
	vsockPath      string // for SSH over vsock
	sshClient      *ssh.Client
	restSocketPath string

	vfkitCmd *vfkitRunner
}

func NewTestVM(t *testing.T, provider OsProvider) *testVM { //nolint:revive
	vm := &testVM{
		provider: provider,
	}
	vmConfig, err := provider.ToVirtualMachine()
	require.NoError(t, err)
	vm.config = vmConfig

	return vm
}

func (vm *testVM) findSSHAccessMethod(t *testing.T, network string) *SSHAccessMethod {
	t.Helper()
	switch network {
	case "any":
		accessMethods := vm.provider.SSHAccessMethods()
		require.NotZero(t, len(accessMethods))
		return &accessMethods[0]
	default:
		for _, accessMethod := range vm.provider.SSHAccessMethods() {
			if accessMethod.network == network {
				return &accessMethod
			}
		}
	}

	t.FailNow()
	return nil
}

func (vm *testVM) AddSSH(t *testing.T, network string) {
	t.Helper()
	vmMacAddress, err := vz.NewRandomLocallyAdministeredMACAddress()
	require.NoError(t, err)
	var (
		dev config.VirtioDevice
	)
	method := vm.findSSHAccessMethod(t, network)
	switch network {
	case "tcp":
		slog.InfoContext(t.Context(), "adding virtio-net device", "MAC", vmMacAddress)
		vm.sshNetwork = "tcp"
		vm.macAddress = vmMacAddress.String()
		vm.port = method.port
		dev, err = config.VirtioNetNew(vm.macAddress)
		require.NoError(t, err)
	case "vsock":
		slog.InfoContext(t.Context(), "adding virtio-vsock device", "port", method.port)
		vm.sshNetwork = "vsock"
		vm.vsockPath = filepath.Join(t.TempDir(), fmt.Sprintf("vsock-%d.sock", method.port))
		dev, err = config.VirtioVsockNew(uint(method.port), vm.vsockPath, false)
		require.NoError(t, err)
	default:
		t.FailNow()
	}

	vm.AddDevice(t, dev)
}

func (vm *testVM) AddDevice(t *testing.T, dev config.VirtioDevice) {
	t.Helper()
	err := vm.config.AddDevice(dev)
	require.NoError(t, err)
}

func (vm *testVM) Start(t *testing.T) {
	t.Helper()
	vm.vfkitCmd = startVfkit(t, t.Context(), vm.config)
	vm.restSocketPath = vm.vfkitCmd.restSocketPath
}

func (vm *testVM) Stop(t *testing.T) {
	t.Helper()
	go vm.SSHRun(t, vm.provider.ShutdownCommand())
	vm.vfkitCmd.Wait(t)
}

func (vm *testVM) Close(t *testing.T) {
	if vm.sshClient != nil {
		vm.sshClient.Close()
	}
	vm.vfkitCmd.Close(t)
}

func (vm *testVM) WaitForSSH(t *testing.T) {
	t.Helper()
	var (
		sshClient *ssh.Client
		err       error
	)
	switch vm.sshNetwork {
	case "tcp":
		ip, err := retryIPFromMAC(t, vm.vfkitCmd.errCh, vm.macAddress)
		require.NoError(t, err)
		sshClient, err = retrySSHDial(t, vm.vfkitCmd.errCh, "tcp", net.JoinHostPort(ip, strconv.FormatUint(uint64(vm.port), 10)), vm.provider.SSHConfig())
		require.NoError(t, err)
	case "vsock":
		sshClient, err = retrySSHDial(t, vm.vfkitCmd.errCh, "unix", vm.vsockPath, vm.provider.SSHConfig())
		require.NoError(t, err)
	default:
		t.FailNow()
	}

	vm.sshClient = sshClient
}

func (vm *testVM) SSHRun(t *testing.T, command string) {
	t.Helper()
	slog.InfoContext(t.Context(), "running command", "command", command)
	sshSession, err := vm.sshClient.NewSession()
	require.NoError(t, err)
	defer sshSession.Close()
	err = sshSession.Run(command)
	slog.InfoContext(t.Context(), "command returned", "error", err)
}

func (vm *testVM) SSHSignal(t *testing.T, signal ssh.Signal) {
	t.Helper()
	sshSession, err := vm.sshClient.NewSession()
	require.NoError(t, err)
	defer sshSession.Close()
	sshSession.Signal(signal)
}

func (vm *testVM) SSHCombinedOutput(t *testing.T, command string) ([]byte, error) {
	t.Helper()
	sshSession, err := vm.sshClient.NewSession()
	require.NoError(t, err)
	defer sshSession.Close()
	return sshSession.CombinedOutput(command)
}
