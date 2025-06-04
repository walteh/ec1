package main

import (
	"context"

	"github.com/containerd/containerd/v2/pkg/shim"

	"github.com/walteh/ec1/cmd/containerd-shim-harpoon-v2/containerd"
)

const (
	ContainerdShimRuntimeID = "io.containerd.harpoon.v2"
	ContainerdShimName      = "containerd-shim-harpoon-v2"
)

func main() {
	shim.Run(context.Background(), containerd.NewManager(ContainerdShimRuntimeID), func(config *shim.Config) {
		config.NoReaper = true
	})
}
