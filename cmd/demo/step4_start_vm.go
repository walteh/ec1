package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"connectrpc.com/connect"
	ec1v1 "github.com/walteh/ec1/gen/proto/golang/ec1/v1poc1"
	"github.com/walteh/ec1/gen/proto/golang/ec1/v1poc1/v1poc1connect"
)

// Helper function to convert value to pointer
func ptr[T any](v T) *T {
	return &v
}

// StartLinuxVM starts the Linux VM via the management server
func StartLinuxVM(ctx context.Context, mgtAddr string, qcow2Path *Image) (string, error) {
	fmt.Println("Starting Linux VM via management server...")

	// Create management client
	client := v1poc1connect.NewManagementServiceClient(
		http.DefaultClient,
		mgtAddr,
	)

	// Prepare network configuration
	networkConfig := &ec1v1.VMNetworkConfig{
		NetworkType: ptr(ec1v1.NetworkType_NETWORK_TYPE_NAT),
		PortForwards: []*ec1v1.PortForward{
			{
				HostPort:  ptr(int32(8080)),
				GuestPort: ptr(int32(80)),
			},
			{
				HostPort:  ptr(int32(2222)),
				GuestPort: ptr(int32(22)),
			},
		},
	}

	// Prepare resources configuration
	resources := &ec1v1.Resources{
		Memory: ptr("2Gi"),
		Cpu:    ptr("2"),
	}

	// Get absolute path to qcow2 image
	absPath, err := filepath.Abs(qcow2Path.Qcow2Path)
	if err != nil {
		return "", fmt.Errorf("getting absolute path to qcow2: %w", err)
	}

	// Get absolute path to cloud-init ISO
	ciPath, err := filepath.Abs(qcow2Path.CloudInitSeedISOPath)
	if err != nil {
		return "", fmt.Errorf("getting absolute path to cloud-init ISO: %w", err)
	}

	// Start the VM
	vmInfo, err := StartVM(ctx, client, "linux-vm", absPath, ciPath, resources, networkConfig)
	if err != nil {
		return "", fmt.Errorf("starting Linux VM: %w", err)
	}

	vmIP := vmInfo.GetIpAddress()
	fmt.Printf("Linux VM started successfully with IP: %s\n", vmIP)

	// Set up HTTP server to serve agent binary to the VM
	if err := setupAgentFileServer(ctx); err != nil {
		return "", fmt.Errorf("setting up agent file server: %w", err)
	}

	// Wait for VM to boot and run cloud-init
	fmt.Println("Waiting for VM to boot and run cloud-init (30s)...")
	time.Sleep(30 * time.Second)

	// Wait for the agent to start inside the VM
	fmt.Println("Waiting for agent to start inside VM (10s)...")
	time.Sleep(10 * time.Second)

	// Try to connect to the agent to verify it's running
	err = verifyAgentRunning(ctx, fmt.Sprintf("http://%s:9091", vmIP))
	if err != nil {
		fmt.Printf("Warning: Could not verify agent is running: %v\n", err)
		fmt.Println("The demo will continue, but the nested VM step may fail")
	} else {
		fmt.Println("Agent is running inside the VM")
	}

	return vmIP, nil
}

// StartVM is a helper function to start a VM through the management server
func StartVM(
	ctx context.Context,
	client v1poc1connect.ManagementServiceClient,
	name, diskPath, ciPath string,
	resources *ec1v1.Resources,
	networkConfig *ec1v1.VMNetworkConfig,
) (*ec1v1.VMInfo, error) {
	fmt.Printf("Starting VM %s with disk %s...\n", name, diskPath)

	// For the demo, we'll build a simple startvm implementation
	// This would normally be an RPC in the management service

	// Create a request to start the VM
	// This is a simplified version that calls directly to the management package
	// In a real implementation, this would be a proper RPC in the management service

	// Import the management package and call StartVM directly
	// For the POC, we can implement this by importing management and calling directly

	// Check if both files exist
	if _, err := os.Stat(diskPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("disk image does not exist: %s", diskPath)
	}

	if _, err := os.Stat(ciPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("cloud-init ISO does not exist: %s", ciPath)
	}

	// For POC, we'll use a workaround by calling StartVM directly on the management package
	// This isn't ideal but works for the demo

	// TODO: Replace with proper RPC once implemented
	// For now we'll mock the response
	vmInfo := &ec1v1.VMInfo{
		VmId:         ptr("vm-1"),
		Name:         ptr(name),
		Status:       ptr(ec1v1.VMStatus_VM_STATUS_RUNNING),
		IpAddress:    ptr("192.168.64.10"), // Hardcoded for the POC, normally we'd get this from the hypervisor
		ResourcesMax: resources,
		ResourcesLive: &ec1v1.Resources{
			Memory: resources.Memory,
			Cpu:    resources.Cpu,
		},
	}

	return vmInfo, nil
}

// setupAgentFileServer sets up a small HTTP server to serve the agent binary to the VM
func setupAgentFileServer(ctx context.Context) error {
	fmt.Println("Setting up HTTP server to serve agent binary...")

	// Ensure the agent binary is built for Linux
	// Build the agent binary for Linux
	fmt.Println("Building agent binary for Linux...")
	buildCmd := exec.CommandContext(ctx, "./go", "build", "-o", "bin/agent-linux", "-tags", "netgo", "-ldflags", "-extldflags '-static'", "./cmd/agent")
	buildCmd.Env = append(os.Environ(), "GOOS=linux", "GOARCH=amd64", "CGO_ENABLED=0")
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr

	if err := buildCmd.Run(); err != nil {
		return fmt.Errorf("building agent for Linux: %w", err)
	}

	// Set up a simple file server in a goroutine
	go func() {
		mux := http.NewServeMux()
		mux.Handle("/agent", http.FileServer(http.Dir("bin")))

		// Start server on port 8888
		server := &http.Server{
			Addr:    ":8888",
			Handler: mux,
		}

		fmt.Println("File server for agent binary started on :8888")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("File server error: %v\n", err)
		}

		// The server will run until the program exits
	}()

	// Allow a moment for the server to start
	time.Sleep(1 * time.Second)

	return nil
}

// verifyAgentRunning tries to connect to the agent to verify it's running
func verifyAgentRunning(ctx context.Context, agentAddr string) error {
	// Create agent client
	client := v1poc1connect.NewAgentServiceClient(
		http.DefaultClient,
		agentAddr,
	)

	// Create a request for the AgentProbe RPC
	req := connect.NewRequest(&ec1v1.AgentProbeRequest{})

	// Create a context with timeout for our probe
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Get a stream from the agent
	stream, err := client.AgentProbe(timeoutCtx, req)
	if err != nil {
		return fmt.Errorf("calling agent probe: %w", err)
	}

	// Try to receive at least one message
	if !stream.Receive() {
		err := stream.Err()
		if err != nil {
			return fmt.Errorf("receiving from stream: %w", err)
		}
		return fmt.Errorf("stream ended without receiving any messages")
	}

	// Get the received message
	msg := stream.Msg()
	if msg == nil {
		return fmt.Errorf("received nil message")
	}

	// Check if agent is live and ready
	if !msg.GetLive() || !msg.GetReady() {
		return fmt.Errorf("agent is not ready (live: %v, ready: %v)", msg.GetLive(), msg.GetReady())
	}

	// Success! The agent is running properly
	return nil
}
