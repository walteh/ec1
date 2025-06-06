package vf

import (
	"context"
	"crypto/sha256"
	"fmt"
	"path/filepath"

	"github.com/Code-Hex/vz/v3"
	"gitlab.com/tozd/go/errors"

	"github.com/walteh/ec1/pkg/host"
)

const snapshotExtension = ".ec1-vf-snapshot"

func (v *VirtualMachine) SaveFullSnapshot(ctx context.Context, path string) error {

	hash := sha256.Sum256([]byte(path))

	if ok, err := v.configuration.ValidateSaveRestoreSupport(); err != nil {
		return errors.Errorf("checking save/restore support: %w", err)
	} else if !ok {
		return errors.New("save/restore is not supported")
	}

	cacheDir, err := host.EmphiricalVMCacheDir(ctx, v.ID())
	if err != nil {
		return errors.Errorf("getting cache directory: %w", err)
	}

	snapshotPath := filepath.Join(cacheDir, "snapshots", fmt.Sprintf("%x%s", hash, snapshotExtension))

	if err := v.vzvm.SaveMachineStateToPath(snapshotPath); err != nil {
		return errors.Errorf("saving snapshot: %w", err)
	}

	return nil
}

func (v *VirtualMachine) RestoreFromFullSnapshot(ctx context.Context, path string) error {

	if ok, err := v.configuration.ValidateSaveRestoreSupport(); err != nil {
		return errors.Errorf("checking save/restore support: %w", err)
	} else if !ok {
		return errors.New("save/restore is not supported")
	}

	hash := sha256.Sum256([]byte(path))

	cacheDir, err := host.EmphiricalVMCacheDir(ctx, v.ID())
	if err != nil {
		return errors.Errorf("getting cache directory: %w", err)
	}

	snapshotPath := filepath.Join(cacheDir, "snapshots", fmt.Sprintf("%x%s", hash, snapshotExtension))

	if v.vzvm.State() != vz.VirtualMachineStateStopped {
		return errors.New("cannot restore from snapshot while VM is running")
	}

	if err := v.vzvm.RestoreMachineStateFromURL(snapshotPath); err != nil {
		return errors.Errorf("restoring from snapshot: %w", err)
	}

	return nil
}
