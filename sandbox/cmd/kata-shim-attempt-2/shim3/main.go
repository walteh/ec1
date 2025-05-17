package main

// Copyright (c) 2018 HyperHQ Inc.
//
// SPDX-License-Identifier: Apache-2.0
//

import (
	_ "github.com/kata-containers/kata-containers/src/runtime/pkg/containerd-shim-v2/plugin"

	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/containerd/log"
	"github.com/kata-containers/kata-containers/src/runtime/pkg/katautils"
	"github.com/kata-containers/kata-containers/src/runtime/pkg/types"
	"github.com/kata-containers/kata-containers/src/runtime/virtcontainers"

	shimapi "github.com/containerd/containerd/v2/pkg/shim"
	manager "github.com/kata-containers/kata-containers/src/runtime/pkg/containerd-shim-v2/manager"

	"github.com/walteh/ec1/pkg/testutils"
	"github.com/walteh/ec1/pkg/vmm/kata"
	"github.com/walteh/ec1/pkg/vmm/vf"
)

func shimConfig(config *shimapi.Config) {
	config.NoReaper = true
	config.NoSubreaper = true
}

func init() {
	virtcontainers.RegisterHypervisor(virtcontainers.VirtframeworkHypervisor, kata.HypervisorRegistrationFunc(vf.NewHypervisor()))
}

func main() {

	unixtime := time.Now().Unix()

	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		log.L.WithField("error", err).Error("Failed to get user home directory")
		os.Exit(1)
	}

	wrkDir := filepath.Join(userHomeDir, "Developer", "tmp", "ksa2", "wrk")

	// Shims often need the namespace from containerd args,
	// shim.Run might handle context setup internally based on args/env.
	// You might extract namespace from os.Args if needed here.

	ctx := context.Background()

	// Handle --version flag as shims usually do
	log_file := os.Getenv("SHIM_LOG_FILE")
	if log_file == "" {
		// make the directory if it doesn't exist
		log_file = fmt.Sprintf("%s/logs/kata-shim-%d.log", wrkDir, unixtime)
		os.MkdirAll(filepath.Dir(log_file), 0755)
	}

	katcfg := os.Getenv("KATA_CONF_FILE")
	if katcfg == "" {
		katcfg = fmt.Sprintf("%s/kata.toml", wrkDir)
		os.Setenv("KATA_CONF_FILE", katcfg)
	}

	// Set up file logging
	f, err := os.OpenFile(log_file, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err == nil {
		log.L.Logger.SetOutput(io.MultiWriter(f))
		ctx = testutils.SetupSlogSimpleToWriter(ctx, f, true)

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

	// CURRENT PROBLEM:
	// DEBU[0000] remote introspection plugin filters           filters="[type==io.containerd.snapshotter.v1, id==native]"
	// ctr: failed to create shim task: Cannot find usable config file (config file "/etc/kata-containers/configuration.toml" unresolvable: file /etc/kata-containers/configuration.toml does not exist, config file "/usr/share/defaults/kata-containers/configuration.toml" unresolvable: file /usr/share/defaults/kata-containers/configuration.toml does not exist)
}
