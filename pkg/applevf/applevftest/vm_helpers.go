package applevftest

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Code-Hex/vz/v3"
	"github.com/crc-org/vfkit/pkg/cmdline"
	"github.com/crc-org/vfkit/pkg/config"
	"github.com/lmittmann/tint"
	slogctx "github.com/veqryn/slog-context"
	"github.com/walteh/ec1/pkg/applevf"
	"gitlab.com/tozd/go/errors"

	vfkithelpers "github.com/crc-org/crc/v2/pkg/drivers/vfkit"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
)

func setupSlog(t *testing.T, ctx context.Context) context.Context {

	cached, err := cacheDirPrefix()
	require.NoError(t, err)

	tmpdir := filepath.Dir(t.TempDir())

	slog.DebugContext(ctx, "redacting as [test-tmp-dir]", "string", strings.TrimPrefix(tmpdir, "/"))
	slog.DebugContext(ctx, "redacting as [cache-dir]", "string", strings.TrimPrefix(cached, "/"))

	tintHandler := tint.NewHandler(os.Stdout, &tint.Options{
		Level:      slog.LevelDebug,
		TimeFormat: "2006-01-02 15:04 05.0000",
		AddSource:  true,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// if the value has the name of the test tmp dir, replace it with [tmpdir]
			if strings.Contains(a.Value.String(), tmpdir) {
				a = slog.Attr{Key: a.Key, Value: slog.StringValue(strings.Replace(a.Value.String(), tmpdir, "[test-tmp-dir]", 1))}
			}
			if strings.Contains(a.Value.String(), cached) {
				a = slog.Attr{Key: a.Key, Value: slog.StringValue(strings.Replace(a.Value.String(), cached, "[cache-dir]", 1))}
			}
			return a
		},
	})

	ctxHandler := slogctx.NewHandler(tintHandler, nil)

	mylogger := slog.New(ctxHandler)
	slog.SetDefault(mylogger)

	return slogctx.NewCtx(ctx, mylogger)
}

func retryIPFromMAC(t *testing.T, ctx context.Context, errCh chan error, macAddress string) (string, error) {
	t.Helper()
	var (
		err error
		ip  string
	)

	tenSeconds := time.After(10 * time.Second)

	for {
		select {
		case err := <-errCh:
			// slog.ErrorContext(ctx, "retryIPFromMAC returned", "error", err)
			return "", errors.Errorf("error observed before finding IP address: %w", err)
		case <-time.After(1 * time.Second):
			slog.DebugContext(ctx, "getting IP address by MAC address", "mac", macAddress)
			ip, err = vfkithelpers.GetIPAddressByMACAddress(macAddress)
			if err != nil {
				if strings.HasPrefix(err.Error(), "could not find an IP address for ") {
					continue
				}
				return "", errors.Errorf("getting IP address by MAC address: %w", err)
			}
			slog.InfoContext(ctx, "found IP address", "ip", ip, "mac", macAddress)
			return ip, nil
		case <-tenSeconds:
			return "", errors.New("timeout getting IP from MAC")
		}
	}
}

func retrySSHDial(t *testing.T, ctx context.Context, errCh chan error, scheme string, address string, sshConfig *ssh.ClientConfig) (*ssh.Client, error) {
	t.Helper()
	var (
		sshClient *ssh.Client
		err       error
	)

	tenSeconds := time.After(30 * time.Second)
	for {
		select {
		case err := <-errCh:
			return nil, errors.Errorf("error observed before establishing SSH connection: %w", err)
		case <-time.After(1 * time.Second):
			slog.DebugContext(ctx, "trying ssh dial", "address", address, "scheme", scheme)
			sshClient, err = ssh.Dial(scheme, address, sshConfig)
			if err == nil {
				slog.InfoContext(ctx, "established SSH connection", "address", address, "scheme", scheme)
				return sshClient, nil
			}
			slog.DebugContext(ctx, "ssh failed", "error", err)
		case <-tenSeconds:
			return nil, errors.New("timeout waiting for SSH")
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
		err := applevf.RunVFKit(ctx, vm, opts)
		if err != nil {
			err = errors.Errorf("running vfkit: %w", err)
		}
		slog.Log(ctx, slog.LevelDebug, "attempting to close error channel")
		errCh <- err
		slog.Log(ctx, slog.LevelDebug, "applevf.RunVFKit closed errCh")
		close(errCh)
	}()

	return &vfkitRunner{
		errCh,
		false,
		restSocketPath,
	}
}

func (cmd *vfkitRunner) Wait(t *testing.T, ctx context.Context) {
	slog.InfoContext(ctx, "waiting for vfkit to return")
	err := <-cmd.errCh
	slog.InfoContext(ctx, "vfkit returned", "error", err)
	require.NoError(t, err)
	cmd.gracefullyShutdown = true
}

func (cmd *vfkitRunner) Close(t *testing.T, ctx context.Context) {
	if cmd != nil && !cmd.gracefullyShutdown {
		slog.InfoContext(ctx, "killing left-over vfkit process")
		// err := cmd.Cmd.Process.Kill()
		// if err != nil {
		// 	slog.WarnContext(ctx, "failed to kill vfkit process", "error", err)
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
	ctx := t.Context()
	ctx = setupSlog(t, ctx)

	vm := &testVM{
		provider: provider,
	}
	vmConfig, err := provider.ToVirtualMachine(ctx)
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

func (vm *testVM) AddSSH(t *testing.T, ctx context.Context, network string) {
	t.Helper()
	vmMacAddress, err := vz.NewRandomLocallyAdministeredMACAddress()
	require.NoError(t, err)
	var (
		dev config.VirtioDevice
	)
	method := vm.findSSHAccessMethod(t, network)
	switch network {
	case "tcp":
		slog.InfoContext(ctx, "adding virtio-net device", "MAC", vmMacAddress)
		vm.sshNetwork = "tcp"
		vm.macAddress = vmMacAddress.String()
		vm.port = method.port
		dev, err = config.VirtioNetNew(vm.macAddress)
		require.NoError(t, err)
	case "vsock":
		slog.InfoContext(ctx, "adding virtio-vsock device", "port", method.port)
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

func (vm *testVM) Start(t *testing.T, ctx context.Context) {
	t.Helper()
	vm.vfkitCmd = startVfkit(t, ctx, vm.config)
	vm.restSocketPath = vm.vfkitCmd.restSocketPath
}

func (vm *testVM) Stop(t *testing.T, ctx context.Context) {
	t.Helper()
	go vm.SSHRun(t, ctx, vm.provider.ShutdownCommand())
	vm.vfkitCmd.Wait(t, ctx)
}

func (vm *testVM) Close(t *testing.T, ctx context.Context) {
	if vm.sshClient != nil {
		vm.sshClient.Close()
	}
	vm.vfkitCmd.Close(t, ctx)
}

func (vm *testVM) WaitForSSH(t *testing.T, ctx context.Context) {
	t.Helper()
	var (
		sshClient *ssh.Client
		err       error
	)
	switch vm.sshNetwork {
	case "tcp":
		ip, err := retryIPFromMAC(t, ctx, vm.vfkitCmd.errCh, vm.macAddress)
		require.NoError(t, err)
		sshClient, err = retrySSHDial(t, ctx, vm.vfkitCmd.errCh, "tcp", net.JoinHostPort(ip, strconv.FormatUint(uint64(vm.port), 10)), vm.provider.SSHConfig())
		require.NoError(t, err)
	case "vsock":
		sshClient, err = retrySSHDial(t, ctx, vm.vfkitCmd.errCh, "unix", vm.vsockPath, vm.provider.SSHConfig())
		require.NoError(t, err)
	default:
		t.FailNow()
	}

	vm.sshClient = sshClient
}

func (vm *testVM) SSHRun(t *testing.T, ctx context.Context, command string) {
	t.Helper()
	slog.InfoContext(ctx, "running command", "command", command)
	sshSession, err := vm.sshClient.NewSession()
	require.NoError(t, err)
	defer sshSession.Close()
	err = sshSession.Run(command)
	slog.InfoContext(ctx, "command returned", "error", err)
}

func (vm *testVM) SSHSignal(t *testing.T, ctx context.Context, signal ssh.Signal) {
	t.Helper()
	sshSession, err := vm.sshClient.NewSession()
	require.NoError(t, err)
	defer sshSession.Close()
	sshSession.Signal(signal)
}

func (vm *testVM) SSHCombinedOutput(t *testing.T, ctx context.Context, command string) ([]byte, error) {
	t.Helper()
	sshSession, err := vm.sshClient.NewSession()
	require.NoError(t, err)
	defer sshSession.Close()
	return sshSession.CombinedOutput(command)
}

func DebugVsockConnection(ctx context.Context, sshClient *ssh.Client) error {
	// List of diagnostic commands to run
	commands := []struct {
		name    string
		command string
	}{
		{"Check vsock modules", "lsmod | grep vsock || echo 'No vsock modules found'"},
		{"Check vsock device", "ls -la /dev/vsock 2>/dev/null || echo 'No vsock device found'"},
		{"SSH service status", "sudo systemctl status sshd || echo 'SSH service not running'"},
		{"SSH listening status", "sudo ss -tlnp | grep sshd || echo 'SSH not listening'"},
		{"SSH config check", "sudo grep -r Listen /etc/ssh/ || echo 'No Listen directives found'"},
		{"SELinux status", "getenforce || echo 'SELinux not available'"},
		{"Check SSH config", "sudo cat /etc/ssh/sshd_config || echo 'Cannot read SSH config'"},
	}

	// Run each command and log the results
	for _, cmd := range commands {
		slog.InfoContext(ctx, "Running diagnostic command", "command", cmd.name)
		session, err := sshClient.NewSession()
		if err != nil {
			return errors.Errorf("creating SSH session for %s: %w", cmd.name, err)
		}

		output, err := session.CombinedOutput(cmd.command)
		session.Close()

		if err != nil {
			slog.WarnContext(ctx, "Command failed", "command", cmd.name, "error", err)
		} else {
			slog.InfoContext(ctx, "Command result", "command", cmd.name, "output", string(output))
		}
	}

	// Try to configure vsock support
	slog.InfoContext(ctx, "Attempting to enable vsock in SSH")
	configureSession, err := sshClient.NewSession()
	if err != nil {
		return errors.Errorf("creating SSH session for configuration: %w", err)
	}
	defer configureSession.Close()

	// Create vsock SSH config and restart service
	configCmd := `
sudo bash -c "cat > /etc/ssh/sshd_config.d/vsock.conf << EOF
ListenVsock Yes
Port 2222
EOF"
sudo systemctl restart sshd
sudo systemctl status sshd
`
	output, err := configureSession.CombinedOutput(configCmd)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to configure SSH for vsock", "error", err, "output", string(output))
		return errors.Errorf("configuring SSH for vsock: %w", err)
	}

	slog.InfoContext(ctx, "SSH configured for vsock", "output", string(output))
	return nil
}
