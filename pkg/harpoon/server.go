package harpoon

import (
	"bytes"
	"context"
	"io"
	"os/exec"

	"gitlab.com/tozd/go/errors"

	harpoonv1 "github.com/walteh/ec1/gen/proto/golang/harpoon/v1"
)

type AgentService struct {
	currentContainerEntrypoint []string
}

var (
	_ harpoonv1.TTRPCGuestServiceService = &AgentService{}
)

func NewAgentService() *AgentService {
	return &AgentService{}
}

// streamOutput handles reading from a source and streaming it to the client
func streamOutput(
	reader io.Reader,
	server harpoonv1.TTRPCGuestService_ExecServer,
	responseBuilder func(func(*harpoonv1.Bytestream_builder)) (*harpoonv1.ExecResponse, error),
	errch chan<- error,
	done chan<- struct{},
	streamType string,
) {
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

func (s *AgentService) Exec(ctx context.Context, server harpoonv1.TTRPCGuestService_ExecServer) error {
	req, err := server.Recv()
	if err != nil {
		return err
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

	go func() {
		defer close(stdinDone)
		for {
			req, err := server.Recv()
			if err != nil {
				if errors.Is(err, io.EOF) {
					return
				}
				errch <- errors.Errorf("reading stdin from client: %w", err)
				return
			}
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
				return
			}
		}
	}()

	env := make(map[string]string)
	for k, v := range start.GetEnvVars() {
		env[k] = v
	}

	argv := start.GetArgv()
	argc := start.GetArgc()
	if start.GetUseEntrypoint() {
		full := append(s.currentContainerEntrypoint, append([]string{argc}, argv...)...)
		argv = full[1:]
		argc = full[0]
	}

	cmd := exec.CommandContext(ctx, argc, argv...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Stdin = stdin

	// Start stdout streaming
	go streamOutput(
		stdout,
		server,
		harpoonv1.NewValidatedExecResponse_WithStdout,
		errch,
		stdoutDone,
		"stdout",
	)

	// Start stderr streaming
	go streamOutput(
		stderr,
		server,
		harpoonv1.NewValidatedExecResponse_WithStderr,
		errch,
		stderrDone,
		"stderr",
	)

	go func() {
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

	errs := []error{}
	go func() {
		for err := range errch {
			errs = append(errs, err)
			er, err := harpoonv1.NewValidatedExecResponse_WithError(func(b *harpoonv1.ExecResponse_Error_builder) {
				b.Error = ptr(err.Error())
			})
			if err != nil {
				errs = append(errs, err)
				return
			}
			err = server.Send(er)
			if err != nil {
				return
			}
		}
	}()

	<-cmdDone
	<-stdoutDone
	<-stderrDone
	<-stdinDone

	close(errch)

	allerrs := errors.Join(errs...)

	return allerrs
}
