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

func newExecResponse(f func(*harpoonv1.ExecResponse_builder)) *harpoonv1.ExecResponse {
	builder := &harpoonv1.ExecResponse_builder{}
	f(builder)
	return builder.Build()
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

			resp := (&harpoonv1.ExecResponse_builder{
				Stdout: (&harpoonv1.Bytestream_builder{
					Data: buf[:n],
					Done: ptr(false),
				}).Build(),
			}).Build()
			err = server.Send(resp)
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
			b, err := buildAndValidate(func() *harpoonv1.Bytestream_builder {
				return &harpoonv1.Bytestream_builder{
					Data: buf[:n],
					Done: ptr(false),
				}
			})
			if err != nil {
				return nil, err
			}
			resd, err := buildAndValidate(func() (*harpoonv1.ExecResponse_builder, error) {

				return &harpoonv1.ExecResponse_builder{
					Stderr: b,
				}, nil
			})
			if err != nil {
				errch <- errors.Errorf("building stderr response: %w", err)
				return
			}
			err = server.Send(resd)
			// err = server.Send(newExecResponse(func(b *harpoonv1.ExecResponse_builder) {
			// 	b.Stderr = newBytestream(func(b *harpoonv1.Bytestream_builder) {
			// 		b.Data = buf[:n]
			// 		b.Done = ptr(false)
			// 	})
			// }))
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
				resd, err := buildAndValidate(func(b *harpoonv1.ExecResponse_builder) error {
					b.Exit, err = buildAndValidate(func(b *harpoonv1.ExecResponse_Exit_builder) error {
						b.ExitCode = &c
						return nil
					})
					return err
				})
				if err != nil {
					errch <- errors.Errorf("building exit response: %w", err)
					return
				}
				err = server.Send(resd)
				// err = server.Send(newExecResponse(func(b *harpoonv1.ExecResponse_builder) {
				// 	b.Exit = (&harpoonv1.ExecResponse_Exit_builder{
				// 		ExitCode: &c,
				// 	}).Build()
				// }))
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
