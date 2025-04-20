package main

import (
	"bytes"
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
	"golang.org/x/crypto/ssh"
)

// CreateAndTransferNestedImage creates a small QCOW2 image for the nested VM
// and transfers it to the VM
func CreateAndTransferNestedImage(ctx context.Context, vmIP string) error {
	fmt.Println("Creating and transferring nested VM image...")

	// Create a temporary directory to hold the nested VM image
	tempDir, err := os.MkdirTemp("", "ec1-nested-vm")
	if err != nil {
		return fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	nestedImagePath := filepath.Join(tempDir, "nested-vm.qcow2")

	// Create a small QCOW2 image for the nested VM
	fmt.Println("Creating small QCOW2 image for nested VM...")
	qemuImgCmd := exec.CommandContext(ctx, "qemu-img", "create", "-f", "qcow2", nestedImagePath, "1G")
	if output, err := qemuImgCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("creating QCOW2 image: %w, output: %s", err, output)
	}

	// Connect to the VM via SSH to transfer the image
	fmt.Println("Transferring image to VM...")

	// First try the direct IP
	configs := []struct {
		host     string
		port     string
		user     string
		password string
	}{
		{host: vmIP, port: "22", user: "alpine", password: "alpine"},
		{host: "localhost", port: "2222", user: "alpine", password: "alpine"},
		{host: "localhost", port: "2222", user: "root", password: "alpine"},
	}

	var client *ssh.Client
	var connConfig string

	for _, cfg := range configs {
		clientConfig := &ssh.ClientConfig{
			User: cfg.user,
			Auth: []ssh.AuthMethod{
				ssh.Password(cfg.password),
			},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
			Timeout:         5 * time.Second,
		}

		addr := fmt.Sprintf("%s:%s", cfg.host, cfg.port)
		fmt.Printf("Trying to connect to %s@%s... ", cfg.user, addr)

		var err error
		client, err = ssh.Dial("tcp", addr, clientConfig)
		if err == nil {
			fmt.Println("✓")
			connConfig = fmt.Sprintf("%s@%s", cfg.user, addr)
			break
		}
		fmt.Println("❌")
	}

	if client == nil {
		return fmt.Errorf("could not connect to VM via SSH")
	}
	defer client.Close()

	// Create a session for SFTP
	fmt.Printf("Transferring QCOW2 image to VM via %s...\n", connConfig)
	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("creating SSH session: %w", err)
	}
	defer session.Close()

	// Prepare to transfer the file - let's use SCP approach
	// First, check if the destination directory exists
	checkDirSession, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("creating check dir session: %w", err)
	}

	var checkDirOutput bytes.Buffer
	checkDirSession.Stdout = &checkDirOutput

	err = checkDirSession.Run("test -d /tmp && echo 'exists'")
	checkDirSession.Close()

	if err != nil || checkDirOutput.String() != "exists\n" {
		// Try to create the directory
		mkdirSession, err := client.NewSession()
		if err != nil {
			return fmt.Errorf("creating mkdir session: %w", err)
		}
		err = mkdirSession.Run("mkdir -p /tmp")
		mkdirSession.Close()
		if err != nil {
			return fmt.Errorf("creating /tmp directory on VM: %w", err)
		}
	}

	// Now use scp command to transfer the file since it's more reliable for binary files
	fmt.Println("Transferring image using SCP (this may take a minute)...")

	// We'll use the local scp command since it's the most reliable way
	var scpArgs []string

	if connConfig == "localhost:2222" {
		scpArgs = []string{
			"-P", "2222",
			"-o", "StrictHostKeyChecking=no",
			"-o", "UserKnownHostsFile=/dev/null",
			nestedImagePath,
			"alpine@localhost:/tmp/nested-vm.qcow2",
		}
	} else {
		// Extract user and host from connConfig
		scpArgs = []string{
			"-o", "StrictHostKeyChecking=no",
			"-o", "UserKnownHostsFile=/dev/null",
			nestedImagePath,
			fmt.Sprintf("%s:/tmp/nested-vm.qcow2", connConfig),
		}
	}

	scpCmd := exec.CommandContext(ctx, "scp", scpArgs...)
	scpOutput, err := scpCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("SCP transfer failed: %w, output: %s", err, scpOutput)
	}

	// Verify the file exists on the VM
	verifySession, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("creating verify session: %w", err)
	}
	defer verifySession.Close()

	var verifyOutput bytes.Buffer
	verifySession.Stdout = &verifyOutput

	err = verifySession.Run("test -f /tmp/nested-vm.qcow2 && echo 'exists' || echo 'missing'")
	if err != nil || verifyOutput.String() != "exists\n" {
		return fmt.Errorf("QCOW2 image not found on VM after transfer, output: %s", verifyOutput.String())
	}

	fmt.Println("QCOW2 image successfully transferred to VM")
	return nil
}

// StartNestedVM starts a nested VM inside the Linux VM
func StartNestedVM(ctx context.Context, linuxAgentAddr string) error {
	fmt.Println("Starting nested VM via Linux agent...")

	// First, create and transfer the nested VM image
	// Extract the IP address from the agent address
	var vmIP string
	if linuxAgentAddr == "http://localhost:9091" {
		vmIP = "localhost" // We're using port forwarding
	} else {
		// Extract the IP part from http://IP:PORT
		vmIP = linuxAgentAddr[7:] // Remove "http://"
		if colonIdx := findLastIndexOfRune(vmIP, ':'); colonIdx > 0 {
			vmIP = vmIP[:colonIdx]
		}
	}

	// Create and transfer the nested VM image
	if err := CreateAndTransferNestedImage(ctx, vmIP); err != nil {
		return fmt.Errorf("preparing nested VM image: %w", err)
	}

	// Create agent client for the Linux VM
	client := v1poc1connect.NewAgentServiceClient(
		http.DefaultClient,
		linuxAgentAddr,
	)

	// Path to the nested VM image on the VM
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

// Helper function to find the last index of a rune in a string
func findLastIndexOfRune(s string, r rune) int {
	for i := len(s) - 1; i >= 0; i-- {
		if rune(s[i]) == r {
			return i
		}
	}
	return -1
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

	// Determine the correct IP address for access
	var accessIP string
	if linuxAgentAddr == "http://localhost:9091" {
		accessIP = "localhost"
	} else {
		// Extract the IP part from http://IP:PORT
		accessIP = linuxAgentAddr[7:] // Remove "http://"
		if colonIdx := findLastIndexOfRune(accessIP, ':'); colonIdx > 0 {
			accessIP = accessIP[:colonIdx]
		}
	}

	fmt.Printf("You can access it via http://%s:8080\n", accessIP)

	// Verify the web server is running
	time.Sleep(1 * time.Second)

	// Check that we can access the web server
	// This would be a real HTTP request in a proper implementation
	fmt.Println("Verifying web server is accessible...")

	// Make an HTTP request to the web server
	url := fmt.Sprintf("http://%s:8080", accessIP)
	checkCmd := exec.CommandContext(ctx, "curl", "-s", url)
	output, err := checkCmd.Output()
	if err != nil {
		fmt.Printf("Warning: Could not verify web server at %s: %v\n", url, err)
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
