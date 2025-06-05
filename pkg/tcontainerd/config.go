package tcontainerd

import (
	"context"
	"os"
	"path/filepath"

	"github.com/containerd/containerd/v2/client"
	"gitlab.com/tozd/go/errors"
)

func NewContainerdClient(ctx context.Context) (*client.Client, error) {
	return client.New(Address(), client.WithDefaultNamespace(Namespace()), client.WithTimeout(Timeout()))
}

func LoadCurrentServerConfig(ctx context.Context) ([]byte, error) {

	// read the lock file
	running, err := isServerRunning(ctx)
	if err != nil {
		return nil, errors.Errorf("checking if server is running: %w", err)
	}
	if !running {
		return nil, errors.Errorf("server is already running, please stop it first")
	}

	configFile := filepath.Join(WorkDir(), "containerd.toml")

	config, err := os.ReadFile(configFile)
	if err != nil {
		return nil, errors.Errorf("reading config file: %w", err)
	}

	return config, nil
}
