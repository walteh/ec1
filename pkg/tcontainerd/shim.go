package tcontainerd

import (
	"context"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"runtime/debug"
	"strconv"
	"syscall"

	"github.com/containerd/containerd/v2/pkg/shim"
	"github.com/containerd/plugin/registry"
	"github.com/moby/sys/reexec"
	"github.com/sirupsen/logrus"
	slogctx "github.com/veqryn/slog-context"

	"github.com/walteh/ec1/cmd/containerd-shim-harpoon-v2/containerd"
	"github.com/walteh/ec1/pkg/logging"
	"github.com/walteh/ec1/pkg/logging/logrusshim"
)

func ShimReexecInit() {
	reexec.Register(ShimSimlinkPath(), ShimMain)
}

// func (s *DevContainerdServer) setupShim(ctx context.Context) error {

// 	os.MkdirAll(filepath.Dir(ShimSimlinkPath()), 0755)

// 	proxySock, err := net.Listen("unix", ShimLogProxySockPath())
// 	if err != nil {
// 		slog.Error("Failed to create log proxy socket", "error", err, "path", ShimLogProxySockPath())
// 		os.Exit(1)
// 	}

// 	// fwd logs from the proxy socket to stdout
// 	go func() {
// 		defer proxySock.Close()
// 		for {
// 			conn, err := proxySock.Accept()
// 			if err != nil {
// 				slog.Error("Failed to accept log proxy connection", "error", err)
// 				return
// 			}
// 			go func() { _, _ = io.Copy(os.Stdout, conn) }()
// 		}
// 	}()

// 	// Set up logging for TestMain

// 	self, _ := os.Executable()

// 	if err := os.Symlink(self, ShimSimlinkPath()); err != nil {
// 		slog.Error("create shim link", "error", err)
// 		os.Exit(1)
// 	}

// 	oldPath := os.Getenv("PATH")
// 	newPath := filepath.Dir(ShimSimlinkPath()) + string(os.PathListSeparator) + oldPath
// 	os.Setenv("PATH", newPath)

// 	return nil
// }

func ShimMain() {

	ctx := context.Background()

	proxySock, err := net.Dial("unix", ShimLogProxySockPath())
	if err != nil {
		slog.Error("Failed to dial log proxy socket", "error", err, "path", ShimLogProxySockPath())
		return
	}
	defer proxySock.Close()

	ctx = logging.SetupSlogSimpleToWriterWithProcessName(ctx, proxySock, true, "shim")

	ctx = slogctx.Append(ctx, slog.String("process", "shim"), slog.String("pid", strconv.Itoa(os.Getpid())))

	defer func() {
		if r := recover(); r != nil {
			slog.ErrorContext(ctx, "panic", "error", r)
			panic(r)
		}
		slog.InfoContext(ctx, "shim exiting")
		slog.DebugContext(ctx, string(debug.Stack()))
	}()

	go func() {
		syschan := make(chan os.Signal, 1)
		signal.Notify(syschan, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
		<-syschan
		slog.InfoContext(ctx, "received signal, shutting down")
	}()

	err = RunShim(ctx)
	if err != nil {
		slog.Error("shim main failed", "error", err)
	}

}

func RunShim(ctx context.Context) error {

	logrusshim.SetLogrusLevel(logrus.DebugLevel)

	if syscall.Getppid() == 1 {
		// our parent died, we probably called ourselves directly
	}

	slog.InfoContext(ctx, "shim starting with args", "args", os.Args[1:], "my_pid", syscall.Getpid(), "my_parent_pid", syscall.Getppid())

	registry.Reset()
	containerd.RegisterPlugins()
	shim.Run(ctx, containerd.NewManager(shimRuntimeID), func(c *shim.Config) {
		c.NoReaper = true
		c.NoSetupLogger = true
	})

	return nil
}
