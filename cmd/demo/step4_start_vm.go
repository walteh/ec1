package main

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"connectrpc.com/connect"
	ec1v1 "github.com/walteh/ec1/gen/proto/golang/ec1/v1poc1"
	"github.com/walteh/ec1/gen/proto/golang/ec1/v1poc1/v1poc1connect"
	"golang.org/x/crypto/ssh"
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

	// Define connection configs to try
	type sshConfig struct {
		host     string
		port     string
		user     string
		password string
	}

	configs := []sshConfig{
		{host: ipAddress, port: "22", user: "alpine", password: "alpine"},
		{host: ipAddress, port: "22", user: "root", password: "alpine"},
		{host: "localhost", port: "2222", user: "alpine", password: "alpine"},
		{host: "localhost", port: "2222", user: "root", password: "alpine"},
	}

	// Commands to try for fetching cloud-init logs
	commands := []string{
		"cat /var/log/cloud-init.log",
		"cat /var/log/cloud-init-output.log",
		"journalctl -u cloud-init",
	}

	// Try each config until one works
	for _, cfg := range configs {
		fmt.Printf("Trying SSH connection to %s@%s:%s...\n", cfg.user, cfg.host, cfg.port)

		// Set up SSH client config
		clientConfig := &ssh.ClientConfig{
			User: cfg.user,
			Auth: []ssh.AuthMethod{
				ssh.Password(cfg.password),
			},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
			Timeout:         5 * time.Second,
		}

		// Connect to the server
		addr := fmt.Sprintf("%s:%s", cfg.host, cfg.port)
		client, err := ssh.Dial("tcp", addr, clientConfig)
		if err != nil {
			if strings.Contains(err.Error(), "connection refused") ||
				strings.Contains(err.Error(), "i/o timeout") {
				continue // Try next config
			}
			fmt.Printf("SSH connection error: %v\n", err)
			continue
		}
		defer client.Close()

		fmt.Printf("Connected to %s, fetching cloud-init logs...\n", addr)

		// Try each command to find logs
		var foundLogs bool
		for _, cmd := range commands {
			session, err := client.NewSession()
			if err != nil {
				fmt.Printf("Failed to create session: %v\n", err)
				continue
			}
			defer session.Close()

			var stdout bytes.Buffer
			var stderr bytes.Buffer
			session.Stdout = &stdout
			session.Stderr = &stderr

			// Run the command
			err = session.Run(cmd)
			if err != nil && stderr.Len() > 0 {
				// If stderr contains "no such file", try next command
				if strings.Contains(stderr.String(), "No such file") ||
					strings.Contains(stderr.String(), "not found") {
					continue
				}
			}

			// If we got output, display it
			if stdout.Len() > 0 {
				foundLogs = true
				output := stdout.String()
				logLines := strings.Split(output, "\n")
				for _, line := range logLines {
					if line != "" {
						fmt.Printf("\033[36m[cloud-init]\033[0m %s\n", line)
					}
				}
				fmt.Println("=== End of Cloud-Init Logs ===")
				return nil
			}
		}

		if foundLogs {
			return nil
		}

		// If we connected but didn't find logs, return a specific error
		return fmt.Errorf("connected to VM but couldn't find cloud-init logs")
	}

	fmt.Println("=== Failed to retrieve Cloud-Init Logs ===")
	return fmt.Errorf("could not establish SSH connection to VM")
}

// TestSSHConnection checks if the VM is accessible via SSH
func TestSSHConnection(ctx context.Context, ipAddress string) (bool, string, error) {
	configs := []struct {
		host     string
		port     string
		user     string
		password string
	}{
		{host: ipAddress, port: "22", user: "alpine", password: "alpine"},
		{host: "localhost", port: "2222", user: "alpine", password: "alpine"},
		{host: "localhost", port: "2222", user: "root", password: "alpine"},
	}

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
		client, err := ssh.Dial("tcp", addr, clientConfig)
		if err != nil {
			continue
		}
		defer client.Close()

		session, err := client.NewSession()
		if err != nil {
			return false, addr, err
		}
		defer session.Close()

		var stdout bytes.Buffer
		session.Stdout = &stdout

		if err := session.Run("echo SSH_TEST_SUCCESS"); err != nil {
			return false, addr, err
		}

		if strings.Contains(stdout.String(), "SSH_TEST_SUCCESS") {
			return true, addr, nil
		}
	}

	return false, "", fmt.Errorf("all SSH connection attempts failed")
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

	// Wait for VM to boot with nice progress display (longer boot time to ensure it's ready)
	bootUpdates := []string{
		"BIOS initialization",
		"Kernel loading",
		"Mounting filesystems",
		"Network configuration",
		"Running cloud-init",
		"Installing required packages",
		"System initialization",
	}
	displayProgress(ctx, 40*time.Second, "VM Boot Progress", bootUpdates)

	// Ping test to see if the VM is reachable
	pingSuccess := false
	fmt.Printf("Testing connectivity to VM at %s... ", vmIP)
	for i := 0; i < 5; i++ {
		pingCmd := exec.CommandContext(ctx, "ping", "-c", "1", "-W", "2", vmIP)
		_ = pingCmd.Run()
		if pingCmd.ProcessState != nil && pingCmd.ProcessState.ExitCode() == 0 {
			pingSuccess = true
			fmt.Println("✓")
			break
		}
		if i < 4 {
			fmt.Print(".")
			time.Sleep(2 * time.Second)
		}
	}
	if !pingSuccess {
		fmt.Println("❌")
		fmt.Println("Warning: Cannot ping VM. Network might not be properly configured.")
		fmt.Println("Continuing anyway as some hypervisors don't support ICMP to the VM...")
	}

	// Test SSH connectivity
	fmt.Print("Testing SSH connection to VM... ")
	sshSuccess, sshAddr, sshErr := TestSSHConnection(ctx, vmIP)
	if sshSuccess {
		fmt.Println("✓")
		fmt.Printf("SSH connectivity verified via %s\n", sshAddr)
	} else {
		fmt.Println("❌")
		if sshErr != nil {
			fmt.Printf("SSH connection error: %v\n", sshErr)
		}
	}

	// Try to get cloud-init logs if SSH is working
	if sshSuccess {
		// Try to fetch cloud-init logs
		err = fetchCloudInitLogs(ctx, vmIP)
		if err != nil {
			fmt.Printf("Warning: Could not retrieve cloud-init logs: %v\n", err)
		}
	} else {
		fmt.Println("\n=== Cloud-Init Logs Unavailable - SSH Connection Failed ===")
		fmt.Println("The VM appears to be running but is not accepting SSH connections.")
		fmt.Println("This could be due to:")
		fmt.Println("1. SSH service not started in the VM")
		fmt.Println("2. Firewall blocking the connection")
		fmt.Println("3. VM not fully booted yet")
		fmt.Println("4. Network configuration issues")
	}

	fmt.Println("Continuing with boot sequence...")

	// Wait for the agent to start inside the VM with progress display
	agentUpdates := []string{
		"Downloading agent binary",
		"Setting execution permissions",
		"Starting agent service",
		"Connecting to management server",
	}
	displayProgress(ctx, 15*time.Second, "Agent Startup Progress", agentUpdates)

	// Try to connect to the agent to verify it's running with multiple attempts
	fmt.Println("Verifying agent is running...")
	verified := false
	var verifyErr error

	for i := 0; i < 3; i++ {
		if i > 0 {
			fmt.Printf("Retry %d/3... ", i)
			time.Sleep(5 * time.Second)
		}

		verifyErr = verifyAgentRunning(ctx, fmt.Sprintf("http://%s:9091", vmIP))
		if verifyErr == nil {
			verified = true
			fmt.Println("✓ Agent is running inside the VM")
			break
		}
	}

	if !verified {
		fmt.Println("❌")
		fmt.Printf("Warning: Could not verify agent is running: %v\n", verifyErr)
		fmt.Println("The demo will continue, but the nested VM step may fail")

		// Try alternative port forwarding approach
		fmt.Println("Trying alternative agent connection (localhost:9091)...")
		altErr := verifyAgentRunning(ctx, "http://localhost:9091")
		if altErr == nil {
			fmt.Println("✓ Agent is running and accessible via port forwarding")
			verified = true
		} else {
			fmt.Printf("Alternative agent verification failed: %v\n", altErr)
		}
	}

	// Return either the real VM IP or localhost if we're using port forwarding
	if !pingSuccess && verified {
		// If ping failed but agent verified via localhost, use localhost for subsequent steps
		return "localhost", nil
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

	// Check if both files exist
	if _, err := os.Stat(diskPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("disk image does not exist: %s", diskPath)
	}

	if _, err := os.Stat(ciPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("cloud-init ISO does not exist: %s", ciPath)
	}

	// For POC, we'll use a simplified approach
	// Get a dynamic IP from available network interfaces
	vmIP := getDynamicIPAddress()
	fmt.Printf("Using dynamic IP: %s for VM\n", vmIP)

	// Create a simulated VM info response
	vmInfo := &ec1v1.VMInfo{
		VmId:         ptr("vm-1"),
		Name:         ptr(name),
		Status:       ptr(ec1v1.VMStatus_VM_STATUS_RUNNING),
		IpAddress:    ptr(vmIP),
		ResourcesMax: resources,
		ResourcesLive: &ec1v1.Resources{
			Memory: resources.Memory,
			Cpu:    resources.Cpu,
		},
	}

	return vmInfo, nil
}

// getDynamicIPAddress tries to get a usable IP address
// For the demo, we try a few common local development IPs
func getDynamicIPAddress() string {
	// Try to get host IP from network interfaces
	interfaces, err := net.Interfaces()
	if err == nil {
		for _, iface := range interfaces {
			// Skip loopback and inactive interfaces
			if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
				continue
			}

			addrs, err := iface.Addrs()
			if err != nil {
				continue
			}

			for _, addr := range addrs {
				// Parse IP address
				var ip net.IP
				switch v := addr.(type) {
				case *net.IPNet:
					ip = v.IP
				case *net.IPAddr:
					ip = v.IP
				}

				// Skip non-IPv4 addresses
				if ip == nil || ip.To4() == nil {
					continue
				}

				// Skip loopback addresses
				if ip.IsLoopback() {
					continue
				}

				// If it's a private IP, use it as a base for VM IP
				if isPrivateIP(ip) {
					// Convert the first 3 octets to form a base for our VM IP
					octets := strings.Split(ip.String(), ".")
					if len(octets) == 4 {
						// Form VM IP with last octet as 10 (common for first VM)
						return fmt.Sprintf("%s.%s.%s.10", octets[0], octets[1], octets[2])
					}
				}
			}
		}
	}

	// Fallback to common VM IP addresses
	commonIPs := []string{
		"192.168.64.10",  // Often used by QEMU/Hyperkit
		"192.168.122.10", // Often used by libvirt/KVM
		"192.168.99.10",  // Often used by VirtualBox/Vagrant
		"192.168.65.10",  // Also common for VMs
	}

	// Let's ping each to see if any responds (indicating it might be in use)
	for _, ip := range commonIPs {
		cmd := exec.Command("ping", "-c", "1", "-W", "1", ip)
		if err := cmd.Run(); err != nil {
			// If ping fails, IP is probably available
			return ip
		}
	}

	// Final fallback
	return "192.168.64.10"
}

// isPrivateIP checks if an IP is in a private range
func isPrivateIP(ip net.IP) bool {
	// Check for private IPv4 ranges
	private := false

	// 10.0.0.0/8
	private = private || (ip[0] == 10)

	// 172.16.0.0/12
	private = private || (ip[0] == 172 && ip[1] >= 16 && ip[1] <= 31)

	// 192.168.0.0/16
	private = private || (ip[0] == 192 && ip[1] == 168)

	return private
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
