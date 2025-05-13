package gvnet

import (
	"context"
	"log/slog"

	"github.com/containers/gvisor-tap-vsock/pkg/tcpproxy"
	"github.com/walteh/run"
	"gitlab.com/tozd/go/errors"
)

var _ run.Runnable = &TCPProxyRunner{}

type TCPProxyRunner struct {
	source string
	target string
	proxy  *tcpproxy.Proxy
	alive  bool
}

func NewTCPProxyRunner(source, target string, proxy *tcpproxy.Proxy) (*TCPProxyRunner, error) {
	err := proxy.Start()
	if err != nil {
		return nil, errors.Errorf("running tcpproxy [%s -> %s]: %w", source, target, err)
	}
	return &TCPProxyRunner{
		source: source,
		target: target,
		proxy:  proxy,
		alive:  false,
	}, nil
}

func (p *TCPProxyRunner) Run(ctx context.Context) error {
	p.alive = true
	defer func() {
		p.alive = false
	}()
	err := p.proxy.Wait()
	if err != nil {
		return errors.Errorf("running tcpproxy [%s -> %s]: %w", p.source, p.target, err)
	}
	return nil
}

func (p *TCPProxyRunner) Close(ctx context.Context) error {
	_ = p.proxy.Close()
	return nil
}

func (p *TCPProxyRunner) Alive() bool {
	return p.alive
}

func (p *TCPProxyRunner) Fields() []slog.Attr {
	return []slog.Attr{
		slog.String("tcpproxy_source", p.source),
		slog.String("tcpproxy_target", p.target),
	}
}

func (p *TCPProxyRunner) Name() string {
	return "tcpproxy"
}
