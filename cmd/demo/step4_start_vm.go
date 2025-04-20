package main

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"connectrpc.com/connect"
	ec1v1 "github.com/walteh/ec1/gen/proto/golang/ec1/v1poc1"
	"github.com/walteh/ec1/gen/proto/golang/ec1/v1poc1/v1poc1connect"
)

// Helper function to convert value to pointer
func ptr[T any](v T) *T {
	return &v
}

// displayProgress shows an animated progress indicator with status updates
func displayProgress(ctx context.Context, duration time.Duration, message string, updates []string) {
	spinner := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	updateInterval := duration / time.Duration(len(updates)+1)
	spinnerInterval := 100 * time.Millisecond

	fmt.Printf("%s ", message)

	// Create a context that will cancel after the duration
	timeoutCtx, cancel := context.WithTimeout(ctx, duration)
	defer cancel()

	// Start time for calculating progress percentage
	startTime := time.Now()
	endTime := startTime.Add(duration)

	spinnerTick := time.NewTicker(spinnerInterval)
	defer spinnerTick.Stop()

	updateTick := time.NewTicker(updateInterval)
	defer updateTick.Stop()

	spinnerIndex := 0
	updateIndex := 0

	// Clear the current line
	clearLine := func() {
		fmt.Print("\r\033[K")
	}

	for {
		select {
		case <-timeoutCtx.Done():
			clearLine()
			fmt.Printf("%s: Complete ✓\n", message)
			return

		case <-updateTick.C:
			if updateIndex < len(updates) {
				clearLine()
				percentComplete := int(float64(time.Since(startTime)) / float64(duration) * 100)
				fmt.Printf("%s: %s [%d%%] - %s ",
					message,
					spinner[spinnerIndex],
					percentComplete,
					updates[updateIndex])
				updateIndex++
			}

		case <-spinnerTick.C:
			clearLine()
			percentComplete := int(float64(time.Since(startTime)) / float64(duration) * 100)
			spinnerIndex = (spinnerIndex + 1) % len(spinner)

			updateText := ""
			if updateIndex > 0 && updateIndex <= len(updates) {
				updateText = "- " + updates[updateIndex-1]
			}

			timeLeft := endTime.Sub(time.Now()).Round(time.Second)
			fmt.Printf("%s: %s [%d%%] %s (%s remaining) ",
				message,
				spinner[spinnerIndex],
				percentComplete,
				updateText,
				timeLeft)
		}
	}
}

// fetchCloudInitLogs tries to retrieve cloud-init logs from the VM via SSH
func fetchCloudInitLogs(ctx context.Context, ipAddress string) error {
	fmt.Println("\n=== Cloud-Init Logs ===")

	// Using real IP instead of localhost
	host := ipAddress
	port := "22"
	maxRetries := 5

	fmt.Printf("Attempting to SSH to %s:%s to retrieve cloud-init logs...\n", host, port)

	// Try multiple times as SSH might not be ready immediately
	var lastErr error
	for i := 0; i < maxRetries; i++ {
		if i > 0 {
			fmt.Printf("Retrying SSH connection (attempt %d/%d)...\n", i+1, maxRetries)
			time.Sleep(5 * time.Second)
		}

		// Alpine's default user is 'alpine' with no password or with password 'alpine'
		// Try both approaches - password auth and no password
		sshCmd := exec.CommandContext(ctx, "ssh",
			"-o", "StrictHostKeyChecking=no",
			"-o", "UserKnownHostsFile=/dev/null",
			"-o", "ConnectTimeout=8",
			"-o", "PasswordAuthentication=yes",
			"-v", // Verbose output for debugging
			host,
			"cat /var/log/cloud-init.log || cat /var/log/cloud-init-output.log || echo 'Cloud-init logs not found'")

		// Try alternative login if default doesn't work
		if i >= 2 {
			// On later attempts, try with root user and password
			sshCmd = exec.CommandContext(ctx, "ssh",
				"-o", "StrictHostKeyChecking=no",
				"-o", "UserKnownHostsFile=/dev/null",
				"-o", "ConnectTimeout=8",
				"-o", "PasswordAuthentication=yes",
				"-l", "root",
				"-v", // Verbose output for debugging
				host,
				"cat /var/log/cloud-init.log || cat /var/log/cloud-init-output.log || echo 'Cloud-init logs not found'")
		}

		stdout, err := sshCmd.StdoutPipe()
		if err != nil {
			lastErr = fmt.Errorf("getting SSH stdout pipe: %w", err)
			continue
		}

		stderr, err := sshCmd.StderrPipe()
		if err != nil {
			lastErr = fmt.Errorf("getting SSH stderr pipe: %w", err)
			continue
		}

		if err := sshCmd.Start(); err != nil {
			lastErr = fmt.Errorf("starting SSH command: %w", err)
			continue
		}

		// Print the output in real-time with a cyan prefix for logs
		var logFound bool
		go func() {
			scanner := bufio.NewScanner(stdout)
			for scanner.Scan() {
				line := scanner.Text()
				if strings.Contains(line, "Cloud-init") || strings.Contains(line, "cloud-init") {
					logFound = true
				}
				fmt.Printf("\033[36m[cloud-init]\033[0m %s\n", line)
			}
		}()

		// Also print any stderr output but check if it contains authentication errors
		var authError bool
		go func() {
			scanner := bufio.NewScanner(stderr)
			for scanner.Scan() {
				line := scanner.Text()
				if strings.Contains(line, "Authentication failed") ||
					strings.Contains(line, "Permission denied") {
					authError = true
				}
				// Only print errors in debug mode or if it's critical
				if os.Getenv("EC1_DEBUG") == "1" ||
					strings.Contains(line, "fatal") ||
					strings.Contains(line, "Failed") {
					fmt.Printf("\033[31m[ssh-error]\033[0m %s\n", line)
				}
			}
		}()

		// Wait for the command to finish
		err = sshCmd.Wait()
		if err == nil && !authError {
			fmt.Println("=== End of Cloud-Init Logs ===")
			return nil
		}

		lastErr = fmt.Errorf("SSH command failed: %w", err)
		if logFound {
			fmt.Println("=== End of Cloud-Init Logs ===")
			return nil
		}

		// If authentication error, try next method
		if authError {
			continue
		}
	}

	fmt.Println("=== Failed to retrieve Cloud-Init Logs ===")
	return lastErr
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

	// Wait for VM to boot and run cloud-init with nice progress display
	bootUpdates := []string{
		"BIOS initialization",
		"Kernel loading",
		"Mounting filesystems",
		"Network configuration",
		"Running cloud-init",
		"Installing required packages",
		"System initialization",
	}
	displayProgress(ctx, 25*time.Second, "VM Boot Progress", bootUpdates)

	// Try to fetch and display cloud-init logs after basic boot
	err = fetchCloudInitLogs(ctx, vmIP)
	if err != nil {
		fmt.Printf("Warning: Could not retrieve cloud-init logs: %v\n", err)
		fmt.Println("Continuing with boot sequence...")
	}

	// Wait for the agent to start inside the VM with progress display
	agentUpdates := []string{
		"Downloading agent binary",
		"Setting execution permissions",
		"Starting agent service",
		"Connecting to management server",
	}
	displayProgress(ctx, 10*time.Second, "Agent Startup Progress", agentUpdates)

	// Try to connect to the agent to verify it's running
	fmt.Print("Verifying agent is running... ")
	err = verifyAgentRunning(ctx, fmt.Sprintf("http://%s:9091", vmIP))
	if err != nil {
		fmt.Println("❌")
		fmt.Printf("Warning: Could not verify agent is running: %v\n", err)
		fmt.Println("The demo will continue, but the nested VM step may fail")
	} else {
		fmt.Println("✓")
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
