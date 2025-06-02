package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"testing"

	"github.com/containerd/containerd/v2/pkg/shim"
	"gitlab.com/tozd/go/errors"

	reexec "github.com/moby/sys/reexec"

	"github.com/walteh/ec1/cmd/containerd-shim-harpoon-v1/containerd"
	"github.com/walteh/ec1/pkg/logging"
)

const reexecPath = "/tmp/" + shimName
const reexecSockPath = "/tmp/" + shimName + ".sock"

func TestMain(m *testing.M) {
	// Set up logging for TestMain
	ctx := logging.SetupSlogSimpleToWriter(context.Background(), os.Stderr, true)

	slog.InfoContext(ctx, "TestMain starting", "reexecPath", reexecPath, "reexecSockPath", reexecSockPath)

	self, _ := os.Executable() // absolute path to the test binary
	slog.InfoContext(ctx, "Test binary path", "path", self)

	// shimLink = filepath.Join(os.TempDir(), shimName)
	_ = os.RemoveAll(reexecPath) // idempotent
	slog.InfoContext(ctx, "Removed existing reexec path", "path", reexecPath)

	if err := os.Symlink(self, reexecPath); err != nil {
		slog.Error("Failed to create shim link", "error", err, "self", self, "reexecPath", reexecPath)
		log.Fatalf("create shim link: %v", err)
	}
	slog.InfoContext(ctx, "Created symlink", "from", self, "to", reexecPath)

	// Set shimBinary to the symlinked path

	// 4.  (Optional) prepend dir to PATH so containerd can exec it.
	oldPath := os.Getenv("PATH")
	newPath := filepath.Dir(reexecPath) + string(os.PathListSeparator) + oldPath
	os.Setenv("PATH", newPath)
	slog.InfoContext(ctx, "Updated PATH", "oldPath", oldPath, "newPath", newPath)

	// 5.  Run all tests.
	slog.InfoContext(ctx, "Starting test execution")
	code := m.Run()
	slog.InfoContext(ctx, "Test execution completed", "exitCode", code)

	// 6.  Clean‑up *after* all shims have finished.
	_ = os.Remove(reexecPath)
	_ = os.Remove(reexecSockPath)
	slog.InfoContext(ctx, "Cleanup completed", "removedPaths", []string{reexecPath, reexecSockPath})
	os.Exit(code)
}

func init() {
	ctx := context.Background()
	// Set up basic logging for init
	slog.InfoContext(ctx, "init() called", "args", os.Args, "pid", os.Getpid())

	reexec.Register(reexecPath, shimProxyWithCleanup)
	slog.InfoContext(ctx, "Registered reexec handler", "path", reexecPath)

	if reexec.Init() {
		slog.InfoContext(ctx, "reexec.Init() returned true, exiting")
		os.Exit(0)
	} else {
		slog.InfoContext(ctx, "reexec.Init() returned false, starting shim server")
		shimShimServer()
	}
}

func handleOneShim(c net.Conn) {
	// Set up logging context
	ctx := logging.SetupSlogSimpleToWriter(context.Background(), os.Stderr, true)

	slog.InfoContext(ctx, "handleOneShim started", "remoteAddr", c.RemoteAddr(), "localAddr", c.LocalAddr())
	defer func() {
		if err := c.Close(); err != nil {
			slog.Error("Failed to close connection", "error", err)
		}
		slog.InfoContext(ctx, "handleOneShim completed, connection closed")
	}()

	if err := handleOneShimInternal(ctx, c); err != nil {
		slog.Error("handleOneShim failed", "error", err)
	}
}

func handleOneShimInternal(ctx context.Context, c net.Conn) error {
	decoder := json.NewDecoder(c)

	var meta RequestMetadata

	slog.InfoContext(ctx, "Decoding metadata from connection")
	if err := decoder.Decode(&meta); err != nil {
		return errors.Errorf("decoding metadata: %w", err)
	}

	slog.InfoContext(ctx, "Decoded metadata",
		"pid", meta.Pid,
		"argv", meta.Argv,
		"stdin", meta.Stdin,
		"stdout", meta.Stdout,
		"stderr", meta.Stderr,
		"responseFile", meta.ResponseFile)

	slog.InfoContext(ctx, "Calling shimMain()")
	if err := shimMainWithErrorHandling(ctx, meta); err != nil {
		return errors.Errorf("shimMain failed: %w", err)
	}
	slog.InfoContext(ctx, "shimMain() completed")

	responseConn, err := net.Dial("unix", meta.ResponseFile)
	if err != nil {
		return errors.Errorf("dial response file: %w", err)
	}
	defer responseConn.Close()

	// returnFile := os.NewFile(uintptr(meta.ResponseFd), "response-return")
	// defer returnFile.Close()

	// slog.InfoContext(ctx, "Creating response connection", "responseFd", meta.ResponseFd)
	// responseConn, err := net.FileConn(returnFile)
	// if err != nil {
	// 	return errors.Errorf("creating response connection: %w", err)
	// }
	// defer func() {
	// 	if err := responseConn.Close(); err != nil {
	// 		slog.Error("Failed to close response connection", "error", err)
	// 	}
	// }()

	encoder := json.NewEncoder(responseConn)

	response := struct {
		Pid      int `json:"pid"`
		ExitCode int `json:"exitCode"`
	}{os.Getpid(), 0}

	slog.InfoContext(ctx, "Encoding response", "response", response)
	if err := encoder.Encode(response); err != nil {
		return errors.Errorf("encoding response: %w", err)
	}
	slog.InfoContext(ctx, "Response sent successfully")

	return nil
}

func shimShimServer() {
	// Set up logging context
	ctx := logging.SetupSlogSimpleToWriter(context.Background(), os.Stderr, true)

	slog.InfoContext(ctx, "Starting shim server", "sockPath", reexecSockPath)

	// create a unix socket in the temp dir

	// remove the socket if it exists
	_ = os.Remove(reexecSockPath)

	// inside your existing process
	ln, err := net.Listen("unix", reexecSockPath) // e.g. /tmp/harp.sock
	if err != nil {
		slog.Error("Failed to listen on socket", "error", err, "sockPath", reexecSockPath)
		log.Fatalf("listen: %v", err)
	}
	slog.InfoContext(ctx, "Listening on unix socket", "sockPath", reexecSockPath)

	go func() {
		defer func() {
			if err := ln.Close(); err != nil {
				slog.Error("Failed to close listener", "error", err)
			}
			slog.InfoContext(ctx, "Shim server listener closed")
		}()

		slog.InfoContext(ctx, "Starting accept loop")
		for {
			slog.InfoContext(ctx, "Waiting for connection")
			c, err := ln.Accept()
			if err != nil {
				slog.Error("Accept failed", "error", err)
				return
			}
			slog.InfoContext(ctx, "Accepted connection", "remoteAddr", c.RemoteAddr())
			go handleOneShim(c) // 100% async; no extra threads
		}
	}()
	slog.InfoContext(ctx, "Shim server started successfully")
}

func shimProxyWithCleanup() {
	ctx := context.Background()

	err := shimProxy(ctx)
	if err != nil {
		slog.Error("shimProxy failed", "error", err)
		os.Exit(1)
	}

	os.Exit(0)
}

func shimProxy(ctx context.Context) error {

	userDir, err := os.UserHomeDir()
	if err != nil {
		return errors.Errorf("get user home dir: %w", err)
	}

	// open log file
	logPath := filepath.Join(userDir, "Developer/github/walteh/ec1/.logs/shim.log")
	logfile, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return errors.Errorf("open log file: %w", err)
	}
	defer func() {
		if err := logfile.Close(); err != nil {
			slog.Error("Failed to close log file", "error", err)
		}
	}()

	fmt.Fprintf(logfile, "\n\n--------------------------------\n\n")
	fmt.Fprintf(logfile, "[shim PROXY] pid=%d argv=%q\n\n", os.Getpid(), os.Args)
	// Set up logging context
	ctx = logging.SetupSlogSimpleToWriter(ctx, logfile, false)

	slog.InfoContext(ctx, "shimProxy started", "args", os.Args, "pid", os.Getpid())

	if len(os.Args) < 2 {
		slog.Error("Insufficient arguments", "args", os.Args)
		return fmt.Errorf("usage: shim-shim /path/to/real-shim [args]")
	}

	myTempResponseDir, err := os.MkdirTemp("", "response-return")
	if err != nil {
		return errors.Errorf("create temp response file: %w", err)
	}
	defer os.RemoveAll(myTempResponseDir)

	responseFile := filepath.Join(myTempResponseDir, "response-return.sock")
	responseListener, err := net.ListenUnix("unix", &net.UnixAddr{Name: responseFile, Net: "unix"})
	if err != nil {
		return errors.Errorf("listen on response file: %w", err)
	}
	defer responseListener.Close()

	slog.InfoContext(ctx, "Connecting to shim server", "sockPath", reexecSockPath)
	conn, err := net.Dial("unix", reexecSockPath)
	if err != nil {
		slog.Error("Failed to connect to shim server", "error", err, "sockPath", reexecSockPath)
		return errors.Errorf("connect to shim server: %w", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			slog.Error("Failed to close connection to shim server", "error", err)
		}
	}()
	slog.InfoContext(ctx, "Connected to shim server")

	encoder := json.NewEncoder(conn)

	// for stdin, just read and send a payload

	socks := map[string]int{
		// "stdin":  int(os.Stdin.Fd()),
		"stdout": int(os.Stdout.Fd()),
		"stderr": int(os.Stderr.Fd()),
	}
	slog.InfoContext(ctx, "Original file descriptors", "socks", socks)

	// Track file descriptors for cleanup
	var createdFds []int
	defer func() {
		for _, fd := range createdFds {
			if err := syscall.Close(fd); err != nil {
				slog.Error("Failed to close created file descriptor", "error", err, "fd", fd)
			}
		}
	}()

	for name, sourceFd := range socks {
		slog.InfoContext(ctx, "Creating socket proxy", "name", name, "sourceFd", sourceFd)
		// create a socket proxy
		targetFd, err := syscall.Socket(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
		if err != nil {
			slog.Error("Failed to create socket", "error", err, "name", name)
			return errors.Errorf("create socket: %w", err)
		}
		createdFds = append(createdFds, targetFd)

		err = syscall.Dup2(sourceFd, targetFd)
		if err != nil {
			slog.Error("Failed to dup2", "error", err, "sourceFd", sourceFd, "targetFd", targetFd)
			return errors.Errorf("dup2: %w", err)
		}

		socks[name] = targetFd
		slog.InfoContext(ctx, "Socket proxy created", "name", name, "sourceFd", sourceFd, "targetFd", targetFd)
	}

	// // create a new response socket
	// slog.InfoContext(ctx, "Creating response socket")
	// responseFd, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
	// if err != nil {
	// 	slog.Error("Failed to create response socket", "error", err)
	// 	return errors.Errorf("create response socket: %w", err)
	// }
	// defer func() {
	// 	if err := syscall.Close(responseFd[0]); err != nil {
	// 		slog.Error("Failed to close response socket 0", "error", err)
	// 	}
	// 	if err := syscall.Close(responseFd[1]); err != nil {
	// 		slog.Error("Failed to close response socket 1", "error", err)
	// 	}
	// }()

	// responseConnectionFd := responseFd[0]
	// responseListenerFd := responseFd[1]

	// listenerFile := os.NewFile(uintptr(responseListenerFd), "response")
	// defer func() {
	// 	if err := listenerFile.Close(); err != nil {
	// 		slog.Error("Failed to close listener file", "error", err)
	// 	}
	// }()

	// listenerConn, err := net.FileConn(listenerFile)
	// if err != nil {
	// 	slog.Error("Failed to create file connection", "error", err, "responseFd", responseFd)
	// 	return errors.Errorf("create file connection: %w", err)
	// }
	// defer func() {
	// 	if err := listenerConn.Close(); err != nil {
	// 		slog.Error("Failed to close listener connection", "error", err)
	// 	}
	// }()

	slog.InfoContext(ctx, "Response socket created", "responseFile", responseFile)

	// send header
	header := RequestMetadata{os.Getpid(), os.Args[1:], socks["stdin"], socks["stdout"], socks["stderr"], responseFile}

	slog.InfoContext(ctx, "Sending header", "header", header)
	err = encoder.Encode(header)
	if err != nil {
		slog.Error("Failed to encode header", "error", err, "header", header)
		return errors.Errorf("encode header: %w", err)
	}
	slog.InfoContext(ctx, "Header sent successfully")

	responseConn, err := responseListener.Accept()
	if err != nil {
		slog.Error("Failed to accept response connection", "error", err)
		return errors.Errorf("accept response connection: %w", err)
	}
	defer responseConn.Close()

	// listen for response on the file listener

	decoder := json.NewDecoder(responseConn)

	var meta struct {
		Pid      int `json:"pid"`
		ExitCode int `json:"exitCode"`
	}

	slog.InfoContext(ctx, "Decoding response")
	err = decoder.Decode(&meta)
	if err != nil {
		slog.Error("Failed to decode response", "error", err)
		return errors.Errorf("decode response: %w", err)
	}
	slog.InfoContext(ctx, "Response decoded", "response", meta)
	slog.InfoContext(ctx, "shimProxy completed successfully")

	return nil
}

var onlyOnceShimAtATime = sync.Mutex{}

type RequestMetadata struct {
	Pid          int      `json:"pid"`
	Argv         []string `json:"argv"`
	Stdin        int      `json:"stdin"`
	Stdout       int      `json:"stdout"`
	Stderr       int      `json:"stderr"`
	ResponseFile string   `json:"responseFile"`
}

// shimMainWithErrorHandling wraps shimMain with proper error handling
func shimMainWithErrorHandling(ctx context.Context, meta RequestMetadata) error {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("shimMain panicked", "panic", r)
		}
	}()

	shimMain(ctx, meta)
	return nil
}

// --- shim entry ---------------------------------------------------------
func shimMain(ctx context.Context, meta RequestMetadata) {
	onlyOnceShimAtATime.Lock()
	defer onlyOnceShimAtATime.Unlock()

	slog.InfoContext(ctx, "shimMain started",
		"pid", os.Getpid(),
		"argv", os.Args)

	slog.InfoContext(ctx, "Creating containerd manager")
	mgr := containerd.NewManager(testRuntime)
	slog.InfoContext(ctx, "Containerd manager created")

	slog.InfoContext(ctx, "Starting shim.Run")

	// Configure shim with proper error handling
	shim.Run(ctx, mgr, func(config *shim.Config) {
		config.NoReaper = true
		config.NoSubreaper = true
		config.Stdin = os.NewFile(uintptr(meta.Stdin), "stdin")
		config.Stdout = os.NewFile(uintptr(meta.Stdout), "stdout")
		config.ExitFunc = func(code int) {
			slog.InfoContext(ctx, "Shim exit function called", "code", code)
			// Don't call os.Exit here as it would terminate the test process
			// Instead, let the function return normally
		}
		config.WithArgs = meta.Argv
		slog.InfoContext(ctx, "Shim config set", "noReaper", config.NoReaper, "args", meta.Argv)
	})

	slog.InfoContext(ctx, "shim.Run completed")
}

// TestShimProxyErrorHandling tests the error handling and cleanup mechanisms
func TestShimProxyErrorHandling(t *testing.T) {
	ctx := logging.SetupSlogSimpleToWriter(context.Background(), os.Stderr, true)

	slog.InfoContext(ctx, "Starting TestShimProxyErrorHandling")

	// Test invalid arguments
	t.Run("invalid_arguments", func(t *testing.T) {
		oldArgs := os.Args
		defer func() { os.Args = oldArgs }()

		os.Args = []string{"test"}

		err := shimProxy(ctx)
		if err == nil {
			t.Error("Expected error for invalid arguments, got nil")
		}

		slog.InfoContext(ctx, "Invalid arguments test completed", "error", err)
	})

	// Test connection failure (when server is not running)
	t.Run("connection_failure", func(t *testing.T) {
		oldArgs := os.Args
		defer func() { os.Args = oldArgs }()

		os.Args = []string{"test", "arg1", "arg2"}

		// Ensure the socket doesn't exist
		_ = os.Remove(reexecSockPath)

		err := shimProxy(ctx)
		if err == nil {
			t.Error("Expected error for connection failure, got nil")
		}

		slog.InfoContext(ctx, "Connection failure test completed", "error", err)
	})

	slog.InfoContext(ctx, "TestShimProxyErrorHandling completed")
}
