package tcontainerd

import (
	"context"
	"io"
	"log/slog"
	"net"
	"os"
	"path/filepath"
)

func (s *DevContainerdServer) setupReexec(ctx context.Context) error {

	os.MkdirAll(filepath.Dir(ShimSimlinkPath()), 0755)
	os.MkdirAll(filepath.Dir(CtrSimlinkPath()), 0755)

	proxySock, err := net.Listen("unix", ShimLogProxySockPath())
	if err != nil {
		slog.Error("Failed to create log proxy socket", "error", err, "path", ShimLogProxySockPath())
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

	// Set up logging for TestMain

	self, _ := os.Executable()

	if err := os.Symlink(self, ShimSimlinkPath()); err != nil {
		slog.Error("create shim link", "error", err)
		os.Exit(1)
	}

	if err := os.Symlink(self, CtrSimlinkPath()); err != nil {
		slog.Error("create ctr link", "error", err)
		os.Exit(1)
	}

	oldPath := os.Getenv("PATH")
	newPath := filepath.Dir(ShimSimlinkPath()) + string(os.PathListSeparator) + oldPath
	newPath = filepath.Dir(CtrSimlinkPath()) + string(os.PathListSeparator) + newPath
	os.Setenv("PATH", newPath)

	return nil
}
