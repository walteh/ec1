package main

import (
	"context"

	"github.com/containerd/containerd/v2/pkg/shim"
	"github.com/walteh/ec1/cmd/kata-shim-attempt-2/shim/containerd"
)

func withoutReaper(config *shim.Config) {
	config.NoReaper = true
}

func main() {
	shim.Run(context.Background(), containerd.NewManager("io.containerd.kata.v2"), withoutReaper)
}
