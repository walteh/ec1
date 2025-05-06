package main

import (
	"context"
	"io"
	"os"

	"github.com/containerd/containerd/v2/pkg/shim"
	"github.com/containerd/log"
	"github.com/walteh/ec1/cmd/kata-shim-attempt-2/shim/containerd"
)

func withoutReaper(config *shim.Config) {
	config.NoReaper = true
}

func main() {
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
	}()

	shim.Run(context.Background(), containerd.NewManager("io.containerd.kata.v2"), withoutReaper)
}
