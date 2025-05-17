package applevftest

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/Code-Hex/vz/v3"
	"github.com/crc-org/vfkit/pkg/cmdline"
	"github.com/crc-org/vfkit/pkg/config"
	"github.com/stretchr/testify/require"
	"gitlab.com/tozd/go/errors"

	vfkithelpers "github.com/crc-org/crc/v2/pkg/drivers/vfkit"

	"github.com/walteh/ec1/pkg/networks/gvnet"
	"github.com/walteh/ec1/pkg/port"
	"github.com/walteh/ec1/sandbox/pkg/cloud/hypervisor/applevf"
)

func init() {
	// pp.
}

func (vm *testVM) retryIPFromMAC(t *testing.T, ctx context.Context, macAddress string) (string, error) {
	t.Helper()
	var (
		err error
		ip  string
	)

	tenSeconds := time.After(10 * time.Second)

	for {
		select {
		case err := <-vm.vfkitCmd.errCh:
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

func (vm *testVM) retrySSHDial(t *testing.T, ctx context.Context, scheme string, address string) (*ssh.Client, error) {
	t.Helper()
	var (
		sshClient *ssh.Client
		err       error
	)

	tenSeconds := time.After(30 * time.Second)
	for {
		select {
		case err := <-vm.vfkitCmd.errCh:
			return nil, errors.Errorf("error observed before establishing SSH connection: %w", err)
		case <-time.After(1 * time.Second):
			slog.DebugContext(ctx, "trying ssh dial", "address", address, "scheme", scheme)
			sshClient, err = ssh.Dial(scheme, address, vm.provider.SSHConfig())
			if err == nil {
				slog.InfoContext(ctx, "established SSH connection", "address", address, "scheme", scheme)
				return sshClient, nil
			}
			slog.DebugContext(ctx, "ssh failed", "error", err)
			go vm.vfkitCmd.TryInspect(t, ctx)
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

	opts.LogLevel = "debug"

	restSocketPath := filepath.Join(t.TempDir(), "rest.sock")
	opts.RestfulURI = fmt.Sprintf("unix://%s", restSocketPath)
	// opts.RestfulURI = "none://"
	opts.Bootloader.Append("efi")

	errCh := make(chan error)
	go func() {
		err := applevf.RunVFKit(ctx, vm, opts)
		if err != nil {
			err = errors.Errorf("running vfkit: %w", err)
		}
		slog.Log(ctx, slog.LevelDebug, "attempting to close error channel", "error", err)
		errCh <- err
		slog.Log(ctx, slog.LevelDebug, "applevf.RunVFKit closed errCh", "error", err)
		close(errCh)
	}()

	return &vfkitRunner{
		errCh,
		false,
		restSocketPath,
	}
}

func (cmd *vfkitRunner) TryInspect(t *testing.T, ctx context.Context) {
	t.Helper()

	// Create a custom transport for Unix socket
	transport := &http.Transport{
		DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
			return net.Dial("unix", cmd.restSocketPath)
		},
	}

	// Create a client with the Unix socket transport
	_ = &http.Client{Transport: transport}

	_ = func(t *testing.T, ctx context.Context, client *http.Client, path string) {

		// Use a standard HTTP URL for the request path
		// Note: The host part will be ignored due to custom transport
		req, err := http.NewRequest("GET", "http://unix"+path, nil)
		require.NoError(t, err)

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		req.Header.Set("User-Agent", "vfkit/test")
		req.Header.Set("X-VFKit-Version", "test")
		req.Header.Set("X-VFKit-Test", "true")

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		slog.DebugContext(ctx, "hitting vm at", "path", path, "body", SlogRawJSONValue{rawJson: body})
	}

	// doRequest(t, ctx, client, "/vm/inspect")

	// doRequest(t, ctx, client, "/vm/state")
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
	sshAddress     string // mac for tcp, vsock for vsock, localhost:port for unixgram
	gvnetPort      uint
	sshClient      *ssh.Client
	restSocketPath string

	tmpDir   string
	vfkitCmd *vfkitRunner

	// fatalSSHErrorChan chan error

	sshRetry func(t *testing.T, ctx context.Context) (*ssh.Client, error)
}

func NewTestVM(t *testing.T, provider OsProvider, tmpDir string) *testVM { //nolint:revive
	ctx := t.Context()
	ctx = setupSlog(t, ctx)

	vm := &testVM{
		provider: provider,
	}
	vmConfig, err := provider.ToVirtualMachine(ctx)
	require.NoError(t, err)
	vm.config = vmConfig

	vm.tmpDir = tmpDir

	// pp.Println(vm.config)

	// reserve port

	myport, err := port.ReservePort(t.Context())
	require.NoError(t, err)
	vm.gvnetPort = uint(myport)

	return vm
}

// func (vm *testVM) findSSHAccessMethod(t *testing.T, network string) *SSHAccessMethod {
// 	t.Helper()
// 	switch network {
// 	case "any":
// 		accessMethods := vm.provider.SSHAccessMethods()
// 		require.NotZero(t, len(accessMethods))
// 		return &accessMethods[0]
// 	default:
// 		for _, accessMethod := range vm.provider.SSHAccessMethods() {
// 			if accessMethod.network == network {
// 				return &accessMethod
// 			}
// 		}
// 	}

// 	t.Fatalf("unknown SSH network: %s", network)
// 	return nil
// }

const (
	GUEST_VSOCK_PORT = 2222
	GUEST_TCP_PORT   = 22
)

func (vm *testVM) AddSSH(t *testing.T, ctx context.Context, network string) {
	t.Helper()

	var (
		dev config.VirtioDevice
		err error
	)

	// method := vm.findSSHAccessMethod(t, network)
	switch network {
	case "tcp":
		vmMacAddress, err := vz.NewRandomLocallyAdministeredMACAddress()
		require.NoError(t, err)
		slog.InfoContext(ctx, "adding virtio-net device", "MAC", vmMacAddress)
		vm.sshNetwork = "tcp"
		vm.sshAddress = vmMacAddress.String()
		dev, err = config.VirtioNetNew(vm.sshAddress)
		require.NoError(t, err)
	case "vsock":
		slog.InfoContext(ctx, "adding virtio-vsock device", "port", GUEST_VSOCK_PORT)
		vm.sshNetwork = "vsock"
		vm.sshAddress = filepath.Join(vm.tmpDir, fmt.Sprintf("vsock-%d.sock", GUEST_VSOCK_PORT))
		dev, err = config.VirtioVsockNew(uint(GUEST_VSOCK_PORT), vm.sshAddress, false)
		require.NoError(t, err)
	case "gvnet":
		vm.sshAddress = fmt.Sprintf("127.0.0.1:%d", vm.gvnetPort)
		vm.sshNetwork = "gvnet"

		// socketPath := "unixgram://" + filepath.Join(vm.tmpDir, "vf.sock")
		// readyChan := make(chan struct{})

		cfg := &gvnet.GvproxyConfig{
			// VMSocket: tapsock.NewVFKitVMSocket(socketPath),
			// GuestSSHPort:       VSOCK_PORT,
			VMHostPort:         fmt.Sprintf("tcp://%s", vm.sshAddress),
			EnableDebug:        false,
			EnableStdioSocket:  false,
			EnableNoConnectAPI: true,
			// ReadyChan:          readyChan,
		}

		// // // create the socket
		// os.Remove(socketPath)
		// os.Create(socketPath)

		// devd, err := cfg.VirtioNetDevice(ctx)
		// require.NoError(t, err)

		// devz, err := config.VirtioNetNew(devd.MacAddress.String())

		// require.NoError(t, err)
		// devz.SetUnixSocketPath(devd.UnixSocketPath)
		// dev = devz

		// pp.Println(dev)
		// require.NoError(t, err)

		devc, waiter, err := gvnet.NewProxy(ctx, cfg)
		if err != nil {
			slog.ErrorContext(ctx, "gvnet start failed", "error", err)
		}

		devz, err := config.VirtioNetNew(devc.MacAddress.String())
		require.NoError(t, err)
		devz.Nat = devc.Nat
		devz.MacAddress = devc.MacAddress
		devz.UnixSocketPath = ""
		devz.Socket = devc.Socket
		dev = devz

		go func() {

			if err := waiter(ctx); err != nil {
				slog.ErrorContext(ctx, "gvnet failed", "error", err)
			}
		}()

		// select {
		// // case <-readyChan:
		// case <-time.After(10 * time.Second):
		// 	t.Fatalf("timeout waiting for gvnet to be ready ")
		// }

	default:
		t.Fatalf("unknown SSH network: %s", network)
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
		ip, err := vm.retryIPFromMAC(t, ctx, vm.sshAddress)
		require.NoError(t, err)
		sshClient, err = vm.retrySSHDial(t, ctx, "tcp", net.JoinHostPort(ip, strconv.FormatUint(uint64(GUEST_TCP_PORT), 10)))
		require.NoError(t, err)
	case "vsock":
		sshClient, err = vm.retrySSHDial(t, ctx, "unix", vm.sshAddress)
		require.NoError(t, err)
	case "gvnet":
		sshClient, err = vm.retrySSHDial(t, ctx, "tcp", vm.sshAddress)
		require.NoError(t, err)
	default:
		t.Fatalf("unknown SSH network: %s", vm.sshNetwork)
	}

	vm.sshClient = sshClient
}

func (vm *testVM) SSHRun(t *testing.T, ctx context.Context, command string) {
	t.Helper()
	slog.InfoContext(ctx, "running command", "command", command)
	sshSession, err := vm.sshClient.NewSession()
	require.NoError(t, err)
	defer sshSession.Close()
	output, err := sshSession.CombinedOutput(command)
	if err != nil {
		slog.ErrorContext(ctx, "command failed", "error", err, "output", string(output))

	}
}

func (vm *testVM) SSHRunFatal(t *testing.T, ctx context.Context, command string) {
	t.Helper()
	slog.InfoContext(ctx, "running command", "command", command)

	sshSession, err := vm.sshClient.NewSession()
	require.NoError(t, err)
	defer sshSession.Close()
	output, err := sshSession.CombinedOutput(command)
	if err != nil {
		slog.ErrorContext(ctx, "command failed", "error", err, "output", string(output))
		vm.vfkitCmd.errCh <- errors.Errorf("important ssh command failed: %w", err)
	}
}

func (vm *testVM) SSHSignal(t *testing.T, ctx context.Context, signal ssh.Signal) {
	t.Helper()
	sshSession, err := vm.sshClient.NewSession()
	require.NoError(t, err)
	defer sshSession.Close()
	err = sshSession.Signal(signal)
	require.NoError(t, err)
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
