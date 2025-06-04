package main

import (
	_ "embed"
	"syscall"

	"context"
	"io"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"runtime/debug"
	"testing"

	"github.com/containerd/containerd/v2/pkg/shim"
	"github.com/go-delve/delve/pkg/logflags"
	"github.com/go-delve/delve/pkg/proc"
	"github.com/go-delve/delve/service"
	"github.com/go-delve/delve/service/dap"
	"github.com/go-delve/delve/service/debugger"
	"github.com/sirupsen/logrus"
	"gitlab.com/tozd/go/errors"

	reexec "github.com/moby/sys/reexec"
	slogctx "github.com/veqryn/slog-context"

	"github.com/walteh/ec1/cmd/containerd-shim-harpoon-v1/containerd"
	"github.com/walteh/ec1/pkg/logging"
	"github.com/walteh/ec1/pkg/logging/logrusshim"
	"github.com/walteh/ec1/pkg/testing/tctx"
	"github.com/walteh/ec1/pkg/testing/tlog"
)

const (
	myTmpDir             = "/tmp/ec1-debugshim-test"
	logProxySockPath     = myTmpDir + "/log-proxy.sock"
	reexecShimBinaryPath = myTmpDir + "/reexec/" + shimName
	reexecSockPath       = myTmpDir + "/reexec/" + shimName + ".sock"
	debugShimBinaryPath  = myTmpDir + "/" + shimName
	dapLogPath           = myTmpDir + "/dap.log"
)

var testLogger *slog.Logger

func getTestLoggerCtx(t testing.TB) context.Context {
	ctx := slogctx.NewCtx(t.Context(), testLogger)
	ctx = tlog.SetupSlogForTestWithContext(t, ctx)
	ctx = tctx.WithContext(ctx, t)
	return ctx
}

//go:embed containerd-shim-harpoondebug-v1
var debugShimBinary []byte

func TestMain(m *testing.M) {
	// create a log proxy socket
	os.RemoveAll(myTmpDir)

	os.MkdirAll(myTmpDir, 0755)

	os.MkdirAll(filepath.Dir(debugShimBinaryPath), 0755)
	os.MkdirAll(filepath.Dir(reexecShimBinaryPath), 0755)
	os.WriteFile(debugShimBinaryPath, debugShimBinary, 0755)

	proxySock, err := net.Listen("unix", logProxySockPath)
	if err != nil {
		slog.Error("Failed to create log proxy socket", "error", err, "path", logProxySockPath)
		os.Exit(1)
	}
	defer proxySock.Close()

	// fwd logs from the proxy socket to stdout
	go func() {
		for {
			conn, err := proxySock.Accept()
			if err != nil {
				slog.Error("Failed to accept log proxy connection", "error", err)
				continue
			}
			go func() { _, _ = io.Copy(os.Stdout, conn) }()
		}
	}()

	// Set up logging for TestMain
	ctx := logging.SetupSlogSimpleToWriterWithProcessName(context.Background(), os.Stdout, true, "test")

	testLogger = slogctx.FromCtx(ctx)

	self, _ := os.Executable()

	if err := os.Symlink(self, reexecShimBinaryPath); err != nil {
		slog.Error("create shim link", "error", err)
		os.Exit(1)
	}

	oldPath := os.Getenv("PATH")
	newPath := filepath.Dir(reexecShimBinaryPath) + string(os.PathListSeparator) + oldPath
	os.Setenv("PATH", newPath)

	shimContainerdExecutablePath = reexecShimBinaryPath

	code := m.Run()

	_ = os.RemoveAll(myTmpDir)
	os.Exit(code)
}

func init() {

	reexec.Register(reexecShimBinaryPath, reexecBinaryForDebugShim)

	if reexec.Init() {
		os.Exit(0)
	}
}

func reexecBinaryForDebugShim() {

	ctx := context.Background()

	err := reexecBinaryForDebugShimE(ctx)
	if err != nil {
		slog.Error("reexecBinaryForInProcShimE failed", "error", err)
		os.Exit(1)
	}
}

func reexecBinaryForDebugShimE(ctx context.Context) error {

	// create slog writer that writes to the log proxy socket
	proxySock, err := net.Dial("unix", logProxySockPath)
	if err != nil {
		return errors.Errorf("dial %s: %w", logProxySockPath, err)
	}
	defer proxySock.Close()

	ctx = logging.SetupSlogSimpleToWriterWithProcessName(ctx, proxySock, true, "shim")

	defer func() {
		if r := recover(); r != nil {
			slog.Error("reexecBinaryForDebugShim panicked", "error", r)
			slog.Debug(string(debug.Stack()))
			os.Exit(1)
		}
		slog.Info("reexecBinaryForDebugShim exited")
	}()

	// log.L.Level = logrus.DebugLevel
	logrusshim.SetLogrusLevel(logrus.DebugLevel)
	// ctx = log.WithLogger(ctx, log.L)

	// go func() {
	// 	err := debugShimDebugBackgroundServer(ctx, proxySock, proxySock)
	// 	if err != nil {
	// 		slog.Error("debugShimDebugBackgroundServer failed", "error", err)
	// 	}
	// }()

	if syscall.Getppid() == 1 {
		// our parent died, we probably called ourselves directly
	}

	slog.Info("shim starting with args", "args", os.Args[1:], "my_pid", syscall.Getpid(), "my_parent_pid", syscall.Getppid())

	shim.Run(ctx, containerd.NewManager(testRuntime), func(c *shim.Config) {
		c.NoReaper = true
		c.NoSetupLogger = true
	})

	return nil
}

func debugShimDebugBackgroundServer(ctx context.Context, stdout, stderr io.Writer) error {
	// Determine listen address (DEBUG_SHIM_LISTEN_ADDRESS or default)
	listenAddress := os.Getenv("DAP_LISTEN_ADDRESSZ")
	if listenAddress == "" {
		listenAddress = "127.0.0.1:2345"
	}

	// Create TCP listener
	conn, err := net.Listen("tcp", listenAddress)
	if err != nil {
		return errors.Errorf("listen %s: %w", listenAddress, err)
	}
	slog.InfoContext(ctx, "shim debug server listening", "addr", listenAddress, "os.Args", os.Args)

	disconnectChan := make(chan struct{})
	cfg := &service.Config{
		Listener:       conn,
		DisconnectChan: disconnectChan,
		Debugger: debugger.Config{
			AttachPid:  os.Getpid(),
			Backend:    "native",
			Foreground: true, // server always runs without terminal client
			Packages:   []string{logging.GetCurrentCallerURI().Package},
			Stdout:     proc.OutputRedirect{Path: dapLogPath},
			Stderr:     proc.OutputRedirect{Path: dapLogPath},
			// DebugInfoDirectories: conf.DebugInfoDirectories,
			// CheckGoVersion:       checkGoVersion,
			// DisableASLR:          disableASLR,
		},
		APIVersion:  2,
		AcceptMulti: true, // allow multiple VS Code reconnects
		// CheckLocalConnUser: checkLocalConnUser,
	}

	// client := rpc2.NewClientFromConn(conn)
	// client.ToggleBreakpointByName()

	err = logflags.Setup(true, "dap", dapLogPath)
	if err != nil {
		return errors.Errorf("logflags.Setup: %w", err)
	}

	server := dap.NewServer(cfg)
	defer server.Stop()
	server.Run()

	<-disconnectChan
	return nil
}
