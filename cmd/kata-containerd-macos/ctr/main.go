package main

import (
	"fmt"
	"os"

	"github.com/containerd/containerd/v2/cmd/ctr/app"
	"github.com/urfave/cli/v2"
)

var pluginCmds = []*cli.Command{}

func main() {
	app := app.New()
	app.Commands = append(app.Commands, pluginCmds...)
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "ctr: %s\n", err)
		os.Exit(1)
	}
}
