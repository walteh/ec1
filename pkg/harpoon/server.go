package harpoon

import (
	"bytes"
	"context"
	"io"
	"os/exec"

	"gitlab.com/tozd/go/errors"

	harpoonv1 "github.com/walteh/ec1/gen/proto/golang/harpoon/v1"
)

type AgentService struct{}

var (
	_ harpoonv1.TTRPCAgentServiceService = &AgentService{}
)

func NewAgentService() *AgentService {
	return &AgentService{}
}

func ptr[T any](v T) *T { return &v }

func (s *AgentService) Exec(ctx context.Context, server harpoonv1.TTRPCAgentService_ExecServer) error {
	req, err := server.Recv()
	if err != nil {
		return err
	}

	stdin := bytes.NewBuffer(req.GetStdin())
	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)

	errch := make(chan error)
	stdinDone := make(chan struct{})
	stdoutDone := make(chan struct{})
	stderrDone := make(chan struct{})
	cmdDone := make(chan struct{})

	go func() {
		defer close(stdinDone)
		for !req.GetStreamDone() {
			_, err := stdin.Write(stdin.Bytes())
			if err != nil {
				errch <- errors.Errorf("writing stdin to command: %w", err)
				return
			}
			req, err = server.Recv()
			if err != nil {
				if errors.Is(err, io.EOF) {
					return
				}
				errch <- errors.Errorf("reading stdin from client: %w", err)
				return
			}
		}
	}()

	cmd := exec.CommandContext(ctx, req.GetArgc(), req.GetArgv()...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Stdin = stdin

	go func() {
		defer close(stdoutDone)
		for {
			buf := make([]byte, 1024)
			n, err := stdout.Read(buf)
			if err != nil {
				if errors.Is(err, io.EOF) {
					return
				}
				errch <- errors.Errorf("reading stdout from command: %w", err)
				return
			}
			err = server.Send(&harpoonv1.ExecResponse{
				Stdout: buf[:n],
			})
			if err != nil {
				if errors.Is(err, io.EOF) {
					close(stdoutDone)
					return
				}
				errch <- errors.Errorf("sending stdout to client: %w", err)
				return
			}
		}
	}()

	go func() {
		defer close(stderrDone)
		for {
			buf := make([]byte, 1024)
			n, err := stderr.Read(buf)
			if err != nil {
				if errors.Is(err, io.EOF) {
					close(stderrDone)
					return
				}
				errch <- errors.Errorf("sending stderr to client: %w", err)
				return
			}
			err = server.Send(&harpoonv1.ExecResponse{
				Stderr: buf[:n],
			})
			if err != nil {
				if errors.Is(err, io.EOF) {
					return
				}
				errch <- errors.Errorf("sending stderr to client: %w", err)
				return
			}
		}
	}()

	go func() {
		defer close(cmdDone)
		err = cmd.Run()
		if err != nil {
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) {
				c := int32(exitErr.ExitCode())
				err = server.Send(&harpoonv1.ExecResponse{
					ExitCode: &c,
				})
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
			err = server.Send(&harpoonv1.ExecResponse{
				Error: ptr(err.Error()),
			})
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

	err = server.Send(&harpoonv1.ExecResponse{
		StreamDone: ptr(true),
	})
	if err != nil {
		return err
	}

	return allerrs
}
