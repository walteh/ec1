package harpoon

import (
	"context"
	"log/slog"

	"github.com/containerd/ttrpc"
	"github.com/mdlayher/vsock"
	"gitlab.com/tozd/go/errors"

	"github.com/walteh/run"

	harpoonv1 "github.com/walteh/ec1/gen/proto/golang/harpoon/v1"
)

type GuestServiceRunnerOpts struct {
	VsockContextID uint32
	VsockPort      uint32
	GuestService   *GuestService
}

type GuestServiceRunner struct {
	ttrpcServer *ttrpc.Server
	vsock       *vsock.Listener
	alive       bool
}

var _ run.Runnable = &GuestServiceRunner{}

// forwarder, err := NewVsockStdioForwarder(ctx, VsockStdioForwarderOpts{
// 	StdinPort:  uint32(ec1init.VsockStdinPort),
// 	StdoutPort: uint32(ec1init.VsockStdoutPort),
// 	StderrPort: uint32(ec1init.VsockStderrPort),
// 	VsockContextID: 3,
// })
// if err != nil {
// 	return nil, errors.Errorf("creating vsock stdio forwarder: %w", err)
// }

func NewGuestServiceRunner(ctx context.Context, opts GuestServiceRunnerOpts) (*GuestServiceRunner, error) {

	ttrpcServe, err := ttrpc.NewServer(ttrpc.WithServerDebugging())
	if err != nil {
		return nil, errors.Errorf("creating ttrpc server: %w", err)
	}

	harpoonv1.RegisterTTRPCGuestServiceService(ttrpcServe, opts.GuestService.WrapWithErrorLogging())

	listener, err := vsock.ListenContextID(opts.VsockContextID, opts.VsockPort, nil)
	if err != nil {
		return nil, errors.Errorf("dialing vsock: %w", err)
	}

	return &GuestServiceRunner{
		ttrpcServer: ttrpcServe,
		vsock:       listener,
	}, nil
}

// func runTtrpc(ctx context.Context) error {

// 	if err != nil {
// 		slog.ErrorContext(ctx, "problem running stdio forwarding", "error", err)
// 		return errors.Errorf("running stdio forwarding: %w", err)
// 	}

// 	listener, err := vsock.ListenContextID(3, uint32(ec1init.VsockPort), nil)
// 	if err != nil {
// 		return errors.Errorf("dialing vsock: %w", err)
// 	}

// 	return ttrpcServe.Serve(ctx, listener)
// }

func (p *GuestServiceRunner) Run(ctx context.Context) error {
	p.alive = true
	defer func() {
		p.alive = false
	}()

	err := p.ttrpcServer.Serve(ctx, p.vsock)
	if err != nil {
		return errors.Errorf("serving ttrpc: %w", err)
	}
	return nil
}

func (p *GuestServiceRunner) Close(ctx context.Context) error {
	_ = p.vsock.Close()
	_ = p.ttrpcServer.Close()
	return nil
}

func (p *GuestServiceRunner) Alive() bool {
	return p.alive
}

func (p *GuestServiceRunner) Fields() []slog.Attr {
	return []slog.Attr{}
}

func (p *GuestServiceRunner) Name() string {
	return "guest-service-runner"
}
