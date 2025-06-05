package tcontainerd

import (
	"context"
	"io"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"runtime/debug"
	"syscall"

	"github.com/containerd/containerd/v2/pkg/shim"
	"github.com/containerd/plugin/registry"
	"github.com/moby/sys/reexec"
	"github.com/sirupsen/logrus"
	"gitlab.com/tozd/go/errors"

	"github.com/walteh/ec1/cmd/containerd-shim-harpoon-v2/containerd"
	"github.com/walteh/ec1/pkg/logging"
	"github.com/walteh/ec1/pkg/logging/logrusshim"
)

func ShimReexecInit() {
	reexec.Register(ShimSimlinkPath(), ShimMain)
}

func (s *DevContainerdServer) setupShim(ctx context.Context) error {

	os.MkdirAll(filepath.Dir(ShimSimlinkPath()), 0755)

	proxySock, err := net.Listen("unix", ShimLogProxySockPath())
	if err != nil {
		slog.Error("Failed to create log proxy socket", "error", err, "path", ShimLogProxySockPath())
		os.Exit(1)
	}

	// fwd logs from the proxy socket to stdout
	go func() {
		defer proxySock.Close()
		for {
			conn, err := proxySock.Accept()
			if err != nil {
				slog.Error("Failed to accept log proxy connection", "error", err)
				return
			}
			go func() { _, _ = io.Copy(os.Stdout, conn) }()
		}
	}()

	// Set up logging for TestMain

	self, _ := os.Executable()

	if err := os.Symlink(self, ShimSimlinkPath()); err != nil {
		slog.Error("create shim link", "error", err)
		os.Exit(1)
	}

	oldPath := os.Getenv("PATH")
	newPath := filepath.Dir(ShimSimlinkPath()) + string(os.PathListSeparator) + oldPath
	os.Setenv("PATH", newPath)

	return nil
}

func ShimMain() {

	err := RunShim(context.Background())
	if err != nil {
		slog.Error("shim main failed", "error", err)
		os.Exit(1)
	}

}

func RunShim(ctx context.Context) error {

	// create slog writer that writes to the log proxy socket
	proxySock, err := net.Dial("unix", ShimLogProxySockPath())
	if err != nil {
		return errors.Errorf("dial %s: %w", ShimLogProxySockPath(), err)
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

	if syscall.Getppid() == 1 {
		// our parent died, we probably called ourselves directly
	}

	slog.Info("shim starting with args", "args", os.Args[1:], "my_pid", syscall.Getpid(), "my_parent_pid", syscall.Getppid())

	registry.Reset()
	containerd.RegisterPlugins()
	shim.Run(ctx, containerd.NewManager(shimRuntimeID), func(c *shim.Config) {
		c.NoReaper = true
		c.NoSetupLogger = true
	})

	return nil
}
