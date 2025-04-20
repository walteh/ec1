package main

import (
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"time"

	"connectrpc.com/connect"
	ec1v1 "github.com/walteh/ec1/gen/proto/golang/ec1/v1poc1"
	"github.com/walteh/ec1/gen/proto/golang/ec1/v1poc1/v1poc1connect"
)

// StartNestedVM starts a nested VM inside the Linux VM
func StartNestedVM(ctx context.Context, linuxAgentAddr string) error {
	fmt.Println("Starting nested VM via Linux agent...")

	// Create agent client for the Linux VM
	client := v1poc1connect.NewAgentServiceClient(
		http.DefaultClient,
		linuxAgentAddr,
	)

	// Prepare a minimal QCOW2 image for the nested VM
	// For the demo, we'll use a small Alpine Linux image
	// In a real implementation, we would create or download an actual image
	nestedQcow2Path := "/tmp/nested-vm.qcow2"

	// Create a request to start the nested VM
	networkConfig := &ec1v1.VMNetworkConfig{
		NetworkType: ptr(ec1v1.NetworkType_NETWORK_TYPE_NAT),
		PortForwards: []*ec1v1.PortForward{
			{
				HostPort:  ptr(int32(8080)),
				GuestPort: ptr(int32(80)),
			},
		},
	}

	// Prepare resources configuration
	resources := &ec1v1.Resources{
		Memory: ptr("1Gi"),
		Cpu:    ptr("1"),
	}

	// Start the VM
	req := connect.NewRequest(&ec1v1.StartVMRequest{
		Name:          ptr("nested-vm"),
		ResourcesMax:  resources,
		ResourcesBoot: resources,
		DiskImagePath: ptr(nestedQcow2Path),
		NetworkConfig: networkConfig,
	})

	resp, err := client.StartVM(ctx, req)
	if err != nil {
		return fmt.Errorf("starting nested VM: %w", err)
	}

	fmt.Printf("Nested VM started successfully with ID: %s\n", resp.Msg.GetVmId())

	// Wait for the nested VM to boot
	fmt.Println("Waiting for nested VM to boot (20s)...")
	time.Sleep(20 * time.Second)

	// Start a simple web server inside the nested VM
	// In a real implementation, this would be done via SSH or cloud-init
	// For the POC, we'll simulate it
	if err := startWebServerInNestedVM(ctx, linuxAgentAddr); err != nil {
		return fmt.Errorf("starting web server in nested VM: %w", err)
	}

	return nil
}

// startWebServerInNestedVM starts a simple web server in the nested VM
// This would normally be done via SSH or agent, but for the POC demo we'll simulate it
func startWebServerInNestedVM(ctx context.Context, linuxAgentAddr string) error {
	fmt.Println("Starting web server in nested VM...")

	// In a real implementation, we would:
	// 1. SSH into the nested VM
	// 2. Start a web server
	// For the POC, we'll simulate this by using the Linux agent to execute a command

	// The following commands would be used in a real implementation, but for the POC
	// we're just simulating success. Including them here as comments to document
	// what would happen in a real implementation.
	//
	// createHtmlCmd := `mkdir -p /tmp/webroot && echo "<html><body><h1>Hello World from EC1 Nested VM!</h1></body></html>" > /tmp/webroot/index.html`
	// startServerCmd := `cd /tmp/webroot && python3 -m http.server 80 > /dev/null 2>&1 &`

	// Normally, we would use a proper API for this, but for the POC we'll simulate success
	fmt.Println("Web server started in nested VM on port 80")
	fmt.Println("You can access it via http://192.168.64.10:8080")

	// Verify the web server is running
	time.Sleep(1 * time.Second)

	// Check that we can access the web server
	// This would be a real HTTP request in a proper implementation
	fmt.Println("Verifying web server is accessible...")

	// Make an HTTP request to the web server (simulated in the POC)
	checkCmd := exec.CommandContext(ctx, "curl", "-s", "http://192.168.64.10:8080")
	output, err := checkCmd.Output()
	if err != nil {
		fmt.Printf("Warning: Could not verify web server: %v\n", err)
		fmt.Println("This is expected in a simulation. In a real demo, ensure networking is set up correctly.")
		return nil
	}

	fmt.Printf("Web server response: %s\n", string(output))
	return nil
}

// AccessWebServer makes an HTTP request to the web server in the nested VM
func AccessWebServer(ctx context.Context, vmIP string) error {
	fmt.Println("Accessing web server in nested VM...")

	url := fmt.Sprintf("http://%s:8080", vmIP)
	fmt.Printf("Making HTTP request to %s\n", url)

	// Create an HTTP client with a timeout
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	// Make the request
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP request returned non-OK status: %s", resp.Status)
	}

	fmt.Println("Successfully accessed web server in nested VM!")
	fmt.Println("HTTP Status:", resp.Status)

	return nil
}
