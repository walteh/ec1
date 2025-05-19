package vmm

import (
	_ "unsafe"

	"context"
	"io"
	"log/slog"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/prometheus/procfs"
	"gitlab.com/tozd/go/errors"

	"github.com/walteh/ec1/pkg/streamexec"
	"github.com/walteh/ec1/pkg/streamexec/protocol"
	"github.com/walteh/ec1/pkg/streamexec/transport"
)

func ObtainSSHConnectionWithGuest(ctx context.Context, address string, cfg *ssh.ClientConfig, timeout <-chan time.Time) (*ssh.Client, error) {
	var (
		sshClient *ssh.Client
		err       error
	)

	var scheme string
	if strings.Contains(address, "://") {
		split := strings.Split(address, "://")
		scheme = split[0]
		address = split[1]
	} else {
		scheme = "tcp"
	}

	// tenSeconds := time.After(30 * time.Second)

	for {
		select {
		case <-ctx.Done():
			return nil, errors.Errorf("error observed before establishing SSH connection: %w", ctx.Err())
		case <-time.After(1 * time.Second):
			slog.DebugContext(ctx, "trying ssh dial", "address", address, "scheme", scheme)
			sshClient, err = ssh.Dial(scheme, address, cfg)
			if err == nil {
				slog.InfoContext(ctx, "established SSH connection", "address", address, "scheme", scheme)
				return sshClient, nil
			}
			slog.DebugContext(ctx, "ssh failed", "error", err)
		case <-timeout:
			return nil, errors.New("timeout waiting for SSH")
		}
	}
}

func Exec(ctx context.Context, vm VirtualMachine, command string) (string, string, string, error) {
	guestListenPort := uint32(2019)

	slog.DebugContext(ctx, "Exposing vsock port", "guestPort", guestListenPort)

	conn, err := vm.VSockConnect(ctx, guestListenPort)
	if err != nil {
		return "", "", "", errors.Errorf("Failed to expose vsock port: %w", err)
	}

	trans := transport.NewFunctionTransport(func() (io.ReadWriteCloser, error) { return conn, nil }, nil)

	scli := streamexec.NewClient(trans, func(conn io.ReadWriter) protocol.Protocol {
		return protocol.NewFramedProtocol(conn)
	})

	err = scli.Connect(ctx)
	if err != nil {
		return "", "", "", errors.Errorf("Failed to connect to streamexec server: %w", err)
	}

	defer scli.Close()

	stdout, stderr, errd, err := scli.ExecuteCommand(ctx, "cat /proc/meminfo")
	if err != nil {
		return "", "", "", errors.Errorf("Failed to execute command: %w", err)
	}

	return string(stdout), string(stderr), string(errd), nil
}

//go:linkname parseMemInfo github.com/prometheus/procfs.parseMemInfo
func parseMemInfo(r io.Reader) (*procfs.Meminfo, error)

func ProcMemInfo(ctx context.Context, vm VirtualMachine) (*procfs.Meminfo, error) {
	stdout, stderr, _, err := Exec(ctx, vm, "cat /proc/meminfo")
	if err != nil {
		return nil, errors.Errorf("Failed to execute command: %w", err)
	}

	if len(stderr) > 0 {
		return nil, errors.Errorf("Failed to execute command: %s", stderr)
	}

	mi, err := parseMemInfo(strings.NewReader(stdout))
	if err != nil {
		return nil, errors.Errorf("Failed to parse meminfo: %w", err)
	}

	return mi, nil
}
