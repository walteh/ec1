package main

import (
	"context"
	"io"
	"os"

	// Imports required by your NewManager and shim internals
	"github.com/containerd/containerd/v2/pkg/shim" // Needed if shim uses namespaces
	"github.com/containerd/log"
	"github.com/kata-containers/kata-containers/src/runtime/pkg/types"
	// ... other necessary imports for your shim logic (like NewManager) ...
)

func main() {
	// Basic context setup for the shim process
	ctx := context.Background()
	// Shims often need the namespace from containerd args,
	// shim.Run might handle context setup internally based on args/env.
	// You might extract namespace from os.Args if needed here.

	// Handle --version flag as shims usually do
	log_file := os.Getenv("SHIM_LOG_FILE")
	if log_file == "" {
		for _, env := range os.Environ() {
			log.L.WithField("env", env).Info("env")
		}
		log.L.WithField("error", "SHIM_LOG_FILE is not set").Error("Failed to open log file")
		os.Exit(1)
	}

	// Set up file logging
	f, err := os.OpenFile(log_file, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err == nil {
		log.L.Logger.SetOutput(io.MultiWriter(f))
	} else {
		log.L.WithField("error", err).Error("Failed to open log file")
	}

	// Then use log.L throughout your code
	log.L.WithField("args", os.Args).Info("Starting shim")

	defer func() {
		// log an exit message
		log.L.Info("Shim exited")

		// debug.PrintStack()
	}()
	// This runs the shim gRPC server, waiting for containerd to connect
	// Replace NewManager with your actual shim service constructor
	shim.Run(ctx, NewManager(types.DefaultKataRuntimeName))
}
