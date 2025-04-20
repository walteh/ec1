package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/walteh/ec1/pkg/agent"
	"github.com/walteh/ec1/pkg/clog"
)

func main() {
	// Parse command line flags
	var (
		hostAddr = flag.String("host", "localhost:9091", "Address for agent to listen on")
		mgtAddr  = flag.String("mgt", "http://localhost:9090", "Address of management server")
	)
	flag.Parse()

	// Create a context that is canceled on interrupt signals
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{AddSource: true}).WithGroup("agent")

	log, ctx := clog.NewLoggerFromHandler(ctx, handler)

	log.InfoContext(ctx, "Processing request", slog.String("status", "started"))

	// Set up signal handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\nReceived termination signal, shutting down...")
		cancel()
	}()

	// Create agent configuration
	config := agent.AgentConfig{
		HostAddr: *hostAddr,
		MgtAddr:  *mgtAddr,
		IDStore:  &agent.FSIDStore{},
	}

	// Create and start the agent
	a, err := agent.New(ctx, config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating agent: %+v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Starting EC1 Agent (ID: %s) on %s\n", a.ID(), *hostAddr)
	fmt.Printf("Connecting to management server at %s\n", *mgtAddr)

	if err := a.Start(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting agent: %+v\n", err)
		os.Exit(1)
	}
}
