package vmm

import (
	_ "unsafe"

	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/prometheus/procfs"
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

func Exec[VM VirtualMachine](ctx context.Context, rvm *RunningVM[VM], command string) (string, string, string, error) {
	guestListenPort := uint32(2019)

	slog.DebugContext(ctx, "Exposing vsock port", "guestPort", guestListenPort)

	stdout, stderr, errd, err := rvm.Exec(ctx, command)
	if err != nil {
		return "", "", "", errors.Errorf("Failed to execute command: %w", err)
	}

	return string(stdout), string(stderr), string(errd), nil
}

//go:linkname parseMemInfo github.com/prometheus/procfs.parseMemInfo
func parseMemInfo(r io.Reader) (*procfs.Meminfo, error)

func ProcMemInfo[VM VirtualMachine](ctx context.Context, rvm *RunningVM[VM]) (*procfs.Meminfo, error) {
	stdout, stderr, errcode, err := rvm.Exec(ctx, "/bin/cat /proc/meminfo")
	if err != nil {
		return nil, errors.Errorf("Failed to execute command: %w", err)
	}

	if len(stderr) > 0 {
		return nil, errors.Errorf("Failed to execute command: %s", stderr)
	}

	if len(stdout) == 0 {
		return nil, errors.New("no output from command")
	}

	mi, err := parseMemInfo(bytes.NewReader(stdout))
	if err != nil {
		return nil, errors.Errorf("Failed to parse meminfo: %w: %s", err, errcode)
	}

	return mi, nil
}

// obviously this is not secure, we need something better long term
// for now its fine because im not even sure it will be used
// if this key thing is depended upon we need to move it to a more secure location
func AddSSHKeyToVM(ctx context.Context, workingDir string) error {
	sshKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return errors.Errorf("creating ssh key: %w", err)
	}

	m, err := x509.MarshalPKCS8PrivateKey(sshKey)
	if err != nil {
		return errors.Errorf("marshalling ssh key: %w", err)
	}

	sshKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: m})

	sshKeyFile := filepath.Join(workingDir, "id_ecdsa")
	err = os.WriteFile(sshKeyFile, sshKeyPEM, 0600)
	if err != nil {
		return errors.Errorf("writing ssh key: %w", err)
	}

	return nil
}
