package main

import (
	"context"
	"fmt"
	"os"
	_ "unsafe"

	"github.com/containerd/containerd/v2/cmd/containerd/command"
	"github.com/containerd/containerd/v2/pkg/namespaces"
	"github.com/containerd/containerd/v2/pkg/shim"
	"github.com/containerd/fifo"

	"github.com/walteh/ec1/pkg/logging"

	"github.com/kata-containers/kata-containers/src/runtime/pkg/katautils"
	"github.com/kata-containers/kata-containers/src/runtime/pkg/types"
)

// func shimConfig(config *runtime.ShimConfig) {
// 	config.NoReaper = true
// 	config.NoSubreaper = true
// }

var s = fifo.OpenFifoDup2

func main() {
	ctx := context.Background()

	n := namespaces.Default

	os.Args = append(os.Args, "-namespace="+n)

	ctx = namespaces.WithNamespace(ctx, n)

	ctx = logging.SetupSlogSimple(ctx)

	go func() {
		if len(os.Args) == 2 && os.Args[1] == "--version" {
			fmt.Printf("%s containerd shim (Golang): id: %q, version: %s, commit: %v\n", katautils.PROJECT, types.DefaultKataRuntimeName, katautils.VERSION, katautils.COMMIT)
			os.Exit(0)
		}
		shim.Run(ctx, NewManager(types.DefaultKataRuntimeName))
	}()

	containerdApp := command.App()

	if err := containerdApp.Run([]string{}); err != nil {
		fmt.Fprintf(os.Stderr, "containerd: %s\n", err)
		os.Exit(1)
	}

	// cdshim.Run(ctx, types.DefaultKataRuntimeName, shim.New, shimConfig)
}
