package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/containerd/containerd/v2/cmd/containerd/command"

	"github.com/walteh/ec1/pkg/logging"
	// ... other necessary imports ...
)

var thisFileDir string

func init() {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		log.Fatal("unable to get the current filename")
	}
	thisFileDir = filepath.Dir(filepath.Dir(filename))
}

func main() {

	if len(os.Args) == 1 {
		os.Args = append(os.Args, "--config="+thisFileDir+"/config.toml", "-l=trace")
	}

	fmt.Println(os.Args)

	ctx := context.Background()
	// n := namespaces.Default
	// os.Args = append(os.Args, "-namespace="+n) // Usually containerd handles its own args parsing
	// ctx = namespaces.WithNamespace(ctx, n)
	ctx = logging.SetupSlogSimple(ctx) // Assuming this is for your containerd setup

	// This starts the containerd daemon
	containerdApp := command.App()
	if err := containerdApp.Run(os.Args); err != nil { // Pass actual args to containerd
		fmt.Fprintf(os.Stderr, "containerd error: %s\n", err)
		os.Exit(1)
	}
}
