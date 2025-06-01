package main

import (
	"context"

	"github.com/containerd/containerd/v2/pkg/shim"

	"github.com/walteh/ec1/cmd/containerd-shim-harpoon-v1/containerd"
)

func withoutReaper(config *shim.Config) {
	config.NoReaper = true
}

func main() {
	shim.Run(context.Background(), containerd.NewManager("io.containerd.rund.v2"), withoutReaper)
}
