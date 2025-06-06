package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/containerd/containerd/v2/pkg/shim"

	"github.com/walteh/ec1/cmd/containerd-shim-harpoon-v2/containerd"
)

const (
	ContainerdShimRuntimeID = "io.containerd.harpoon.v2"
	ContainerdShimName      = "containerd-shim-harpoon-v2"
)

func main() {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("panic", "error", r)
			panic(r)
		}
	}()

	go func() {
		syschan := make(chan os.Signal, 1)
		signal.Notify(syschan, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
		<-syschan
		slog.Info("received signal, shutting down")
	}()
	shim.Run(context.Background(), containerd.NewManager(ContainerdShimRuntimeID), func(config *shim.Config) {
		config.NoReaper = true
	})

}
