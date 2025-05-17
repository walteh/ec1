package vmm

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"

	"gitlab.com/tozd/go/errors"
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
