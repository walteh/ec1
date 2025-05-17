package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"connectrpc.com/connect"
	"github.com/apex/log"
	"github.com/containers/winquit/pkg/winquit"
	"github.com/lmittmann/tint"
	slogctx "github.com/veqryn/slog-context"
	ec1v1 "github.com/walteh/ec1/gen/proto/golang/ec1/v1poc1"
	"github.com/walteh/ec1/gen/proto/golang/ec1/v1poc1/v1poc1connect"
	"github.com/walteh/ec1/pkg/cloud/management"
	"golang.org/x/sync/errgroup"
)

// step 1: start up the management server (mac, native)
// step 2: start up the management server local agent (mac, native)
// step 3: create the qcow2 image, initalize it (run cloud-init) with cloud-init, lets use alpine nocloud for simplicity (we download it as needed)
// step 4: instruct the agent to start the vm
// step 5: the agent should ssh into the vm and install the agent
// step 6: start the agent
// - use systemd to start the agent
// - make sure the agent is running and has registerd the probe with the management server
// - at this point, we should have two agents running, one as part of our management server, and one as part of the nested vm

// step 7: start a nested vm
// - use the already created qcow2 image
// - copy the image to the agent vm
// - instruct the agent to start the vm
// step 8: instruct the agent to run a web server that returns "hello world"

// via a raw http request on the host from the maangement server running program, access the web server on the nested vm and print the response

// // If the user provides a log-file, we re-direct log messages
// // from logrus to the file
// if cfg.logFile != "" {
// 	lf, err := os.Create(cfg.logFile)
// 	if err != nil {
// 		return errors.Errorf("opening log file %s: %w", cfg.logFile, err)
// 	}
// 	defer func() {
// 		if err := lf.Close(); err != nil {
// 			slog.WarnContext(ctx, "closing log file", "error", err)
// 		}
// 	}()
// 	log.SetOutput(lf)

// 	// If debug is set, lets seed the log file with some basic information
// 	// about the environment and how it was called
// 	log.Debugf("gvnet version: %q", version.String())
// 	log.Debugf("os: %q arch: %q", runtime.GOOS, runtime.GOARCH)
// 	log.Debugf("command line: %q", os.Args)
// }

func main() {
	// Parse command line flags
	var (
		// mgtAddr    = flag.String("mgt", "http://localhost:9090", "Management server address")
		// localAgent = flag.String("agent", "localhost:9091", "Local agent address")
		clean = flag.Bool("clean", false, "Clean up before starting (removes existing binaries and images)")
		// diskPath   = flag.String("disk", "", "Path to disk image")
	)
	flag.Parse()

	// Create a context that is canceled on interrupt signals
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	w := os.Stdout

	// create a new logger
	logger := tint.NewHandler(w, &tint.Options{
		Level:      slog.LevelDebug,
		TimeFormat: "2006-01-02 15:04 05.0000",
		AddSource:  true,
	}).WithGroup("main")

	// 	// Create a PID file if requested
	// if len(cfg.pidFile) > 0 {
	// 	f, err := os.Create(cfg.pidFile)
	// 	if err != nil {
	// 		return errors.Errorf("creating pid file %s: %w", cfg.pidFile, err)
	// 	}
	// 	// Remove the pid-file when exiting
	// 	defer func() {
	// 		if err := os.Remove(cfg.pidFile); err != nil {
	// 			slog.ErrorContext(ctx, "removing pid file", "error", err)
	// 		}
	// 	}()
	// 	pid := os.Getpid()
	// 	if _, err := f.WriteString(strconv.Itoa(pid)); err != nil {
	// 		return errors.Errorf("writing pid file %s: %w", cfg.pidFile, err)
	// 	}
	// }

	mylogger := slog.New(slogctx.NewHandler(logger, nil))
	slog.SetDefault(mylogger)
	ctx = slogctx.NewCtx(ctx, mylogger)

	// Set up signal handling
	// Setup signal channel for catching user signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	// if debug {
	// 	log.SetLevel(log.DebugLevel)
	// }

	// Intercept WM_QUIT/WM_CLOSE events if on Windows as SIGTERM (noop on other OSs)
	winquit.SimulateSigTermOnQuit(sigChan)

	groupErrs := errgroup.Group{}

	groupErrs.Go(func() error {
		select {
		case <-sigChan:
			fmt.Println("\nReceived termination signal, shutting down...")
			cancel()
			return errors.New("signal caught")
		case <-ctx.Done():
			return nil
		}
	})

	// Check if we need to clean up first
	if *clean {
		fmt.Println("Cleaning up previous run...")
		cleanupPreviousRun()
	}

	groupErrs.Go(func() error {
		select {
		// Catch signals so exits are graceful and defers can run
		case <-sigChan:
			cancel()
			return errors.New("signal caught")
		case <-ctx.Done():
			return nil
		}
	})

	groupErrs.Go(func() error {
		return runEC1Demo(ctx)
	})

	// Wait for all of the go funcs to finish up
	if err := groupErrs.Wait(); err != nil && err != context.Canceled {
		log.Errorf("gvnet exiting: %v", err)
		// exitCode = 1
		os.Exit(1)
	}

}
func ptr[T any](v T) *T {
	return &v
}

type DemoContext struct {
	ManagementClient v1poc1connect.ManagementServiceClient
}

// runEC1Demo runs the complete EC1 nested virtualization demo
func runEC1Demo(ctx context.Context) error {
	fmt.Println("Starting EC1 Nested Virtualization Demo")
	fmt.Println("=======================================")

	// Create a context with cancellation for the servers
	serverCtx, cancelServers := context.WithCancel(ctx)
	defer cancelServers()

	// Use a WaitGroup to ensure servers have started before proceeding

	// Step 1: Start management server in-process
	fmt.Println("\n=== Step 1: Starting Management Server ===")

	mgr, err := management.New(serverCtx, management.ServerConfig{})
	if err != nil {
		return fmt.Errorf("failed to create management server: %w", err)
	}

	serve, err := mgr.Start(ctx, "localhost:9096")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error starting management server: %v\n", err)
		os.Exit(1)
	}

	go func() {
		if err := serve(); err != nil {
			fmt.Fprintf(os.Stderr, "Error serving management server: %v\n", err)
			os.Exit(1)
		}
	}()

	_, err = mgr.InitializeLocalAgentInsideLocalVM(ctx, connect.NewRequest(&ec1v1.InitializeLocalAgentInsideLocalVMRequest{
		Qcow2ImagePath:    ptr("./build/nocloud_alpine-3.21.2-aarch64-uefi-cloudinit-r0.qcow2"),
		CloudinitMetadata: ptr(""),
		CloudinitUserdata: ptr(""),
		Arch:              ptr("arm64"),
		Os:                ptr("linux"),
	}))
	if err != nil {
		return fmt.Errorf("failed to initialize local agent inside local VM: %w", err)
	}

	// The demo is complete, but we'll keep the management server and agent running
	// until the context is canceled (e.g., by pressing Ctrl+C)
	<-ctx.Done()

	return nil
}

// cleanupPreviousRun cleans up binaries and artifacts from previous runs
func cleanupPreviousRun() {
	// Try to remove binary files
	_ = os.RemoveAll("bin/mgt")
	_ = os.RemoveAll("bin/agent")
	_ = os.RemoveAll("bin/agent-linux")

	// Try to remove image files
	_ = os.RemoveAll("images")
}
