package applevftest

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"github.com/walteh/ec1/pkg/hypervisors/vf/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFailedVfkitStart(t *testing.T) {
	ctx := t.Context()
	ctx = setupSlog(t, ctx)

	puipuiProvider := NewPuipuiProvider()
	slog.InfoContext(ctx, "fetching os image")

	vm := FullSetupOS(t, puipuiProvider)
	defer vm.Close(t, ctx)
	require.NotNil(t, vm)

	vm.AddSSH(t, ctx, "vsock")

	dev, err := config.NVMExpressControllerNew("/a/b")
	require.NoError(t, err)
	vm.AddDevice(t, dev)

	vm.Start(t, ctx)

	slog.InfoContext(ctx, "waiting for SSH")
	_, err = vm.retrySSHDial(t, ctx, "unix", vm.sshAddress)
	require.Error(t, err)
}

func testSSHAccess(t *testing.T, ctx context.Context, vm *testVM, network string) {

	slog.InfoContext(ctx, "testing SSH access over", "network", network)
	vm.AddSSH(t, ctx, network)
	vm.Start(t, ctx)

	slog.InfoContext(ctx, "waiting for SSH")
	vm.WaitForSSH(t, ctx)

	data, err := vm.SSHCombinedOutput(t, ctx, "whoami")
	require.NoError(t, err)
	slog.InfoContext(ctx, "executed whoami - output", "output", string(data))

	// messWithAlpine(t, ctx, vm)

	// saveCloudInitLogFilesToCacheDir(t, ctx, vm)

	slog.InfoContext(ctx, "shutting down VM")
	vm.Stop(t, ctx)
}

func saveCloudInitLogFilesToCacheDir(t *testing.T, ctx context.Context, vm *testVM) {
	out, err := vm.SSHCombinedOutput(t, ctx, "doas cat /var/log/cloud-init.log")
	require.NoError(t, err, "should check cloud-init log content: got: %s %s", err, string(out))
	// slog.InfoContext(ctx, "cloud-init log content", "output", string(out))

	outPutOut, err := vm.SSHCombinedOutput(t, ctx, "doas cat /var/log/cloud-init-output.log")
	require.NoError(t, err, "should check cloud-init log content: got: %s %s", err, string(outPutOut))
	// slog.InfoContext(ctx, "cloud-init log content", "output", string(out))

	cacheDir, err := cacheDirPrefix()
	require.NoError(t, err)

	runid := time.Now().Format("2006-01-02_15-04-05_test-output") + "-" + vm.provider.Name()

	err = os.MkdirAll(filepath.Join(cacheDir, "test-logs", runid), 0755)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(cacheDir, "test-logs", runid, "cloud-init.log"), out, 0600)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(cacheDir, "test-logs", runid, "cloud-init-output.log"), outPutOut, 0600)
	require.NoError(t, err)

	slog.InfoContext(ctx, "saved cloud-init logs to cache dir", "path", filepath.Join(cacheDir, runid))
}

func messWithAlpine(t *testing.T, ctx context.Context, vm *testVM) {
	// Check which shell you're using
	out, err := vm.SSHCombinedOutput(t, ctx, "echo $SHELL")
	require.NoError(t, err, "should get shell info")
	slog.InfoContext(ctx, "shell info", "output", string(out))

	// check if we can connect to the internet
	out, err = vm.SSHCombinedOutput(t, ctx, "ping -c 1 google.com")
	require.NoError(t, err, "should check internet connection")
	slog.InfoContext(ctx, "internet connection", "output", string(out))

	// Check if cloud-init ran at all
	out, err = vm.SSHCombinedOutput(t, ctx, "ls -la /var/log/cloud-init* 2>/dev/null || echo 'no cloud-init logs'")
	require.NoError(t, err, "should check cloud-init logs")
	slog.InfoContext(ctx, "cloud-init logs", "output", string(out))

	// Check installed packages
	out, err = vm.SSHCombinedOutput(t, ctx, "apk list --installed | grep -E 'sudo|doas' || echo 'packages not found'")
	require.NoError(t, err, "should check installed packages")
	slog.InfoContext(ctx, "sudo/doas packages", "output", string(out))

}

func TestSSHAccess(t *testing.T) {

	ctx := t.Context()
	ctx = setupSlog(t, ctx)

	providers := map[string]OsProvider{
		"fedora": NewFedoraProvider(),
		"puipui": NewPuipuiProvider(),
		// "alpine": NewAlpineProvider(),
	}

	for name, provider := range providers {
		t.Run(name, func(t *testing.T) {
			slog.InfoContext(ctx, "fetching os image")
			vm := FullSetupOS(t, provider)
			defer vm.Close(t, ctx)
			require.NotNil(t, vm)
			slog.InfoContext(ctx, "starting VM")
			testSSHAccess(t, ctx, vm, "gvproxy")
		})
	}
}

// guest listens over vsock, host connects to the guest
func TestVsockConnect(t *testing.T) {

	ctx := t.Context()
	ctx = setupSlog(t, ctx)

	puipuiProvider := NewPuipuiProvider()
	slog.InfoContext(ctx, "fetching os image")

	vm := FullSetupOS(t, puipuiProvider)
	defer vm.Close(t, ctx)
	require.NotNil(t, vm)

	vm.AddSSH(t, ctx, "tcp")

	vsockConnectPath := filepath.Join(vm.tmpDir, "vsock-connect.sock")
	dev, err := config.VirtioVsockNew(1234, vsockConnectPath, false)
	require.NoError(t, err)
	vm.AddDevice(t, dev)

	vm.Start(t, ctx)
	vm.WaitForSSH(t, ctx)

	slog.InfoContext(ctx, "path to vsock socket", "path", vsockConnectPath)
	go func() {
		for i := 0; i < 5; i++ {
			conn, err := net.DialTimeout("unix", vsockConnectPath, time.Second)
			require.NoError(t, err)
			defer conn.Close()
			data, err := io.ReadAll(conn)
			require.NoError(t, err)
			if len(data) != 0 {
				slog.InfoContext(ctx, "read data from guest", "data", string(data))
				require.Equal(t, []byte("hello host"), data)
				break
			}
		}
	}()
	slog.InfoContext(ctx, "running socat")
	vm.SSHRun(t, ctx, "echo -n 'hello host' | socat - VSOCK-LISTEN:1234")

	slog.InfoContext(ctx, "stopping VM")
	vm.Stop(t, ctx)
}

// host listens over vsock, guest connects to the host
func TestVsockListen(t *testing.T) {

	ctx := t.Context()
	ctx = setupSlog(t, ctx)

	puipuiProvider := NewPuipuiProvider()
	slog.InfoContext(ctx, "fetching os image")

	vm := FullSetupOS(t, puipuiProvider)
	defer vm.Close(t, ctx)
	require.NotNil(t, vm)

	vm.AddSSH(t, ctx, "tcp")

	tempDir := ShortTestTempDir(t)
	vsockListenPath := filepath.Join(tempDir, "vsock-listen.sock")
	ln, err := net.Listen("unix", vsockListenPath)
	require.NoError(t, err)
	go func() {
		conn, err := ln.Accept()
		// call ln.Close() after a timeout to unblock Accept() and fail the test?
		require.NoError(t, err)
		data, err := io.ReadAll(conn)
		require.NoError(t, err)
		slog.InfoContext(ctx, "read", "data", string(data))
		require.Equal(t, []byte("hello host"), data)
	}()
	slog.InfoContext(ctx, "path to vsock socket", "path", vsockListenPath)
	dev, err := config.VirtioVsockNew(1235, vsockListenPath, true)
	require.NoError(t, err)
	vm.AddDevice(t, dev)

	vm.Start(t, ctx)
	vm.WaitForSSH(t, ctx)

	vm.SSHRun(t, ctx, "echo -n 'hello host' | socat -T 2 STDIN VSOCK-CONNECT:2:1235")

	vm.Stop(t, ctx)
}

func TestFileSharing(t *testing.T) {

	ctx := t.Context()
	ctx = setupSlog(t, ctx)

	puipuiProvider := NewPuipuiProvider()
	slog.InfoContext(ctx, "fetching os image")

	vm := FullSetupOS(t, puipuiProvider)
	defer vm.Close(t, ctx)
	require.NotNil(t, vm)

	vm.AddSSH(t, ctx, "tcp")

	sharedDir := ShortTestTempDir(t)
	share, err := config.VirtioFsNew(sharedDir, "vfkit-test-share")
	require.NoError(t, err)
	vm.AddDevice(t, share)
	slog.InfoContext(ctx, "shared directory", "path", sharedDir)

	vm.Start(t, ctx)
	vm.WaitForSSH(t, ctx)

	vm.SSHRun(t, ctx, "mkdir /mnt")
	vm.SSHRun(t, ctx, "mount -t virtiofs vfkit-test-share /mnt")

	err = os.WriteFile(filepath.Join(sharedDir, "from-host.txt"), []byte("data from host"), 0600)
	require.NoError(t, err)
	data, err := vm.SSHCombinedOutput(t, ctx, "cat /mnt/from-host.txt")
	require.NoError(t, err)
	require.Equal(t, "data from host", string(data))

	vm.SSHRun(t, ctx, "echo -n 'data from guest' > /mnt/from-guest.txt")
	data, err = os.ReadFile(filepath.Join(sharedDir, "from-guest.txt"))
	require.NoError(t, err)
	require.Equal(t, "data from guest", string(data))

	vm.Stop(t, ctx)
}

type createDevFunc func(t *testing.T, vm *testVM) (config.VirtioDevice, error)
type pciidTest struct {
	vendorID  int
	deviceID  int
	createDev createDevFunc
}

var pciidTests = map[string]pciidTest{
	"virtio-net": {
		vendorID: 0x1af4, // Red Hat
		deviceID: 0x1041,
		createDev: func(t *testing.T, vm *testVM) (config.VirtioDevice, error) {
			return config.VirtioNetNew("")
		},
	},
	"virtio-serial": {
		vendorID: 0x1af4, // Red Hat
		deviceID: 0x1043,
		createDev: func(t *testing.T, vm *testVM) (config.VirtioDevice, error) {
			return config.VirtioSerialNew(filepath.Join(vm.tmpDir, "serial.log"))
		},
	},
	"virtio-rng": {
		vendorID: 0x1af4, // Red Hat
		deviceID: 0x1044,
		createDev: func(_ *testing.T, vm *testVM) (config.VirtioDevice, error) {
			return config.VirtioRngNew()
		},
	},
	"virtio-fs": {
		vendorID: 0x1af4, // Red Hat
		deviceID: 0x105a,
		createDev: func(_ *testing.T, vm *testVM) (config.VirtioDevice, error) {
			return config.VirtioFsNew(vm.tmpDir, "vfkit-share-test")
		},
	},
	"virtio-balloon": {
		vendorID: 0x1af4, // Red Hat
		deviceID: 0x1045,
		createDev: func(_ *testing.T, vm *testVM) (config.VirtioDevice, error) {
			return config.VirtioBalloonNew()
		},
	},
}

var pciidMacOS13Tests = map[string]pciidTest{
	"virtio-gpu": {
		vendorID: 0x1af4, // Red Hat
		deviceID: 0x1050,
		createDev: func(_ *testing.T, vm *testVM) (config.VirtioDevice, error) {
			return config.VirtioGPUNew()
		},
	},
	"virtio-input/trackpad": {
		vendorID: 0x106b, // Apple
		deviceID: 0x1a06,
		createDev: func(_ *testing.T, vm *testVM) (config.VirtioDevice, error) {
			return config.VirtioInputNew("pointing")
		},
	},
	"virtio-input/keyboard": {
		vendorID: 0x106b, // Apple
		deviceID: 0x1a06,
		createDev: func(_ *testing.T, vm *testVM) (config.VirtioDevice, error) {
			return config.VirtioInputNew("keyboard")
		},
	},
}

var pciidMacOS14Tests = map[string]pciidTest{
	"nvm-express": {
		vendorID: 0x106b, // Apple
		deviceID: 0x1a09,
		createDev: func(t *testing.T, vm *testVM) (config.VirtioDevice, error) {
			diskimg := filepath.Join(vm.tmpDir, "nvmexpress.img")
			f, err := os.Create(diskimg)
			require.NoError(t, err)
			require.NoError(t, f.Close())
			return config.NVMExpressControllerNew(diskimg)
		},
	},
}

var pciidVersionedTests = map[int]map[string]pciidTest{
	13: pciidMacOS13Tests,
	14: pciidMacOS14Tests,
}

func restInspect(t *testing.T, vm *testVM) *config.VirtualMachine {
	tr := &http.Transport{
		Dial: func(_, _ string) (conn net.Conn, err error) {
			return net.Dial("unix", vm.restSocketPath)
		},
	}
	client := &http.Client{Transport: tr}
	resp, err := client.Get("http://vfkit/vm/inspect")
	require.NoError(t, err)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	var unmarshalledVM config.VirtualMachine
	err = json.Unmarshal(body, &unmarshalledVM)
	require.NoError(t, err)
	return &unmarshalledVM
}

func testPCIId(t *testing.T, ctx context.Context, test pciidTest, provider OsProvider) {
	vm := FullSetupOS(t, provider)
	defer vm.Close(t, ctx)
	require.NotNil(t, vm)

	vm.AddSSH(t, ctx, "tcp")
	dev, err := test.createDev(t, vm)
	require.NoError(t, err)
	vm.AddDevice(t, dev)

	vm.Start(t, ctx)
	vm.WaitForSSH(t, ctx)
	checkPCIDevice(t, vm, test.vendorID, test.deviceID)

	unmarshalledVM := restInspect(t, vm)
	require.Equal(t, vm.config, unmarshalledVM)

	vm.Stop(t, ctx)
}

func TestPCIIds(t *testing.T) {

	ctx := t.Context()
	ctx = setupSlog(t, ctx)

	puipuiProvider := NewPuipuiProvider()
	slog.InfoContext(ctx, "fetching os image")

	for name, test := range pciidTests {
		t.Run(name, func(t *testing.T) {
			testPCIId(t, ctx, test, puipuiProvider)
		})
	}

	for macosVersion, tests := range pciidVersionedTests {
		if err := macOSAvailable(float64(macosVersion)); err == nil {
			for name, test := range tests {
				t.Run(name, func(t *testing.T) {
					testPCIId(t, ctx, test, puipuiProvider)
				})
			}
		} else {
			t.Logf("Skipping macOS %d tests", macosVersion)
		}
	}
}

func TestVirtioSerialPTY(t *testing.T) {

	ctx := t.Context()
	ctx = setupSlog(t, ctx)

	puipuiProvider := NewPuipuiProvider()
	slog.InfoContext(ctx, "fetching os image")

	vm := FullSetupOS(t, puipuiProvider)
	defer vm.Close(t, ctx)
	require.NotNil(t, vm)

	vm.AddSSH(t, ctx, "tcp")
	dev, err := config.VirtioSerialNewPty()
	require.NoError(t, err)
	vm.AddDevice(t, dev)

	vm.Start(t, ctx)
	vm.WaitForSSH(t, ctx)
	runtimeVM := restInspect(t, vm)
	var foundVirtioSerial bool
	for _, dev := range runtimeVM.Devices {
		runtimeDev, ok := dev.(*config.VirtioSerial)
		if ok {
			assert.NotEmpty(t, runtimeDev.PtyName)
			foundVirtioSerial = true
			break
		}
	}
	require.True(t, foundVirtioSerial)

	vm.Stop(t, ctx)
}

func checkPCIDevice(t *testing.T, vm *testVM, vendorID, deviceID int) {

	ctx := t.Context()
	ctx = setupSlog(t, ctx)

	re := regexp.MustCompile(fmt.Sprintf("(?m)[[:blank:]]%04x:%04x\n", vendorID, deviceID))
	lspci, err := vm.SSHCombinedOutput(t, ctx, "lspci")
	slog.InfoContext(ctx, "lspci", "output", string(lspci))
	require.NoError(t, err)
	require.Regexp(t, re, string(lspci))
}

func TestCloudInit(t *testing.T) {

	ctx := t.Context()
	ctx = setupSlog(t, ctx)

	if err := macOSAvailable(13); err != nil {
		t.Log("Skipping TestCloudInit test")
		return
	}
	fedoraProvider := NewFedoraProvider()
	slog.InfoContext(ctx, "fetching os image")

	vm := FullSetupOS(t, fedoraProvider)
	defer vm.Close(t, ctx)
	require.NotNil(t, vm)

	vm.AddSSH(t, ctx, "tcp")

	vm.Start(t, ctx)
	vm.WaitForSSH(t, ctx)

	data, err := vm.SSHCombinedOutput(t, ctx, "whoami")
	require.NoError(t, err)
	slog.InfoContext(ctx, "executed whoami - output", "output", string(data))
	require.Equal(t, "vfkituser\n", string(data))

	slog.InfoContext(ctx, "stopping vm")
	vm.Stop(t, ctx)
}
