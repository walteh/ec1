package tapsock

import (
	"context"
	"log/slog"
	"net"
	"net/url"
	"os"

	"golang.org/x/sync/errgroup"

	"github.com/containers/gvisor-tap-vsock/pkg/tap"
	"github.com/containers/gvisor-tap-vsock/pkg/transport"
	"github.com/containers/gvisor-tap-vsock/pkg/types"
	"gitlab.com/tozd/go/errors"
)

// this package should be creating the remote address too.
// im pretty sure we just need to dial instead of listen here and it will work.
// then pass the file descriptor to the vm and boom
// gvproxy listens on the socket,

type VMSocket interface {
	Protocol() types.Protocol
	Validate() error
	Listen(ctx context.Context, g *errgroup.Group, vn *tap.Switch) (net.Addr, error)
	URL() string
}

func NewVFKitVMSocket(path string) *VFKitSocket {
	return &VFKitSocket{Path: path}
}

func NewQEMUVMSocket(path string) *QEMUSocket {
	return &QEMUSocket{Path: path}
}

func NewBessVMSocket(path string) *BessSocket {
	return &BessSocket{Path: path}
}

type VFKitSocket struct {
	Path string
}

func (s *VFKitSocket) URL() string {
	return s.Path
}

func (s *VFKitSocket) Protocol() types.Protocol {
	return types.VfkitProtocol
}

func (s *VFKitSocket) Validate() error {
	uri, err := url.Parse(s.Path)
	if err != nil || uri == nil {
		return errors.Errorf("invalid value for listen-vfkit %w", err)
	}
	if uri.Scheme != "unixgram" {
		return errors.New("listen-vfkit must be unixgram:// address")
	}

	if _, err := os.Stat(uri.Path); err == nil {
		return errors.Errorf("%q already exists", uri.Path)
	}
	return nil
}

func (s *VFKitSocket) Listen(ctx context.Context, g *errgroup.Group, vn *tap.Switch) (net.Addr, error) {
	conn, err := transport.ListenUnixgram(s.Path)
	if err != nil {
		return nil, errors.Errorf("vfkit listen error %w", err)
	}

	g.Go(func() error {
		<-ctx.Done()
		if err := conn.Close(); err != nil {
			slog.WarnContext(ctx, "error closing vfkit socket", "socket", s.Path, "error", err)
		}
		vfkitSocketURI, _ := url.Parse(s.Path)
		return os.Remove(vfkitSocketURI.Path)
	})

	g.Go(func() error {
		vfkitConn, err := transport.AcceptVfkit(conn)
		if err != nil {
			return errors.Errorf("vfkit accept error %w", err)
		}
		return vn.Accept(ctx, vfkitConn, types.VfkitProtocol)
	})

	return conn.LocalAddr(), nil
}

type QEMUSocket struct {
	Path string
}

func (s *QEMUSocket) URL() string {
	return s.Path
}

func (s *QEMUSocket) Protocol() types.Protocol {
	return types.QemuProtocol
}

func (s *QEMUSocket) Validate() error {
	uri, err := url.Parse(s.Path)
	if err != nil || uri == nil {
		return errors.Errorf("invalid value for listen-qemu %w", err)
	}
	if _, err := os.Stat(uri.Path); err == nil && uri.Scheme == "unix" {
		return errors.Errorf("%q already exists", uri.Path)
	}
	return nil
}

func (s *QEMUSocket) Listen(ctx context.Context, g *errgroup.Group, vn *tap.Switch) (net.Addr, error) {
	return commonListen(ctx, g, "qemu", s.Path, vn, types.QemuProtocol)
}

type BessSocket struct {
	Path string
}

func (s *BessSocket) URL() string {
	return s.Path
}

func (s *BessSocket) Protocol() types.Protocol {
	return types.BessProtocol
}

func (s *BessSocket) Validate() error {
	uri, err := url.Parse(s.Path)
	if err != nil || uri == nil {
		return errors.Errorf("invalid value for listen-bess %w", err)
	}
	if uri.Scheme != "unixpacket" {
		return errors.New("listen-bess must be unixpacket:// address")
	}
	if _, err := os.Stat(uri.Path); err == nil {
		return errors.Errorf("%q already exists", uri.Path)
	}
	return nil
}

func (s *BessSocket) Listen(ctx context.Context, g *errgroup.Group, vn *tap.Switch) (net.Addr, error) {
	return commonListen(ctx, g, "bess", s.Path, vn, types.BessProtocol)
}

func commonListen(ctx context.Context, g *errgroup.Group, name, path string, swtch *tap.Switch, protocol types.Protocol) (net.Addr, error) {
	listener, err := net.ListenUnix("unixpacket", &net.UnixAddr{Name: path, Net: "unixpacket"})
	if err != nil {
		return nil, errors.Wrap(err, "listen error")
	}

	g.Go(func() error {
		<-ctx.Done()
		if err := listener.Close(); err != nil {
			slog.ErrorContext(ctx, "error closing "+name, "socket", path, "error", err)
		}
		return os.Remove(path)
	})

	g.Go(func() error {
		conn, err := listener.Accept()
		if err != nil {
			return errors.Wrap(err, "accept error")
		}
		return swtch.Accept(ctx, conn, protocol)
	})

	return listener.Addr(), nil
}
