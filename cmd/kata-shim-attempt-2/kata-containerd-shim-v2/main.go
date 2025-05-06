package main

import (
	"context"

	"github.com/containerd/containerd/v2/pkg/shim"
	"github.com/walteh/ec1/cmd/containerd-kata/kata-containerd-shim-v2/containerd"
)

func withoutReaper(config *shim.Config) {
	config.NoReaper = true
}

func main() {
	shim.Run(context.Background(), containerd.NewManager("io.containerd.kata.v2"), withoutReaper)
}
