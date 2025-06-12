package harpoon

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"gitlab.com/tozd/go/errors"
	"golang.org/x/sys/unix"

	"github.com/containerd/containerd/v2/pkg/oci"

	harpoonv1 "github.com/walteh/ec1/gen/proto/golang/harpoon/v1"
	"github.com/walteh/ec1/pkg/logging"
)

type GuestService struct {
	forwarder GuestStdioForwarder
	spec      *oci.Spec
	// currentContainerEntrypoint []string
}

var (
	_ harpoonv1.TTRPCGuestServiceService = &GuestService{}
)

func NewAgentService(forwarder GuestStdioForwarder, spec *oci.Spec) *GuestService {
	return &GuestService{
		forwarder: forwarder,
		spec:      spec,
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

func (s *GuestService) RunCommand(ctx context.Context, req *harpoonv1.RunCommandRequest) (resp *harpoonv1.RunCommandResponse, err error) {
	cmd := exec.CommandContext(ctx, req.GetArgc(), req.GetArgv()...)
	vars := []string{}
	for k, v := range req.GetEnvVars() {
		vars = append(vars, k+"="+v)
	}
	cmd.Env = vars
	cmd.Dir = s.spec.Process.Cwd
	cmd.Stdin = bytes.NewBuffer(req.GetStdin())
	stdoutBuffer := bytes.NewBuffer(nil)
	stderrBuffer := bytes.NewBuffer(nil)
	cmd.Stdout = stdoutBuffer
	cmd.Stderr = stderrBuffer

	err = cmd.Run()

	exitCode := 0
	if exitErr, ok := err.(*exec.ExitError); ok {
		exitCode = exitErr.ExitCode()
	}

	resp = harpoonv1.NewRunCommandResponse(func(b *harpoonv1.RunCommandResponse_builder) {
		b.Stdout = stdoutBuffer.Bytes()
		b.Stderr = stderrBuffer.Bytes()
		b.ExitCode = ptr(int32(exitCode))
	})
	if err != nil {
		return nil, errors.Errorf("building run command response: %w", err)
	}
	return resp, nil
}

func (s *GuestService) RunSpecSignal(ctx context.Context, reqz harpoonv1.TTRPCGuestService_RunSpecSignalServer) (err error) {

	defer func() {
		if r := recover(); r != nil {
			slog.ErrorContext(ctx, "panic in Run", "error", r)
			err = errors.Errorf("panic in Run: %v", r)
		}
	}()

	reqd, err := reqz.Recv()
	if err != nil {
		return errors.Errorf("receiving signal request: %w", err)
	}
	if reqd.HasSignal() {
		return errors.Errorf("signal request must be empty to start the command")
	}

	argc := s.spec.Process.Args[0]
	argv := s.spec.Process.Args[1:]

	command := exec.CommandContext(ctx, argc, argv...)

	logwr := logging.GetDefaultLogWriter()

	command.Stdout = io.MultiWriter(logwr, s.forwarder.Stdout())
	command.Stderr = io.MultiWriter(logwr, s.forwarder.Stderr())
	command.Stdin = s.forwarder.Stdin()
	command.Env = s.spec.Process.Env
	// command.Dir = s.spec.Process.Cwd

	command.SysProcAttr = &syscall.SysProcAttr{
		// Cloneflags: syscall.CLONE_NEWNS,
	}
	// command.Stdin = stdinFifo

	slog.InfoContext(ctx, "running command", "argc", argc, "argv", argv, "dir_not_used", s.spec.Process.Cwd)

	err = command.Start()
	if err != nil {
		return errors.Errorf("starting command: %w", err)
	}

	go func() {
		for {
			reqd, err := reqz.Recv()
			if err != nil {
				if errors.Is(err, io.EOF) {
					return
				}
				slog.ErrorContext(ctx, "receiving signal request", "error", err)
				return
			}
			if !reqd.HasSignal() {
				continue
			}
			err = command.Process.Signal(syscall.Signal(reqd.GetSignal()))
			if err != nil && !errors.Is(err, os.ErrProcessDone) {
				slog.ErrorContext(ctx, "sending signal to command", "error", err)
			}
		}

	}()

	stat, err := command.Process.Wait()
	if err != nil {
		return errors.Errorf("waiting for command: %w", err)
		// if exitErr, ok := err.(*exec.ExitError); ok {
		// 	exitCode = exitErr.ExitCode()
		// } else {
		// 	return errors.Errorf("harpoon: running command [%s %s]: %w", argc, strings.Join(argv, " "), err)
		// }
	}

	exitCode := int32(stat.ExitCode())

	// resp, err = harpoonv1.NewValidatedRunResponse(func(b *harpoonv1.RunResponse_builder) {
	// 	b.ExitCode = ptr(int32(exitCode))
	// })
	resp, err := harpoonv1.NewRunSpecSignalResponseE(func(b *harpoonv1.RunSpecSignalResponse_builder) {
		b.ExitCode = ptr(int32(exitCode))
	})
	if err != nil {
		return errors.Errorf("building run response: %w", err)
	}

	slog.InfoContext(ctx, "command finished, responding to client", "err", err, "exitCode", exitCode)

	err = reqz.Send(resp)
	if err != nil {
		return errors.Errorf("sending run response: %w", err)
	}

	slog.InfoContext(ctx, "client responded to")

	return nil
}

func (s *GuestService) RunSpec(ctx context.Context, req *harpoonv1.RunSpecRequest) (resp *harpoonv1.RunSpecResponse, err error) {

	defer func() {
		if r := recover(); r != nil {
			slog.ErrorContext(ctx, "panic in Run", "error", r)
			resp = nil
			err = errors.Errorf("panic in Run: %v", r)
		}
		slog.InfoContext(ctx, "Run finished")
	}()

	argc := s.spec.Process.Args[0]
	argv := s.spec.Process.Args[1:]

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

	// resp, err = harpoonv1.NewValidatedRunResponse(func(b *harpoonv1.RunResponse_builder) {
	// 	b.ExitCode = ptr(int32(exitCode))
	// })
	resp = harpoonv1.NewRunSpecResponse(func(b *harpoonv1.RunSpecResponse_builder) {
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

	r, err := harpoonv1.NewTimeSyncResponseE(func(b *harpoonv1.TimeSyncResponse_builder) {
		b.PreviousTimeNs = &nowNano
	})
	if err != nil {
		return nil, errors.Errorf("building time sync response: %w", err)
	}
	return r, nil
}

// func (s *GuestService) Exec(ctx context.Context, server harpoonv1.TTRPCGuestService_ExecServer) (err error) {

// 	slog.InfoContext(ctx, "exec request received")

// 	defer func() {
// 		if r := recover(); r != nil {
// 			slog.ErrorContext(ctx, "exec request panicked", "error", r)
// 			fmt.Fprintln(logging.GetDefaultLogWriter(), string(debug.Stack()))
// 			err = errors.New("exec request panicked")
// 			return
// 		}
// 	}()

// 	req, err := server.Recv()
// 	if err != nil {
// 		return errors.Errorf("receiving request: %w", err)
// 	}

// 	start := req.GetStart()
// 	if start == nil {
// 		return errors.New("start request is required")
// 	}

// 	stdin := bytes.NewBuffer(nil)
// 	stdout := bytes.NewBuffer(nil)
// 	stderr := bytes.NewBuffer(nil)

// 	errch := make(chan error)
// 	stdinDone := make(chan struct{})
// 	stdoutDone := make(chan struct{})
// 	stderrDone := make(chan struct{})
// 	cmdDone := make(chan struct{})
// 	terminateDone := make(chan struct{})

// 	errs := []error{}
// 	go func() {
// 		defer func() {
// 			if r := recover(); r != nil {
// 				slog.ErrorContext(ctx, "err goroutine panicked", "error", r)
// 				panic(r)
// 			}
// 		}()
// 		for err := range errch {
// 			slog.ErrorContext(ctx, "err goroutine received error", "error", err)
// 			errs = append(errs, err)
// 			er, err := harpoonv1.NewExecResponse_WithErrorE(func(b *harpoonv1.ExecResponse_Error_builder) {
// 				b.Error = ptr(err.Error())
// 			})
// 			if err != nil {
// 				slog.ErrorContext(ctx, "building error response", "error", err)
// 				errs = append(errs, err)
// 				continue
// 			}

// 			err = server.Send(er)
// 			if err != nil {
// 				slog.ErrorContext(ctx, "sending error response", "error", err)
// 				errs = append(errs, err)
// 				continue
// 			}
// 		}
// 	}()

// 	go func() {
// 		defer func() {
// 			if r := recover(); r != nil {
// 				slog.ErrorContext(ctx, "stdin goroutine panicked", "error", r)
// 				panic(r)
// 			}
// 		}()
// 		defer close(stdinDone)
// 		for {
// 			req, err := server.Recv()
// 			if err != nil {
// 				if errors.Is(err, io.EOF) {
// 					slog.DebugContext(ctx, "stdin EOF")
// 					return
// 				}
// 				errch <- errors.Errorf("reading stdin from client: %w", err)
// 				return
// 			}

// 			slog.DebugContext(ctx, "stdin request received", "req", valuelog.NewPrettyValue(req))

// 			if req.GetTerminate() != nil {
// 				close(terminateDone)
// 				return
// 			}
// 			if req.GetStdin() == nil {
// 				errch <- errors.New("stdin request is required")
// 				return
// 			}
// 			_, err = stdin.Write(req.GetStdin().GetData())
// 			if err != nil {
// 				errch <- errors.Errorf("writing stdin to command: %w", err)
// 				return
// 			}
// 			if req.GetStdin().GetDone() {
// 				slog.DebugContext(ctx, "stdin request done")
// 				return
// 			}
// 		}
// 	}()

// 	// env := make(map[string]string)
// 	// for k, v := range start.GetEnvVars() {
// 	// 	env[k] = v
// 	// }

// 	// argv := start.GetArgv()
// 	// argc := start.GetArgc()
// 	// if start.GetUseEntrypoint() {
// 	// 	full := append(s.currentContainerEntrypoint, append([]string{argc}, argv...)...)
// 	// 	argv = full[1:]
// 	// 	argc = full[0]
// 	// }

// 	var spec *oci.Spec
// 	specd, err := os.ReadFile(filepath.Join(ec1init.Ec1AbsPath, ec1init.ContainerSpecFile))
// 	if err != nil {
// 		if !os.IsNotExist(err) {
// 			return errors.Errorf("reading spec: %w", err)
// 		}
// 	} else {
// 		err = json.Unmarshal(specd, &spec)
// 		if err != nil {
// 			return errors.Errorf("unmarshalling spec: %w", err)
// 		}
// 	}

// 	argc := spec.Process.Args[0]
// 	argv := spec.Process.Args[1:]

// 	cmd := exec.CommandContext(ctx, argc, argv...)
// 	cmd.Stdout = stdout
// 	cmd.Stderr = stderr
// 	cmd.Stdin = stdin
// 	cmd.Env = spec.Process.Env
// 	cmd.SysProcAttr = &syscall.SysProcAttr{
// 		// Cloneflags: syscall.CLONE_NEWNS,
// 	}
// 	cmd.Dir = spec.Process.Cwd

// 	// Start stdout streaming
// 	go streamOutput(
// 		ctx,
// 		stdout,
// 		server,
// 		harpoonv1.NewExecResponse_WithStdoutE,
// 		errch,
// 		stdoutDone,
// 		"stdout",
// 	)

// 	// Start stderr streaming
// 	go streamOutput(
// 		ctx,
// 		stderr,
// 		server,
// 		harpoonv1.NewExecResponse_WithStderrE,
// 		errch,
// 		stderrDone,
// 		"stderr",
// 	)

// 	go func() {
// 		defer func() {
// 			if r := recover(); r != nil {
// 				slog.ErrorContext(ctx, "cmd goroutine panicked", "error", r)
// 				panic(r)
// 			}
// 		}()
// 		defer close(cmdDone)
// 		err = cmd.Run()
// 		if err != nil {
// 			var exitErr *exec.ExitError
// 			if errors.As(err, &exitErr) {
// 				c := int32(exitErr.ExitCode())
// 				ec, err := harpoonv1.NewExecResponse_WithExitE(func(b *harpoonv1.ExecResponse_Exit_builder) {
// 					b.ExitCode = &c
// 				})
// 				if err != nil {
// 					errch <- errors.Errorf("building exit response: %w", err)
// 					return
// 				}

// 				err = server.Send(ec)
// 				if err != nil {
// 					errch <- errors.Errorf("sending exit response: %w", err)
// 					return
// 				}

// 			} else {
// 				errch <- errors.Errorf("running command - non-exit error: %w", err)
// 			}
// 			return
// 		}
// 	}()

// 	<-cmdDone
// 	<-stdoutDone
// 	<-stderrDone
// 	<-stdinDone
// 	<-terminateDone

// 	close(errch)

// 	allerrs := errors.Join(errs...)

// 	if len(errs) > 0 {
// 		slog.ErrorContext(ctx, "exec request finished", "err", allerrs)
// 	}

// 	slog.InfoContext(ctx, "exec request finished")

// 	return allerrs
// }

// // streamOutput handles reading from a source and streaming it to the client
// func streamOutput(
// 	ctx context.Context,
// 	reader io.Reader,
// 	server harpoonv1.TTRPCGuestService_ExecServer,
// 	responseBuilder func(func(*harpoonv1.Bytestream_builder)) (*harpoonv1.ExecResponse, error),
// 	errch chan<- error,
// 	done chan<- struct{},
// 	streamType string,
// ) {
// 	defer func() {
// 		if r := recover(); r != nil {
// 			slog.ErrorContext(ctx, "streamOutput goroutine panicked", "streamType", streamType, "error", r)
// 			panic(r)
// 		}
// 	}()
// 	defer close(done)
// 	for {
// 		buf := make([]byte, 1024)
// 		n, err := reader.Read(buf)
// 		if err != nil {
// 			if errors.Is(err, io.EOF) {
// 				return
// 			}
// 			errch <- errors.Errorf("reading %s from command: %w", streamType, err)
// 			return
// 		}

// 		resp, err := responseBuilder(func(b *harpoonv1.Bytestream_builder) {
// 			b.Data = buf[:n]
// 			b.Done = ptr(false)
// 		})
// 		if err != nil {
// 			errch <- errors.Errorf("building %s response: %w", streamType, err)
// 			return
// 		}

// 		err = server.Send(resp)
// 		if err != nil {
// 			if errors.Is(err, io.EOF) {
// 				return
// 			}
// 			errch <- errors.Errorf("sending %s to client: %w", streamType, err)
// 			return
// 		}
// 	}
// }
