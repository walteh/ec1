package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/walteh/ec1/pkg/agent"
)

func main() {
	// Parse command line flags
	var (
		hostAddr = flag.String("host", "localhost:9091", "Address for agent to listen on")
		agentID  = flag.String("id", "agent-1", "Unique ID for this agent")
		mgtAddr  = flag.String("mgt", "http://localhost:9090", "Address of management server")
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

	// Create agent configuration
	config := agent.AgentConfig{
		HostAddr: *hostAddr,
		AgentID:  *agentID,
		MgtAddr:  *mgtAddr,
	}

	// Create and start the agent
	a, err := agent.New(ctx, config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating agent: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Starting EC1 Agent (ID: %s) on %s\n", *agentID, *hostAddr)
	fmt.Printf("Connecting to management server at %s\n", *mgtAddr)

	if err := a.Start(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting agent: %v\n", err)
		os.Exit(1)
	}
}
