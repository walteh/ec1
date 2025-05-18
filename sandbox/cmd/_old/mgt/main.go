package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/walteh/ec1/sandbox/pkg/cloud/management"
)

func main() {
	// Parse command line flags
	var (
		hostAddr = flag.String("host", "localhost:9090", "Address to listen on")
	)
	flag.Parse()

	// Create a context that is canceled on interrupt signals
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\nReceived termination signal, shutting down...")
		cancel()
	}()

	// Create and start the management server
	config := management.ServerConfig{
		// Host: *hostAddr,
	}

	server, err := management.New(ctx, config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating server: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Starting EC1 Management Server on %s\n", *hostAddr)
	if _, err := server.Start(ctx, *hostAddr); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting server: %v\n", err)
		os.Exit(1)
	}
}
