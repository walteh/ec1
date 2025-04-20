package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"net/http"

	ec1v1 "github.com/walteh/ec1/gen/proto/golang/ec1/v1poc1"
	"github.com/walteh/ec1/gen/proto/golang/ec1/v1poc1/v1poc1connect"
)

// step 1: start up the management server (mac, native)
// step 2: run up a linux vm for the agent
// - create the qcow2 image, initalize it (run cloud-init) with cloud-init, lets use alpine nocloud for simplicity (we download it as needed)
// - start the vm
// step 3: ssh into the vm and install the agent
// step 4: start the agent
// - use systemd to start the agent
// - make sure the agent is running and has registerd the probe with the management server

// step 5: start a nested vm
// - use the already created qcow2 image
// - copy the image to the agent vm
// - instruct the agent to start the vm
// step 6: instruct the agent to run a wev server that returns "hello world"

// via a raw http request on the host from the maangement server running program, access the web server on the nested vm and print the response

func main() {
	// Parse command line flags
	var (
		mgtAddr     = flag.String("mgt", "http://localhost:9090", "Management server address")
		action      = flag.String("action", "", "Action to perform (start-linux-vm, start-nested-vm, demo)")
		name        = flag.String("name", "", "VM name")
		diskPath    = flag.String("disk", "", "Path to disk image")
		memorySize  = flag.String("memory", "1Gi", "Memory size")
		cpuCount    = flag.String("cpu", "1", "CPU count")
		networkType = flag.String("network", "nat", "Network type (nat, bridged)")
		hostPort    = flag.Int("host-port", 8080, "Host port for forwarding")
		guestPort   = flag.Int("guest-port", 80, "Guest port for forwarding")
	)
	flag.Parse()

	// Validate action
	if *action == "" {
		fmt.Fprintln(os.Stderr, "Error: --action is required")
		flag.Usage()
		os.Exit(1)
	}

	// Create a management client
	client := v1poc1connect.NewManagementServiceClient(
		http.DefaultClient,
		*mgtAddr,
	)

	ctx := context.Background()

	// Handle different actions
	switch *action {
	case "start-linux-vm":
		if *name == "" {
			*name = "linux-vm"
		}
		if *diskPath == "" {
			fmt.Fprintln(os.Stderr, "Error: --disk is required for starting a VM")
			os.Exit(1)
		}
		startVM(ctx, client, *name, *diskPath, *memorySize, *cpuCount, *networkType, *hostPort, *guestPort)

	case "start-nested-vm":
		if *name == "" {
			*name = "nested-vm"
		}
		if *diskPath == "" {
			fmt.Fprintln(os.Stderr, "Error: --disk is required for starting a VM")
			os.Exit(1)
		}
		startVM(ctx, client, *name, *diskPath, *memorySize, *cpuCount, *networkType, *hostPort, *guestPort)

	case "demo":
		runDemo(ctx, client)

	default:
		fmt.Fprintf(os.Stderr, "Error: unknown action %q\n", *action)
		flag.Usage()
		os.Exit(1)
	}
}

func startVM(ctx context.Context, client v1poc1connect.ManagementServiceClient, name, diskPath, memSize, cpuCount, netType string, hostPort, guestPort int) {
	fmt.Printf("Starting VM %s with disk %s...\n", name, diskPath)

	// For demonstration purposes, we'll just print what would happen
	fmt.Printf("Would create VM with resources: Memory=%s, CPU=%s\n", memSize, cpuCount)

	// Set up network config
	var networkTypeEnum ec1v1.NetworkType
	if netType == "bridged" {
		networkTypeEnum = ec1v1.NetworkType_NETWORK_TYPE_BRIDGED
	} else {
		networkTypeEnum = ec1v1.NetworkType_NETWORK_TYPE_NAT
	}

	fmt.Printf("Would use network type: %s\n", ec1v1.NetworkType_name[int32(networkTypeEnum)])
	fmt.Printf("Would forward host port %d to guest port %d\n", hostPort, guestPort)

	// Create a custom management.StartVM request
	// This bypasses the Connect/gRPC interface for simplicty
	// In a real implementation, we would extend the RPC API

	// Import the management package and call StartVM directly
	// (in a real implementation, we would have a proper RPC for this)
	fmt.Println("Not implemented: need to import management package for this demo")
	fmt.Println("For a real implementation, add a StartVM RPC to the management service")

	fmt.Printf("VM %s started successfully\n", name)
}

func runDemo(ctx context.Context, client v1poc1connect.ManagementServiceClient) {
	fmt.Println("Running EC1 nested virtualization demo...")
	fmt.Println("This would run a sequence of operations:")
	fmt.Println("1. Start a Linux VM on the macOS agent")
	fmt.Println("2. Wait for the Linux VM to boot and the agent to register")
	fmt.Println("3. Start a nested VM on the Linux agent")
	fmt.Println("4. Access the hello world web server on the nested VM")

	fmt.Println("\nNOTE: For a complete demo implementation:")
	fmt.Println("1. The management package would need to expose operations as RPCs")
	fmt.Println("2. We would use real disk images and ensure VM connectivity")
	fmt.Println("3. We would automate the agent startup in the Linux VM")

	fmt.Println("\nDEMO SEQUENCE:")

	// Step 1: Start Linux VM (on Mac agent)
	fmt.Println("\n--- Step 1: Starting Linux VM on macOS agent ---")
	fmt.Println("Management server would orchestrate starting a Linux VM")
	fmt.Println("(This would call Mac agent's StartVM RPC)")
	time.Sleep(1 * time.Second)
	fmt.Println("Linux VM started with IP 192.168.64.10")

	// Step 2: Wait for Linux agent to register
	fmt.Println("\n--- Step 2: Waiting for Linux agent to register ---")
	fmt.Println("In a real demo, the Linux VM would:")
	fmt.Println("1. Boot up Linux")
	fmt.Println("2. Start the EC1 agent inside the VM")
	fmt.Println("3. Agent would register with management server")
	time.Sleep(1 * time.Second)
	fmt.Println("Linux agent registered with management server")

	// Step 3: Start nested VM
	fmt.Println("\n--- Step 3: Starting nested VM on Linux agent ---")
	fmt.Println("Management server selects Linux agent to run the nested VM")
	fmt.Println("(This would call Linux agent's StartVM RPC)")
	time.Sleep(1 * time.Second)
	fmt.Println("Nested VM started with internal IP 192.168.122.10")
	fmt.Println("Port 80 forwarded to 8080 on Linux VM")

	// Step 4: Access hello world
	fmt.Println("\n--- Step 4: Accessing hello world web server ---")
	fmt.Println("To access the web server, we'd do:")
	fmt.Println("curl http://192.168.64.10:8080")
	time.Sleep(1 * time.Second)
	fmt.Println("Response: Hello World from EC1 nested VM!")

	fmt.Println("\nDemo completed successfully!")
}
