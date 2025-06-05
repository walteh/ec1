package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/containerd/containerd/v2/cmd/ctr/app"
	"github.com/urfave/cli/v2"

	"github.com/walteh/ec1/pkg/logging"
	"github.com/walteh/ec1/pkg/tcontainerd"
)

var pluginCmds = []*cli.Command{}

func main() {

	apd := app.New()

	args := []string{"ctr"}

	args = append(args, "-a", tcontainerd.Address())
	args = append(args, "-n", tcontainerd.Namespace())
	args = append(args, "--debug")

	args = append(args, os.Args[1:]...)

	ctx := context.Background()

	ctx = logging.SetupSlogSimpleToWriterWithProcessName(ctx, os.Stdout, true, "ctr")

	slog.InfoContext(ctx, "Starting ctr", "args", args)

	os.Args = args

	if err := apd.RunContext(ctx, args); err != nil {
		fmt.Fprintf(os.Stderr, "ctr: %s\n", err)
		os.Exit(1)
	}
}
