package tcontainerd

import (
	"context"
	"io"
	"log/slog"
	"net"
	"os"
	"path/filepath"
)

func SetupReexec(ctx context.Context, justSymlinks bool) error {

	os.MkdirAll(filepath.Dir(ShimSimlinkPath()), 0755)
	os.MkdirAll(filepath.Dir(CtrSimlinkPath()), 0755)

	self, _ := os.Executable()
	os.Remove(ShimSimlinkPath())
	os.Remove(CtrSimlinkPath())

	if err := os.Symlink(self, ShimSimlinkPath()); err != nil {
		slog.Error("create shim link", "error", err)
		os.Exit(1)
	}

	if err := os.Symlink(self, CtrSimlinkPath()); err != nil {
		slog.Error("create ctr link", "error", err)
		os.Exit(1)
	}

	// drop the permissions on the proxy socket
	if err := os.Chown(CtrSimlinkPath(), 1000, 1000); err != nil {
		slog.Error("Failed to drop permissions on log proxy socket", "error", err, "path", ShimLogProxySockPath())
		os.Exit(1)
	}

	if err := os.Chown(ShimSimlinkPath(), 1000, 1000); err != nil {
		slog.Error("Failed to drop permissions on log proxy socket", "error", err, "path", ShimLogProxySockPath())
		os.Exit(1)
	}

	// drop the privileges on the proxy socket

	// drop the privileges on myself now
	if err := os.Chown(self, 1000, 1000); err != nil {
		slog.Error("Failed to drop permissions on myself", "error", err, "path", self)
		os.Exit(1)
	}

	if justSymlinks {
		return nil
	}

	proxySock, err := net.Listen("unix", ShimLogProxySockPath())
	if err != nil {
		slog.Error("Failed to create log proxy socket", "error", err, "path", ShimLogProxySockPath())
		os.Exit(1)
	}

	if err := os.Chown(ShimLogProxySockPath(), 1000, 1000); err != nil {
		slog.Error("Failed to drop privileges on log proxy socket", "error", err, "path", ShimLogProxySockPath())
		os.Exit(1)
	}

	// fwd logs from the proxy socket to stdout
	go func() {
		defer proxySock.Close()
		for {
			conn, err := proxySock.Accept()
			if err != nil {
				slog.Error("Failed to accept log proxy connection", "error", err)
				return
			}
			go func() { _, _ = io.Copy(os.Stdout, conn) }()
		}
	}()

	oldPath := os.Getenv("PATH")
	newPath := filepath.Dir(ShimSimlinkPath()) + string(os.PathListSeparator) + oldPath
	os.Setenv("PATH", newPath)

	return nil
}
