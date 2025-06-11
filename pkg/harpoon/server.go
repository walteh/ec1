package harpoon

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"strings"
	"syscall"
	"time"

	"gitlab.com/tozd/go/errors"
	"golang.org/x/sys/unix"

	"github.com/containerd/containerd/v2/pkg/oci"

	harpoonv1 "github.com/walteh/ec1/gen/proto/golang/harpoon/v1"
	"github.com/walteh/ec1/pkg/ec1init"
	"github.com/walteh/ec1/pkg/logging"
	"github.com/walteh/ec1/pkg/logging/valuelog"
)

type GuestService struct {
	forwarder GuestStdioForwarder
	// currentContainerEntrypoint []string
}

var (
	_ harpoonv1.TTRPCGuestServiceService = &GuestService{}
)

func NewAgentService(forwarder GuestStdioForwarder) *GuestService {
	return &GuestService{
		forwarder: forwarder,
	}
}

func (s *GuestService) WrapWithErrorLogging() harpoonv1.TTRPCGuestServiceService {
	return WrapGuestServiceWithErrorLogging(s)
}

func (s *GuestService) Readiness(ctx context.Context, req *harpoonv1.ReadinessRequest) (*harpoonv1.ReadinessResponse, error) {
	return harpoonv1.NewReadinessResponse(func(b *harpoonv1.ReadinessResponse_builder) {
		b.Ready = ptr(true)
	}), nil
}

func (s *GuestService) Run(ctx context.Context, req *harpoonv1.RunRequest) (resp *harpoonv1.RunResponse, err error) {
	var spec *oci.Spec
	specd, err := os.ReadFile(filepath.Join(ec1init.Ec1AbsPath, ec1init.ContainerSpecFile))
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, errors.Errorf("reading spec: %w", err)
		}
	} else {
		err = json.Unmarshal(specd, &spec)
		if err != nil {
			return nil, errors.Errorf("unmarshalling spec: %w", err)
		}
	}
	// var stdin io.Reader
	// if req.GetStdin() != nil {
	// 	stdin = bytes.NewBuffer(req.GetStdin())
	// } else {
	// 	stdin = bytes.NewReader([]byte{})
	// }
	// stdout := bytes.NewBuffer(nil)
	// stderr := bytes.NewBuffer(nil)

	// stdoutFifo, err := fifo.OpenFifo(ctx, ec1init.DevStdoutPort, os.O_RDWR|syscall.O_NONBLOCK 0)
	// if err != nil {
	// 	return nil, errors.Errorf("opening stdout fifo: %w", err)
	// }

	// stderrFifo, err := fifo.OpenFifo(ctx, ec1init.DevStderrPort, os.O_RDWR|syscall.O_NONBLOCK 0)
	// if err != nil {
	// 	return nil, errors.Errorf("opening stderr fifo: %w", err)
	// }

	// make sure our vport exists
	// ExecCmdForwardingStdio(ctx, "ls", "-lah", ec1init.DevStdoutPort)
	// ExecCmdForwardingStdio(ctx, "ls", "-lah", ec1init.DevStderrPort)

	// stdoutFifo, err := os.OpenFile(ec1init.DevStdoutPort, os.O_WRONLY|syscall.O_NONBLOCK, 0)
	// if err != nil {
	// 	return nil, errors.Errorf("opening stdout fifo: %w", err)
	// }
	// stderrFifo, err := os.OpenFile(ec1init.DevStderrPort, os.O_WRONLY|syscall.O_NONBLOCK, 0)
	// if err != nil {
	// 	return nil, errors.Errorf("opening stderr fifo: %w", err)
	// }

	// stdoutFifo, err := OpenSerialPort(ctx, ec1init.DevStdoutPort)
	// if err != nil {
	// 	return nil, errors.Errorf("opening stdout port: %w", err)
	// }
	// stderrFifo, err := OpenSerialPort(ctx, ec1init.DevStderrPort)
	// if err != nil {
	// 	return nil, errors.Errorf("opening stderr port: %w", err)
	// }

	// stdoutFifo, err := openVirtioPortWithRetry(ctx, ec1init.DevStdoutPort)
	// if err != nil {
	// 	return nil, errors.Errorf("opening stdout port: %w", err)
	// }
	// stderrFifo, err := openVirtioPortWithRetry(ctx, ec1init.DevStderrPort)
	// if err != nil {
	// 	return nil, errors.Errorf("opening stderr port: %w", err)
	// }

	defer func() {
		if r := recover(); r != nil {
			slog.ErrorContext(ctx, "panic in Run", "error", r)
			resp = nil
			err = errors.Errorf("panic in Run: %v", r)
		}
		slog.InfoContext(ctx, "Run finished")
	}()

	argc := spec.Process.Args[0]
	argv := spec.Process.Args[1:]

	command := exec.CommandContext(ctx, argc, argv...)

	logwr := logging.GetDefaultLogWriter()

	command.Stdout = io.MultiWriter(logwr, s.forwarder.Stdout())
	command.Stderr = io.MultiWriter(logwr, s.forwarder.Stderr())
	command.Stdin = s.forwarder.Stdin()
	// command.Stdin = stdinFifo

	slog.InfoContext(ctx, "running command", "argc", argc, "argv", argv)

	err = command.Run()

	exitCode := 0

	slog.InfoContext(ctx, "command finished", "err", err, "exitCode", exitCode)

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return nil, errors.Errorf("harpoon: running command [%s %s]: %w", argc, strings.Join(argv, " "), err)
		}
	}

	resp, err = harpoonv1.NewValidatedRunResponse(func(b *harpoonv1.RunResponse_builder) {
		b.ExitCode = ptr(int32(exitCode))
	})
	if err != nil {
		return nil, errors.Errorf("building run response: %w", err)
	}

	return resp, nil
}

// func openVirtioPortWithRetry(ctx context.Context, devicePath string) (*os.File, error) {
// 	backoff := 50 * time.Millisecond
// 	maxBackoff := 2 * time.Second
// 	maxRetries := 10

// 	for i := 0; i < maxRetries; i++ {
// 		file, err := os.OpenFile(devicePath, os.O_WRONLY|syscall.O_NONBLOCK, 0)
// 		if err == nil {
// 			return file, nil
// 		}

// 		// If it's not the "no such device" error, fail immediately
// 		if !strings.Contains(err.Error(), "no such device") {
// 			return nil, err
// 		}

// 		slog.DebugContext(ctx, "retrying virtio port open",
// 			"device", devicePath,
// 			"attempt", i+1,
// 			"backoff", backoff,
// 			"error", err)

// 		select {
// 		case <-ctx.Done():
// 			return nil, ctx.Err()
// 		case <-time.After(backoff):
// 			// Exponential backoff
// 			backoff = min(backoff*2, maxBackoff)
// 		}
// 	}

// 	return nil, errors.Errorf("failed to open %s after %d retries", devicePath, maxRetries)
// }

func waitForVirtioPortActive(ctx context.Context, devicePath string) error {
	// Extract port name from device path
	portName := strings.TrimPrefix(devicePath, "/dev/")
	activePath := fmt.Sprintf("/sys/class/virtio-ports/%s/active", portName)

	backoff := 50 * time.Millisecond
	maxBackoff := 2 * time.Second
	maxRetries := 10

	for i := 0; i < maxRetries; i++ {
		// Check if port is active in sysfs
		if data, err := os.ReadFile(activePath); err == nil {
			active := strings.TrimSpace(string(data))
			if active == "1" {
				// Port is active, try to open
				file, err := os.OpenFile(devicePath, os.O_WRONLY|syscall.O_NONBLOCK, 0)
				if err == nil {
					file.Close()
					return nil
				}
			}
		}

		slog.DebugContext(ctx, "waiting for virtio port to become active",
			"device", devicePath,
			"attempt", i+1,
			"backoff", backoff)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
			backoff = min(backoff*2, maxBackoff)
		}
	}

	return errors.Errorf("virtio port %s never became active", devicePath)
}

func openVirtioPortWithRetry(ctx context.Context, devicePath string) (*os.File, error) {
	// First wait for the port to become active
	if err := waitForVirtioPortActive(ctx, devicePath); err != nil {
		return nil, err
	}

	// Then open the device
	file, err := os.OpenFile(devicePath, os.O_WRONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, errors.Errorf("opening virtio port %s: %w", devicePath, err)
	}

	return file, nil
}

func (s *GuestService) TimeSync(ctx context.Context, req *harpoonv1.TimeSyncRequest) (*harpoonv1.TimeSyncResponse, error) {

	nowNano := uint64(time.Now().UnixNano())
	updateNano := uint64(req.GetUnixTimeNs())

	tv := unix.NsecToTimeval(int64(updateNano))

	if err := unix.Settimeofday(&tv); err != nil {
		slog.ErrorContext(ctx, "Settimeofday failed", "error", err)
		return nil, errors.Errorf("unix.Settimeofday failed: %w", err)
	}

	offset := int64(nowNano) - int64(updateNano)

	slog.InfoContext(ctx, "time sync", "update", time.Unix(0, int64(updateNano)).UTC().Format(time.RFC3339), "ns_diff", time.Duration(offset))

	return harpoonv1.NewTimeSyncResponse(func(b *harpoonv1.TimeSyncResponse_builder) {
		b.PreviousTimeNs = &nowNano
	}), nil
}

func (s *GuestService) Exec(ctx context.Context, server harpoonv1.TTRPCGuestService_ExecServer) (err error) {

	slog.InfoContext(ctx, "exec request received")

	defer func() {
		if r := recover(); r != nil {
			slog.ErrorContext(ctx, "exec request panicked", "error", r)
			fmt.Fprintln(logging.GetDefaultLogWriter(), string(debug.Stack()))
			err = errors.New("exec request panicked")
			return
		}
	}()

	req, err := server.Recv()
	if err != nil {
		return errors.Errorf("receiving request: %w", err)
	}

	start := req.GetStart()
	if start == nil {
		return errors.New("start request is required")
	}

	stdin := bytes.NewBuffer(nil)
	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)

	errch := make(chan error)
	stdinDone := make(chan struct{})
	stdoutDone := make(chan struct{})
	stderrDone := make(chan struct{})
	cmdDone := make(chan struct{})
	terminateDone := make(chan struct{})

	errs := []error{}
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.ErrorContext(ctx, "err goroutine panicked", "error", r)
				panic(r)
			}
		}()
		for err := range errch {
			slog.ErrorContext(ctx, "err goroutine received error", "error", err)
			errs = append(errs, err)
			er, err := harpoonv1.NewValidatedExecResponse_WithError(func(b *harpoonv1.ExecResponse_Error_builder) {
				b.Error = ptr(err.Error())
			})
			if err != nil {
				slog.ErrorContext(ctx, "building error response", "error", err)
				errs = append(errs, err)
				continue
			}
			err = server.Send(er)
			if err != nil {
				slog.ErrorContext(ctx, "sending error response", "error", err)
				errs = append(errs, err)
				continue
			}
		}
	}()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.ErrorContext(ctx, "stdin goroutine panicked", "error", r)
				panic(r)
			}
		}()
		defer close(stdinDone)
		for {
			req, err := server.Recv()
			if err != nil {
				if errors.Is(err, io.EOF) {
					slog.DebugContext(ctx, "stdin EOF")
					return
				}
				errch <- errors.Errorf("reading stdin from client: %w", err)
				return
			}

			slog.DebugContext(ctx, "stdin request received", "req", valuelog.NewPrettyValue(req))

			if req.GetTerminate() != nil {
				close(terminateDone)
				return
			}
			if req.GetStdin() == nil {
				errch <- errors.New("stdin request is required")
				return
			}
			_, err = stdin.Write(req.GetStdin().GetData())
			if err != nil {
				errch <- errors.Errorf("writing stdin to command: %w", err)
				return
			}
			if req.GetStdin().GetDone() {
				slog.DebugContext(ctx, "stdin request done")
				return
			}
		}
	}()

	// env := make(map[string]string)
	// for k, v := range start.GetEnvVars() {
	// 	env[k] = v
	// }

	// argv := start.GetArgv()
	// argc := start.GetArgc()
	// if start.GetUseEntrypoint() {
	// 	full := append(s.currentContainerEntrypoint, append([]string{argc}, argv...)...)
	// 	argv = full[1:]
	// 	argc = full[0]
	// }

	var spec *oci.Spec
	specd, err := os.ReadFile(filepath.Join(ec1init.Ec1AbsPath, ec1init.ContainerSpecFile))
	if err != nil {
		if !os.IsNotExist(err) {
			return errors.Errorf("reading spec: %w", err)
		}
	} else {
		err = json.Unmarshal(specd, &spec)
		if err != nil {
			return errors.Errorf("unmarshalling spec: %w", err)
		}
	}

	argc := spec.Process.Args[0]
	argv := spec.Process.Args[1:]

	cmd := exec.CommandContext(ctx, argc, argv...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Stdin = stdin
	cmd.Env = spec.Process.Env
	cmd.SysProcAttr = &syscall.SysProcAttr{
		// Cloneflags: syscall.CLONE_NEWNS,
	}
	cmd.Dir = spec.Process.Cwd

	// Start stdout streaming
	go streamOutput(
		ctx,
		stdout,
		server,
		harpoonv1.NewValidatedExecResponse_WithStdout,
		errch,
		stdoutDone,
		"stdout",
	)

	// Start stderr streaming
	go streamOutput(
		ctx,
		stderr,
		server,
		harpoonv1.NewValidatedExecResponse_WithStderr,
		errch,
		stderrDone,
		"stderr",
	)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.ErrorContext(ctx, "cmd goroutine panicked", "error", r)
				panic(r)
			}
		}()
		defer close(cmdDone)
		err = cmd.Run()
		if err != nil {
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) {
				c := int32(exitErr.ExitCode())
				ec, err := harpoonv1.NewValidatedExecResponse_WithExit(func(b *harpoonv1.ExecResponse_Exit_builder) {
					b.ExitCode = &c
				})
				if err != nil {
					errch <- errors.Errorf("building exit response: %w", err)
					return
				}

				err = server.Send(ec)
				if err != nil {
					errch <- errors.Errorf("sending exit response: %w", err)
					return
				}

			} else {
				errch <- errors.Errorf("running command - non-exit error: %w", err)
			}
			return
		}
	}()

	<-cmdDone
	<-stdoutDone
	<-stderrDone
	<-stdinDone
	<-terminateDone

	close(errch)

	allerrs := errors.Join(errs...)

	if len(errs) > 0 {
		slog.ErrorContext(ctx, "exec request finished", "err", allerrs)
	}

	slog.InfoContext(ctx, "exec request finished")

	return allerrs
}

// streamOutput handles reading from a source and streaming it to the client
func streamOutput(
	ctx context.Context,
	reader io.Reader,
	server harpoonv1.TTRPCGuestService_ExecServer,
	responseBuilder func(func(*harpoonv1.Bytestream_builder)) (*harpoonv1.ExecResponse, error),
	errch chan<- error,
	done chan<- struct{},
	streamType string,
) {
	defer func() {
		if r := recover(); r != nil {
			slog.ErrorContext(ctx, "streamOutput goroutine panicked", "streamType", streamType, "error", r)
			panic(r)
		}
	}()
	defer close(done)
	for {
		buf := make([]byte, 1024)
		n, err := reader.Read(buf)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return
			}
			errch <- errors.Errorf("reading %s from command: %w", streamType, err)
			return
		}

		resp, err := responseBuilder(func(b *harpoonv1.Bytestream_builder) {
			b.Data = buf[:n]
			b.Done = ptr(false)
		})
		if err != nil {
			errch <- errors.Errorf("building %s response: %w", streamType, err)
			return
		}

		err = server.Send(resp)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return
			}
			errch <- errors.Errorf("sending %s to client: %w", streamType, err)
			return
		}
	}
}
