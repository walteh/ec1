package machines

import (
	"context"

	"golang.org/x/crypto/ssh"
)

type OsProvider interface {
	URL() string
	Initialize(ctx context.Context, cacheDir string) error
	// ToVirtualMachine(ctx context.Context) (*config.VirtualMachine, error)
	SSHConfig() *ssh.ClientConfig
	ShutdownCommand() string
	Name() string
	Version() string
}
