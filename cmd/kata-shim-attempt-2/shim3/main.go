package main

// Copyright (c) 2018 HyperHQ Inc.
//
// SPDX-License-Identifier: Apache-2.0
//

import (
	"context"
	"fmt"
	"io"
	"os"

	shimapi "github.com/containerd/containerd/v2/pkg/shim"
	"github.com/containerd/log"

	manager "github.com/kata-containers/kata-containers/src/runtime/pkg/containerd-shim-v2/manager"
	"github.com/kata-containers/kata-containers/src/runtime/pkg/katautils"
	"github.com/kata-containers/kata-containers/src/runtime/pkg/types"

	_ "github.com/kata-containers/kata-containers/src/runtime/pkg/containerd-shim-v2/plugin"
)

func shimConfig(config *shimapi.Config) {
	config.NoReaper = true
	config.NoSubreaper = true
}

func main() {

	ctx := context.Background()
	// Shims often need the namespace from containerd args,
	// shim.Run might handle context setup internally based on args/env.
	// You might extract namespace from os.Args if needed here.

	// Handle --version flag as shims usually do
	log_file := os.Getenv("SHIM_LOG_FILE")
	if log_file == "" {
		log_file = "/Users/dub6ix/Developer/github/walteh/ec1/cmd/kata-shim-attempt-2/out/kata-shim-1746565187.log"
	}
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

	if len(os.Args) == 2 && os.Args[1] == "--version" {
		fmt.Printf("%s containerd shim (Golang): id: %q, version: %s, commit: %v\n", katautils.PROJECT, types.DefaultKataRuntimeName, katautils.VERSION, katautils.COMMIT)
		os.Exit(0)
	}

	shimapi.Run(ctx, manager.NewShimManager(types.DefaultKataRuntimeName), shimConfig)
}
