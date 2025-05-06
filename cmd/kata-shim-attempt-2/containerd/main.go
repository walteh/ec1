package main

import (
	"fmt"
	"os"

	"github.com/containerd/containerd/v2/cmd/containerd/command"

	_ "github.com/containerd/containerd/v2/cmd/containerd/builtins"
)

func main() {
	app := command.App()
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "containerd: %s\n", err)
		os.Exit(1)
	}
}
