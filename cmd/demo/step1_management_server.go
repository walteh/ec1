package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/walteh/ec1/pkg/management"
)

// StartManagementServer starts the management server in a separate process
func StartManagementServer(ctx context.Context, addr string) error {
	fmt.Println("Starting management server...")

	// Check if the management server binary exists
	_, err := os.Stat("bin/mgt")
	if err != nil {
		// Try to build it
		fmt.Println("Building management server...")
		buildCmd := exec.CommandContext(ctx, "./go", "build", "-o", "bin/mgt", "./cmd/mgt")
		buildCmd.Stdout = os.Stdout
		buildCmd.Stderr = os.Stderr
		if err := buildCmd.Run(); err != nil {
			return fmt.Errorf("failed to build management server: %w", err)
		}
	}

	// Start the management server in a separate process
	mgtCmd := exec.CommandContext(ctx, "bin/mgt", "--host", addr)
	mgtCmd.Stdout = os.Stdout
	mgtCmd.Stderr = os.Stderr

	if err := mgtCmd.Start(); err != nil {
		return fmt.Errorf("failed to start management server: %w", err)
	}

	// Wait a moment for the server to start
	time.Sleep(2 * time.Second)
	fmt.Println("Management server started successfully")

	return nil
}

// For testing and direct invocation, we also provide a function that runs
// the management server in the same process
func StartManagementServerInProcess(ctx context.Context, addr string) error {
	fmt.Println("Starting management server in current process...")

	config := management.ServerConfig{
		HostAddr: addr,
	}

	server := management.New(config)

	fmt.Printf("EC1 Management Server listening on %s\n", addr)
	return server.Start(ctx)
}
