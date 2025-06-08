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
	libdispatch "github.com/walteh/ec1/pkg/libdipatch"
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
	// runtime.LockOSThread()
	// defer runtime.UnlockOSThread()

	// if os.Getuid() == 0 {
	// 	syscall.Setuid(1000)
	// 	syscall.Setgid(1000)
	// }

	// if os.Getppid() == 1 {
	// 	// run myself as a child process and wait so i don't have pid as 1
	// 	cmd := exec.Command(os.Args[0], os.Args[1:]...)
	// 	cmd.Stdout = os.Stdout
	// 	cmd.Stderr = os.Stderr
	// 	cmd.Stdin = os.Stdin
	// 	err := cmd.Run()
	// 	if err != nil {
	// 		slog.Error("Failed to run myself as a child process", "error", err)
	// 		os.Exit(1)
	// 	}
	// 	os.Exit(0)
	// }

	ctx := context.Background()

	proxySock, err := net.Dial("unix", ShimLogProxySockPath())
	if err != nil {
		slog.Error("Failed to dial log proxy socket", "error", err, "path", ShimLogProxySockPath())
		return
	}
	defer proxySock.Close()

	ctx = logging.SetupSlogSimpleToWriterWithProcessName(ctx, proxySock, true, "shim")

	ctx = slogctx.Append(ctx, slog.String("process", "shim"), slog.String("pid", strconv.Itoa(os.Getpid())), slog.String("ppid", strconv.Itoa(syscall.Getppid())))

	slog.InfoContext(ctx, "SHIM_STARTING", "args", os.Args[1:])

	// Set up panic and exit monitoring BEFORE anything else
	defer func() {
		if r := recover(); r != nil {
			slog.ErrorContext(ctx, "SHIM_EXIT_PANIC", "panic", r, "stack", string(debug.Stack()))
			panic(r)
		}
		slog.InfoContext(ctx, "SHIM_EXIT_NORMAL")
	}()

	if syscall.Getppid() == 1 {

		// runtime.LockOSThread() // ensures main() stays on the true main thread
		// defer runtime.UnlockOSThread()
		// go func() {
		// 	defer func() {
		// 		if r := recover(); r != nil {
		// 			slog.ErrorContext(ctx, "panic in DispatchMain", "panic", r)
		// 			panic(r)
		// 		}
		// 	}()
		// 	libdispatch.DispatchMain()
		// }()

		var rusage syscall.Rusage
		if err := syscall.Getrusage(syscall.RUSAGE_SELF, &rusage); err == nil {
			slog.InfoContext(ctx, "SHIM_INITIAL_RESOURCE_USAGE",
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
						slog.InfoContext(ctx, "SHIM_RESOURCE_USAGE_CHECK",
							"max_rss", rusage.Maxrss,
							"user_time", float64(rusage.Utime.Usec)/1000000,
							"sys_time", float64(rusage.Stime.Usec)/1000000,
							"num_goroutines", runtime.NumGoroutine())
					}
				}
			}
		}()

	}

	// Monitor for unexpected signals and OS-level events
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGABRT, syscall.SIGSEGV, syscall.SIGBUS)

	go func() {
		sig := <-signalChan
		slog.ErrorContext(ctx, "SHIM_EXIT_SIGNAL", "signal", sig)
		// Log stack trace of all goroutines
		buf := make([]byte, 64*1024)
		n := runtime.Stack(buf, true)
		slog.ErrorContext(ctx, string(buf[:n]))

		// Don't exit immediately, let the signal handler work
		time.Sleep(1 * time.Second)
	}()

	errc := make(chan error)

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	go func() {
		errc <- RunShim(ctx)
	}()

	if syscall.Getppid() == 1 {
		slog.InfoContext(ctx, "SHIM_DISPATCH_MAIN")
		libdispatch.DispatchMain()
	} else {
		select {
		case err := <-errc:
			slog.ErrorContext(ctx, "SHIM_MAIN_FAILED", "error", err)
		case <-time.After(30 * time.Second):
			slog.ErrorContext(ctx, "SHIM_MAIN_TIMEOUT")
		}
	}

	// <-time.After(30 * time.Second)

}

func RunShim(ctx context.Context) error {

	logrusshim.SetLogrusLevel(logrus.DebugLevel)

	registry.Reset()
	containerd.RegisterPlugins()

	shim.Run(ctx, containerd.NewManager(shimRuntimeID), func(c *shim.Config) {
		c.NoReaper = true
		c.NoSetupLogger = true
	})

	return nil
}
