package containerd

import (
	"encoding/json"
	"net"
	"os"
	"path"
	"path/filepath"

	"github.com/containerd/containerd/api/types"
	"github.com/containerd/containerd/v2/core/mount"
	"github.com/containerd/log"
	"github.com/opencontainers/runtime-spec/specs-go"
	"gitlab.com/tozd/go/errors"
)

func shortenPath(p string) (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", errors.Errorf("getting working directory: %w", err)
	}

	shortened, err := filepath.Rel(wd, path.Join(p))
	if err != nil || len(shortened) > len(p) {
		return p, nil
	}

	return shortened, nil
}

func processMounts(targetRoot string, rootfs []*types.Mount, specMounts []specs.Mount) ([]mount.Mount, error) {
	var mounts []mount.Mount
	for _, m := range rootfs {
		mm, err := processMount(targetRoot, m.Type, m.Source, m.Target, m.Options)
		if err != nil {
			return nil, errors.Errorf("processing mount: %w", err)
		}

		if mm != nil {
			mounts = append(mounts, *mm)
		}
	}

	for _, m := range specMounts {
		mm, err := processMount(targetRoot, m.Type, m.Source, m.Destination, m.Options)
		if err != nil {
			return nil, errors.Errorf("processing mount: %w", err)
		}

		if mm != nil {
			mounts = append(mounts, *mm)
		}
	}

	return mounts, nil
}

func processMount(rootfs, mtype, source, target string, options []string) (*mount.Mount, error) {
	m := &mount.Mount{
		Type:    mtype,
		Source:  source,
		Target:  target,
		Options: options,
	}

	switch mtype {
	case "bind":
		stat, err := os.Stat(source)
		if err != nil {
			return nil, errors.Errorf("statting source '%s': %w", source, err)
		}

		if stat.IsDir() {
			fullPath := filepath.Join(rootfs, target)
			if err = os.MkdirAll(fullPath, 0o755); err != nil {
				return nil, errors.Errorf("creating directory '%s' to mount '%s': %w", fullPath, source, err)
			}

			return m, nil
		} else {
			// skip, only dirs are supported by bindfs
		}
	case "devfs":
		return m, nil
	}

	mountJson, err := json.Marshal(m)
	if err != nil {
		return nil, errors.Errorf("marshalling mount: %w", err)
	}

	log.L.Warn("skipping mount: ", string(mountJson))
	return nil, nil
}

func unixSocketCopy(from, to *net.UnixConn) error {
	for {
		// TODO: How we determine buffer size that is guaranteed to be enough?
		b := make([]byte, 1024)
		oob := make([]byte, 1024)
		n, oobn, _, addr, err := from.ReadMsgUnix(b, oob)
		if err != nil {
			return err
		}
		_, _, err = to.WriteMsgUnix(b[:n], oob[:oobn], addr)
		if err != nil {
			return err
		}
	}
}
