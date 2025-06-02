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
	// log.Printf("[shim main] pid=%d argv=%q", os.Getpid(), os.Args)
	shim.Run(context.Background(), containerd.NewManager("io.containerd.harpoon.v1"), withoutReaper)
}
