package agent

import (
	"context"
	"os"
	"path/filepath"

	"gitlab.com/tozd/go/errors"

	"github.com/walteh/ec1/sanbox/pkg/cloud/id"
)

type IDStore interface {
	GetInstanceID(ctx context.Context) (id.ID, bool, error)
	SetInstanceID(ctx context.Context, id id.ID) error
}

type FSIDStore struct {
	dir string
}

func cacheDirFile(file string) (string, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cacheDir, "ec1", file), nil
}

func (s *FSIDStore) GetInstanceID(ctx context.Context) (id.ID, bool, error) {
	cacheDir, err := cacheDirFile("agent_id")
	if err != nil {
		return id.ID{}, false, errors.Errorf("getting cache directory: %w", err)
	}

	instanceID, err := os.ReadFile(cacheDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return id.ID{}, false, nil
		}
		return id.ID{}, false, errors.Errorf("reading agent ID: %w", err)
	}

	p, err := id.ParseID(string(instanceID))
	if err != nil {
		return id.ID{}, false, errors.Errorf("parsing agent ID: %w", err)
	}

	return p, true, nil
}

func (s *FSIDStore) SetInstanceID(ctx context.Context, id id.ID) error {
	cacheDir, err := cacheDirFile("agent_id")
	if err != nil {
		return errors.Errorf("getting cache directory: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(cacheDir), 0755); err != nil {
		return errors.Errorf("creating cache directory: %w", err)
	}

	if err := os.WriteFile(cacheDir, []byte(id.String()), 0644); err != nil {
		return errors.Errorf("writing agent ID: %w", err)
	}

	return nil
}
