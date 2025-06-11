package harpoon

import (
	"context"
	"io"
	"log/slog"
	"net"

	"github.com/mdlayher/vsock"
	"github.com/walteh/run"
	"gitlab.com/tozd/go/errors"
)

type GuestStdioForwarder interface {
	Stdin() io.Reader
	Stdout() io.Writer
	Stderr() io.Writer
}

type VsockStdioForwarderOpts struct {
	StdinPort  uint32
	StdoutPort uint32
	StderrPort uint32

	VsockContextID uint32
}

type VsockStdioForwarder struct {
	StdinReader  io.Reader
	StdoutWriter io.Writer
	StderrWriter io.Writer

	processes []*VsockStdioConnection
}

func (f *VsockStdioForwarder) Processes() []*VsockStdioConnection {
	return f.processes
}

func (f *VsockStdioForwarder) Stdin() io.Reader {
	return f.StdinReader
}

func (f *VsockStdioForwarder) Stdout() io.Writer {
	return f.StdoutWriter
}

func (f *VsockStdioForwarder) Stderr() io.Writer {
	return f.StderrWriter
}

func NewVsockStdioForwarder(ctx context.Context, opts VsockStdioForwarderOpts) (*VsockStdioForwarder, error) {

	if opts.VsockContextID == 0 {
		// default (constant?) for virtualization framework for guest to refer to host
		opts.VsockContextID = 3
	}

	stdinpr, stdinwr := io.Pipe()
	stdoutpr, stdoutwr := io.Pipe()
	stderrpr, stderrwr := io.Pipe()

	stdin, err := VsockStdioWriterConnection("stdin", opts.VsockContextID, opts.StdinPort, stdinwr)
	if err != nil {
		return nil, errors.Errorf("creating stdin connection: %w", err)
	}
	stdout, err := VsockStdioReaderConnection("stdout", opts.VsockContextID, opts.StdoutPort, stdoutpr)
	if err != nil {
		return nil, errors.Errorf("creating stdout connection: %w", err)
	}
	stderr, err := VsockStdioReaderConnection("stderr", opts.VsockContextID, opts.StderrPort, stderrpr)
	if err != nil {
		return nil, errors.Errorf("creating stderr connection: %w", err)
	}

	// errgroup := errgroup.Group{}

	// listener, err := vsock.ListenContextID(3, opts.StdinPort, nil)
	// if err != nil {
	// 	return nil, errors.Errorf("dialing vsock: %w", err)
	// }

	// errgroup.Go(func() error {
	// 	for {
	// 		slog.InfoContext(ctx, "waiting for stdin vsock connection")
	// 		conn, err := listener.Accept()
	// 		if err != nil {
	// 			if errors.Is(err, net.ErrClosed) {
	// 				return nil
	// 			}
	// 			slog.ErrorContext(ctx, "error accepting vsock", "error", err)
	// 			continue
	// 		}
	// 		defer conn.Close()
	// 		slog.InfoContext(ctx, "accepted stdin vsock connection")
	// 		go func() {
	// 			defer conn.Close()
	// 			_, err = io.Copy(stdinwr, conn)
	// 			if err != nil {
	// 				slog.ErrorContext(ctx, "error copying stdin to vsock", "error", err)
	// 			}
	// 		}()
	// 	}
	// })

	// listener, err = vsock.ListenContextID(3, opts.StdoutPort, nil)
	// if err != nil {
	// 	return nil, errors.Errorf("dialing vsock: %w", err)
	// }

	// errgroup.Go(func() error {
	// 	for {
	// 		slog.InfoContext(ctx, "waiting for stdout vsock connection")
	// 		conn, err := listener.Accept()
	// 		if err != nil {
	// 			if errors.Is(err, net.ErrClosed) {
	// 				return nil
	// 			}
	// 			slog.ErrorContext(ctx, "error accepting vsock", "error", err)
	// 			continue
	// 		}
	// 		slog.InfoContext(ctx, "accepted stdout vsock connection")
	// 		go func() {
	// 			defer conn.Close()
	// 			_, err = io.Copy(conn, stdoutpr)
	// 			if err != nil {
	// 				slog.ErrorContext(ctx, "error copying stdout to vsock", "error", err)
	// 			}
	// 		}()
	// 	}
	// })

	// listener, err = vsock.ListenContextID(3, opts.StderrPort, nil)
	// if err != nil {
	// 	return nil, errors.Errorf("dialing vsock: %w", err)
	// }

	// errgroup.Go(func() error {
	// 	for {
	// 		slog.InfoContext(ctx, "waiting for stderr vsock connection")
	// 		conn, err := listener.Accept()
	// 		if err != nil {
	// 			if errors.Is(err, net.ErrClosed) {
	// 				return nil
	// 			}
	// 			return errors.Errorf("error accepting vsock: %w", err)
	// 		}
	// 		slog.InfoContext(ctx, "accepted stderr vsock connection")
	// 		go func() {
	// 			slog.InfoContext(ctx, "copying stderr to vsock")
	// 			_, err = io.Copy(conn, stderrpr)
	// 			if err != nil {
	// 				slog.ErrorContext(ctx, "error copying stderr to vsock", "error", err)
	// 			}
	// 		}()
	// 	}
	// })

	// wait := make(chan error)

	// go func() {
	// 	defer func() {
	// 		if r := recover(); r != nil {
	// 			slog.ErrorContext(ctx, "panic in runStdioForwarding", "error", r)
	// 		}
	// 	}()
	// 	err := errgroup.Wait()
	// 	if err != nil {
	// 		slog.ErrorContext(ctx, "error in runStdioForwarding", "error", err)
	// 	}
	// }()

	return &VsockStdioForwarder{
		StdinReader:  stdinpr,
		StdoutWriter: stdoutwr,
		StderrWriter: stderrwr,
		processes:    []*VsockStdioConnection{stdin, stdout, stderr},
	}, nil
}

var _ run.Runnable = &VsockStdioConnection{}

type VsockStdioConnection struct {
	port   uint32
	writer io.Writer
	reader io.Reader
	vsock  *vsock.Listener
	name   string
	alive  bool
}

func VsockStdioReaderConnection(name string, ctxid uint32, port uint32, reader io.Reader) (*VsockStdioConnection, error) {
	listener, err := vsock.ListenContextID(ctxid, port, nil)
	if err != nil {
		return nil, errors.Errorf("dialing vsock: %w", err)
	}
	return &VsockStdioConnection{
		name:   name,
		port:   port,
		vsock:  listener,
		reader: reader,
	}, nil
}

func VsockStdioWriterConnection(name string, ctxid uint32, port uint32, writer io.Writer) (*VsockStdioConnection, error) {
	listener, err := vsock.ListenContextID(ctxid, port, nil)
	if err != nil {
		return nil, errors.Errorf("dialing vsock: %w", err)
	}
	return &VsockStdioConnection{
		name:   name,
		port:   port,
		vsock:  listener,
		writer: writer,
	}, nil
}

func (p *VsockStdioConnection) Run(ctx context.Context) error {
	p.alive = true
	defer func() {
		p.alive = false
	}()

	for {
		slog.InfoContext(ctx, "waiting for stdout vsock connection")
		conn, err := p.vsock.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return nil
			}
			slog.ErrorContext(ctx, "error accepting vsock", "error", err)
			continue
		}
		slog.InfoContext(ctx, "accepted stdout vsock connection")
		go func() {
			defer conn.Close()
			if p.writer != nil {
				_, err = io.Copy(p.writer, conn)
			} else {
				_, err = io.Copy(conn, p.reader)
			}
			if err != nil {
				slog.ErrorContext(ctx, "error copying stdout to vsock", "error", err)
			}
		}()
	}

}

func (p *VsockStdioConnection) Close(ctx context.Context) error {
	_ = p.vsock.Close()
	return nil
}

func (p *VsockStdioConnection) Alive() bool {
	return p.alive
}

func (p *VsockStdioConnection) Fields() []slog.Attr {
	return []slog.Attr{
		slog.String("pipe_name", p.name),
		slog.Uint64("pipe_port", uint64(p.port)),
	}
}

func (p *VsockStdioConnection) Name() string {
	return p.name
}
