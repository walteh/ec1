package tcontainerd

import (
	"context"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"runtime"
	"runtime/debug"
	"strconv"
	"syscall"
	"time"

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

	// Set up panic and exit monitoring BEFORE anything else
	defer func() {
		if r := recover(); r != nil {
			slog.ErrorContext(ctx, "FATAL: shim main panic", "panic", r, "stack", string(debug.Stack()))
			panic(r)
		}
		slog.InfoContext(ctx, "shim exiting normally")
		slog.DebugContext(ctx, string(debug.Stack()))
	}()

	// Monitor for unexpected signals and OS-level events
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT,
		syscall.SIGKILL, syscall.SIGABRT, syscall.SIGSEGV, syscall.SIGBUS)

	go func() {
		sig := <-signalChan
		slog.ErrorContext(ctx, "FATAL: shim received unexpected signal", "signal", sig, "pid", os.Getpid())
		// Log stack trace of all goroutines
		buf := make([]byte, 64*1024)
		n := runtime.Stack(buf, true)
		slog.ErrorContext(ctx, string(buf[:n]))

		// Don't exit immediately, let the signal handler work
		time.Sleep(1 * time.Second)
	}()

	slog.InfoContext(ctx, "starting shim run")
	err = RunShim(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "FATAL: shim main failed", "error", err)
		// Don't return here, let it exit normally to see if this is the cause
	}
	slog.InfoContext(ctx, "shim run completed")
}

func RunShim(ctx context.Context) error {

	logrusshim.SetLogrusLevel(logrus.DebugLevel)

	if syscall.Getppid() == 1 {
		// our parent died, we probably called ourselves directly
		slog.WarnContext(ctx, "parent process is PID 1 - running as orphan")
	}

	slog.InfoContext(ctx, "shim starting with args", "args", os.Args[1:], "my_pid", syscall.Getpid(), "my_parent_pid", syscall.Getppid())

	// Log resource limits
	var rusage syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &rusage); err == nil {
		slog.InfoContext(ctx, "initial resource usage",
			"max_rss", rusage.Maxrss,
			"user_time", rusage.Utime,
			"sys_time", rusage.Stime)
	}

	// Start a goroutine to monitor resource usage
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				var rusage syscall.Rusage
				if err := syscall.Getrusage(syscall.RUSAGE_SELF, &rusage); err == nil {
					slog.InfoContext(ctx, "resource usage check",
						"max_rss", rusage.Maxrss,
						"user_time", rusage.Utime,
						"sys_time", rusage.Stime,
						"num_goroutines", runtime.NumGoroutine())
				}
			}
		}
	}()

	registry.Reset()
	containerd.RegisterPlugins()

	slog.InfoContext(ctx, "calling shim.Run")
	shim.Run(ctx, containerd.NewManager(shimRuntimeID), func(c *shim.Config) {
		c.NoReaper = true
		c.NoSetupLogger = true
	})
	slog.InfoContext(ctx, "shim.Run completed")

	return nil
}
