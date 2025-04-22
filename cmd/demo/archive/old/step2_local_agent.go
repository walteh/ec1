package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/walteh/ec1/gen/proto/golang/ec1/v1poc1/v1poc1connect"
	"github.com/walteh/ec1/pkg/agent"
)

// StartLocalAgent starts the agent on the local machine in a separate process
func StartLocalAgent(ctx context.Context, agentAddr, mgtAddr string) error {
	fmt.Println("Starting local agent...")

	// Check if the agent binary exists
	_, err := os.Stat("bin/agent")
	if err != nil {
		// Try to build it
		fmt.Println("Building agent...")
		buildCmd := exec.CommandContext(ctx, "./go", "build", "-o", "bin/agent", "./cmd/agent")
		buildCmd.Stdout = os.Stdout
		buildCmd.Stderr = os.Stderr
		if err := buildCmd.Run(); err != nil {
			return fmt.Errorf("failed to build agent: %w", err)
		}
	}

	// Start the agent in a separate process
	agentCmd := exec.CommandContext(ctx, "bin/agent", "--host", agentAddr, "--mgt", mgtAddr)
	agentCmd.Stdout = os.Stdout
	agentCmd.Stderr = os.Stderr

	if err := agentCmd.Start(); err != nil {
		return fmt.Errorf("failed to start agent: %w", err)
	}

	// Wait a moment for the agent to start and register
	time.Sleep(3 * time.Second)
	fmt.Println("Local agent started successfully")

	return nil
}

// For testing and direct invocation, we also provide a function that runs
// the agent in the same process
func StartLocalAgentInProcess(ctx context.Context, agentAddr, mgtAddr string) (v1poc1connect.AgentServiceClient, error) {
	fmt.Println("Starting local agent in current process...")

	// Create agent configuration
	config := agent.AgentConfig{
		HostAddr: agentAddr,
		MgtAddr:  mgtAddr,
		IDStore:  &agent.FSIDStore{},
	}

	// Create and start the agent
	a, err := agent.New(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("creating agent: %w", err)
	}

	fmt.Printf("Starting EC1 Agent (ID: %s) on %s\n", a.ID(), agentAddr)
	fmt.Printf("Connecting to management server at %s\n", mgtAddr)

	err = a.Start(ctx)
	if err != nil {
		return nil, fmt.Errorf("starting agent: %w", err)
	}

	return a, nil
}
