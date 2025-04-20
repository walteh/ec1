package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

// step 1: start up the management server (mac, native)
// step 2: start up the management server local agent (mac, native)
// step 3: create the qcow2 image, initalize it (run cloud-init) with cloud-init, lets use alpine nocloud for simplicity (we download it as needed)
// step 4: instruct the agent to start the vm
// step 5: the agent should ssh into the vm and install the agent
// step 6: start the agent
// - use systemd to start the agent
// - make sure the agent is running and has registerd the probe with the management server
// - at this point, we should have two agents running, one as part of our management server, and one as part of the nested vm

// step 7: start a nested vm
// - use the already created qcow2 image
// - copy the image to the agent vm
// - instruct the agent to start the vm
// step 8: instruct the agent to run a web server that returns "hello world"

// via a raw http request on the host from the maangement server running program, access the web server on the nested vm and print the response

func main() {
	// Parse command line flags
	var (
		mgtAddr    = flag.String("mgt", "http://localhost:9090", "Management server address")
		localAgent = flag.String("agent", "localhost:9091", "Local agent address")
		action     = flag.String("action", "demo", "Action to perform (start-mgt, start-agent, create-image, start-linux-vm, start-nested-vm, demo)")
		clean      = flag.Bool("clean", false, "Clean up before starting (removes existing binaries and images)")
		diskPath   = flag.String("disk", "", "Path to disk image")
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

	// Check if we need to clean up first
	if *clean {
		fmt.Println("Cleaning up previous run...")
		cleanupPreviousRun()
	}

	// Handle different actions
	switch *action {
	case "start-mgt":
		if err := StartManagementServer(ctx, *mgtAddr); err != nil {
			fmt.Fprintf(os.Stderr, "Error starting management server: %v\n", err)
			os.Exit(1)
		}

	case "start-agent":
		if err := StartLocalAgent(ctx, *localAgent, *mgtAddr); err != nil {
			fmt.Fprintf(os.Stderr, "Error starting local agent: %v\n", err)
			os.Exit(1)
		}

	case "create-image":
		imagePath, err := CreateQCOW2Image(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating QCOW2 image: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("QCOW2 image created: %s\n", imagePath)

	case "start-linux-vm":
		if *diskPath == "" {
			fmt.Fprintln(os.Stderr, "Error: --disk is required for starting a VM")
			os.Exit(1)
		}
		vmIP, err := StartLinuxVM(ctx, *mgtAddr, *diskPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error starting Linux VM: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Linux VM started with IP: %s\n", vmIP)

	case "start-nested-vm":
		linuxAgentAddr := fmt.Sprintf("http://192.168.64.10:9091")
		if err := StartNestedVM(ctx, linuxAgentAddr); err != nil {
			fmt.Fprintf(os.Stderr, "Error starting nested VM: %v\n", err)
			os.Exit(1)
		}

	case "demo":
		if err := runEC1Demo(ctx, *mgtAddr, *localAgent); err != nil {
			fmt.Fprintf(os.Stderr, "Error running demo: %v\n", err)
			os.Exit(1)
		}

	default:
		fmt.Fprintf(os.Stderr, "Error: unknown action %q\n", *action)
		flag.Usage()
		os.Exit(1)
	}
}

// runEC1Demo runs the complete EC1 nested virtualization demo
func runEC1Demo(ctx context.Context, mgtAddr, localAgentAddr string) error {
	fmt.Println("Starting EC1 Nested Virtualization Demo")
	fmt.Println("=======================================")

	// Create a context with cancellation for the servers
	serverCtx, cancelServers := context.WithCancel(ctx)
	defer cancelServers()

	// Use a WaitGroup to ensure servers have started before proceeding
	var wg sync.WaitGroup

	// Step 1: Start management server in-process
	fmt.Println("\n=== Step 1: Starting Management Server ===")
	wg.Add(1)
	errCh := make(chan error, 2) // Channel to collect errors from goroutines

	go func() {
		defer wg.Done()
		fmt.Println("Management server starting in-process...")
		if err := StartManagementServerInProcess(serverCtx, strings.TrimPrefix(mgtAddr, "http://")); err != nil {
			errCh <- fmt.Errorf("management server error: %w", err)
		}
	}()

	// Let the management server start up
	time.Sleep(2 * time.Second)

	// Step 2: Start local agent in-process
	fmt.Println("\n=== Step 2: Starting Local Agent ===")
	wg.Add(1)
	go func() {
		defer wg.Done()
		fmt.Println("Local agent starting in-process...")
		if err := StartLocalAgentInProcess(serverCtx, localAgentAddr, mgtAddr); err != nil {
			errCh <- fmt.Errorf("local agent error: %w", err)
		}
	}()

	// Let the agent start up and register
	time.Sleep(3 * time.Second)

	// Check for startup errors
	select {
	case err := <-errCh:
		return err
	default:
		// No errors, continue
	}

	// Step 3: Create QCOW2 image
	fmt.Println("\n=== Step 3: Creating QCOW2 Image ===")
	qcow2Path, err := CreateQCOW2Image(ctx)
	if err != nil {
		return fmt.Errorf("creating QCOW2 image: %w", err)
	}

	// Steps 4-6: Start Linux VM and set up agent
	fmt.Println("\n=== Steps 4-6: Starting Linux VM and Setting Up Agent ===")
	vmIP, err := StartLinuxVM(ctx, mgtAddr, qcow2Path)
	if err != nil {
		return fmt.Errorf("starting Linux VM: %w", err)
	}

	// Step 7-8: Start nested VM and web server
	fmt.Println("\n=== Steps 7-8: Starting Nested VM and Web Server ===")
	linuxAgentAddr := fmt.Sprintf("http://%s:9091", vmIP)
	if err := StartNestedVM(ctx, linuxAgentAddr); err != nil {
		return fmt.Errorf("starting nested VM: %w", err)
	}

	// Final step: Access web server
	fmt.Println("\n=== Accessing Web Server on Nested VM ===")
	if err := AccessWebServer(ctx, vmIP); err != nil {
		return fmt.Errorf("accessing web server: %w", err)
	}

	fmt.Println("\nEC1 Nested Virtualization Demo completed successfully!")
	fmt.Println("To clean up resources, restart the demo with --clean flag")
	fmt.Println("Press Ctrl+C to exit")

	// The demo is complete, but we'll keep the management server and agent running
	// until the context is canceled (e.g., by pressing Ctrl+C)
	<-ctx.Done()

	return nil
}

// cleanupPreviousRun cleans up binaries and artifacts from previous runs
func cleanupPreviousRun() {
	// Try to remove binary files
	_ = os.RemoveAll("bin/mgt")
	_ = os.RemoveAll("bin/agent")
	_ = os.RemoveAll("bin/agent-linux")

	// Try to remove image files
	_ = os.RemoveAll("images")
}
