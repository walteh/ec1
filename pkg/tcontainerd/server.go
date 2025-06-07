package tcontainerd

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/containerd/containerd/v2/cmd/containerd/command"
	"gitlab.com/tozd/go/errors"

	"github.com/walteh/ec1/pkg/logging/logrusshim"
)

func init() {
	logrusshim.ForwardLogrusToSlogGlobally()
}

type DevContainerdServer struct {
	debug bool
}

func NewDevContainerdServer(ctx context.Context, debug bool) (*DevContainerdServer, error) {
	server := &DevContainerdServer{
		debug: debug,
	}

	if err := server.EnsureOnlyOneInstanceRunning(ctx); err != nil {
		return nil, errors.Errorf("ensuring only one instance is running: %w", err)
	}

	if err := server.setupReexec(ctx); err != nil {
		return nil, errors.Errorf("setting up shim: %w", err)
	}

	if err := server.setupDirectories(ctx); err != nil {
		return nil, errors.Errorf("setting up directories: %w", err)
	}

	if err := server.createConfig(ctx); err != nil {
		return nil, errors.Errorf("creating config: %w", err)
	}

	if err := server.createNerdctlConfig(ctx); err != nil {
		return nil, errors.Errorf("creating nerdctl config: %w", err)
	}

	return server, nil
}

func (s *DevContainerdServer) setupDirectories(ctx context.Context) error {
	workdir := WorkDir()
	dirs := []string{
		workdir,
		filepath.Join(workdir, "root"),
		filepath.Join(workdir, "state"),
		filepath.Join(workdir, "run"),
		filepath.Join(workdir, "snapshots"),
		filepath.Join(workdir, "content"),
		filepath.Join(workdir, "metadata"),
		filepath.Join(workdir, "fifo"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return errors.Errorf("creating directory %s: %w", dir, err)
		}
	}

	slog.InfoContext(ctx, "Created containerd directories", "workdir", workdir)
	return nil
}

func (s *DevContainerdServer) Start(ctx context.Context) error {
	slog.InfoContext(ctx, "Starting containerd server")

	// Start containerd using embedded command
	args := []string{
		"containerd",
		"--config", ContainerdConfigTomlPath(),
		"--address", Address(),
		"--state", ContainerdStateDir(),
		"--root", ContainerdRootDir(),
		"--log-level", func() string {
			if s.debug {
				return "debug"
			}
			return "info"
		}(),
	}

	slog.InfoContext(ctx, "Starting containerd with args", "args", args)

	app := command.App()
	return app.Run(args)
}

func (s *DevContainerdServer) StartBackground(ctx context.Context) error {
	// Start containerd and wait for it to be ready
	go func() {
		if err := s.Start(ctx); err != nil && err != context.Canceled {
			slog.ErrorContext(ctx, "Containerd failed in background", "error", err)
		}
	}()

	// Wait for containerd to be ready
	slog.InfoContext(ctx, "Waiting for containerd to be ready")

	startDeadline := time.Now().Add(timeout)
	for {
		if s.isReady(ctx) {
			slog.InfoContext(ctx, "Containerd is ready")
			return nil
		}
		if time.Now().After(startDeadline) {
			return errors.Errorf("timeout (%s) waiting for containerd to start", timeout)
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func (s *DevContainerdServer) Stop(ctx context.Context) {
	slog.InfoContext(ctx, "Stopping containerd server")
	os.Remove(LockFile())
	// The embedded containerd should handle shutdown gracefully when context is cancelled
}

func (s *DevContainerdServer) isReady(ctx context.Context) bool {
	conn, err := net.DialTimeout("unix", Address(), time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// func (s *DevContainerdServer) waitForContainerdReady(ctx context.Context) error {
// 	slog.InfoContext(ctx, "Waiting for containerd to be ready")

// 	timeout := time.After(s.config.Timeout)
// 	ticker := time.NewTicker(500 * time.Millisecond)
// 	defer ticker.Stop()

// 	for {
// 		select {
// 		case <-ctx.Done():
// 			return errors.Errorf("context cancelled")
// 		case <-timeout:
// 			return errors.Errorf("timeout waiting for containerd to be ready")
// 		case <-ticker.C:
// 			if s.isReady(ctx) {
// 				slog.InfoContext(ctx, "Containerd is ready")
// 				return nil
// 			}
// 		}
// 	}
// }

func (s *DevContainerdServer) PrintConnectionInfoForground() {
	fmt.Printf("\n🎉 Containerd Development Server Running!\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("📁 Work Directory: %s\n", WorkDir())
	fmt.Printf("🔌 Socket Address: %s\n", Address())
	fmt.Printf("🏷️  Default Namespace: %s\n", Namespace())
	fmt.Printf("🔧 Harpoon Runtime: %s\n", shimRuntimeID)
	fmt.Printf("🛠️  Shim Binary: %s\n", ShimSimlinkPath())
	fmt.Printf("\n📋 Useful Commands:\n")
	fmt.Printf("  # List images\n")
	fmt.Printf("  ctr --address %s --namespace %s images list\n\n", Address(), Namespace())
	fmt.Printf("  # Pull an image\n")
	fmt.Printf("  ctr --address %s --namespace %s images pull docker.io/library/alpine:latest\n\n", Address(), Namespace())
	fmt.Printf("  # Run a container with harpoon\n")
	fmt.Printf("  ctr --address %s --namespace %s run --runtime %s --rm docker.io/library/alpine:latest test echo hello\n\n", Address(), Namespace(), shimRuntimeID)
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("Press Ctrl+C to stop\n\n")
}

func (s *DevContainerdServer) PrintConnectionInfoBackground() {
	fmt.Printf("\n🎉 Containerd Development Server Running!\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("📁 Work Directory: %s\n", WorkDir())
	fmt.Printf("🔌 Socket Address: %s\n", Address())
	fmt.Printf("🏷️  Default Namespace: %s\n", Namespace())
	fmt.Printf("🔧 Harpoon Runtime: %s\n", shimRuntimeID)
	fmt.Printf("🛠️  Shim Binary: %s\n", ShimSimlinkPath())
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	// tell them how to stop it
	fmt.Printf("To stop:\n")
	fmt.Printf("  pkill -f 'containerd.*%s'\n", filepath.Base(Address()))
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
}

// we need a function that makes sure we are the only one running on this system
func (s *DevContainerdServer) EnsureOnlyOneInstanceRunning(ctx context.Context) error {
	if _, err := os.Stat(LockFile()); err == nil {
		isRunning, err := isServerRunning(ctx)
		if err != nil {
			return errors.Errorf("failed to check if server is running: %w", err)
		}
		if isRunning {
			return errors.Errorf("server is already running")
		}

		// if the process is not running, remove the lock file
		os.Remove(LockFile())
	}

	// write the pid to the lock file
	pid := os.Getpid()
	os.MkdirAll(filepath.Dir(LockFile()), 0755)
	err := os.WriteFile(LockFile(), []byte(strconv.Itoa(pid)), 0644)
	if err != nil {
		return errors.Errorf("failed to write lock file: %w", err)
	}

	// clear everything in the work dir
	os.RemoveAll(WorkDir())
	os.MkdirAll(WorkDir(), 0755)

	// write a cleanup function that will remove the lock file
	cleanup := func() {
		os.Remove(WorkDir())
		os.Remove(LockFile())
	}

	go func() {
		sigc := make(chan os.Signal, 1)
		signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)
		<-sigc
		slog.InfoContext(ctx, "shutting down")
		cleanup()
	}()

	return nil
}

func isServerRunning(ctx context.Context) (bool, error) {
	// grab the pid from the lock file
	pid, err := os.ReadFile(LockFile())
	if err != nil {
		return false, errors.Errorf("failed to read lock file: %w", err)
	}

	pidInt, err := strconv.Atoi(strings.TrimSpace(string(pid)))
	if err != nil {
		return false, errors.Errorf("failed to convert pid to int: %w", err)
	}

	// check if the pid is still running
	process, err := os.FindProcess(pidInt)
	if err != nil {
		return false, errors.Errorf("failed to find process: %w", err)
	}
	err = process.Signal(syscall.Signal(0))
	if err != nil {
		if err.Error() == "os: process already finished" {
			return false, nil
		}
		return false, errors.Errorf("failed to signal process: %w", err)
	}

	return true, nil
}
