package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"connectrpc.com/connect"
	ec1v1 "github.com/walteh/ec1/gen/proto/golang/ec1/v1poc1"
	"github.com/walteh/ec1/gen/proto/golang/ec1/v1poc1/v1poc1connect"
	"github.com/walteh/ec1/pkg/agent"
	"github.com/walteh/ec1/pkg/management"
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
		clean      = flag.Bool("clean", false, "Clean up before starting (removes existing binaries and images)")
		// diskPath   = flag.String("disk", "", "Path to disk image")
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
	if err := runEC1Demo(ctx, *mgtAddr, *localAgent); err != nil {
		fmt.Fprintf(os.Stderr, "Error running demo: %v\n", err)
		os.Exit(1)
	}

}
func ptr[T any](v T) *T {
	return &v
}

type DemoContext struct {
	ManagementClient v1poc1connect.ManagementServiceClient
	AgentClient      v1poc1connect.AgentServiceClient
	Qcow2Image       *Image
}

// runEC1Demo runs the complete EC1 nested virtualization demo
func runEC1Demo(ctx context.Context) error {
	fmt.Println("Starting EC1 Nested Virtualization Demo")
	fmt.Println("=======================================")

	// Create a context with cancellation for the servers
	serverCtx, cancelServers := context.WithCancel(ctx)
	defer cancelServers()

	// Use a WaitGroup to ensure servers have started before proceeding
	var wg sync.WaitGroup

	// Step 1: Start management server in-process
	fmt.Println("\n=== Step 1: Starting Management Server ===")

	managerInstance := management.New(management.ServerConfig{
		HostAddr: "localhost:9090",
	})

	managerClient, cleanup := management.NewInMemoryManagementClient(ctx, managerInstance)
	defer cleanup()

	agentInstance, err := agent.New(ctx, agent.AgentConfig{
		HostAddr:                 "localhost:9091",
		InMemoryManagementClient: managerClient,
	})

	agentClient, cleanup := agent.NewInMemoryAgentClient(ctx, agentInstance)
	defer cleanup()

	demoCtx := DemoContext{
		ManagementClient: managerClient,
		AgentClient:      agentClient,
	}

	// Step 3: Create QCOW2 image
	fmt.Println("\n=== Step 3: Creating QCOW2 Image ===")
	qcow2Path, err := CreateQCOW2Image(ctx)
	if err != nil {
		return fmt.Errorf("creating QCOW2 image: %w", err)
	}

	defer qcow2Path.Cleanup()

	// Steps 4-6: Start Linux VM and set up agent
	fmt.Println("\n=== Steps 4-6: Starting Linux VM and Setting Up Agent ===")
	resp, err := demoCtx.ManagementClient.StartVM(ctx, connect.NewRequest(&ec1v1.StartVMRequest{
		Name: ptr("demo-vm"),
		DiskImage: &ec1v1.DiskImage{
			Path: ptr(qcow2Path.Qcow2Path),
			Type: ptr(ec1v1.DiskImageType_DISK_IMAGE_TYPE_QCOW2),
		},
		CloudInit: &ec1v1.CloudInitConfig{
			IsoPath: ptr(qcow2Path.CloudInitSeedISOPath),
		},
		ResourcesMax: &ec1v1.Resources{
			Cpu:    ptr("1"),
			Memory: ptr("1024"),
		},
		NetworkConfig: &ec1v1.VMNetworkConfig{
			NetworkType: ptr(ec1v1.NetworkType_NETWORK_TYPE_NAT),
		},
	})
)
	if err != nil {
		return fmt.Errorf("starting VM: %w", err)
	}

	vmIP := resp.Msg.GetIpAddress()

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
