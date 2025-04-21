package main

import (
	"context"
	"fmt"
	"os"
	"time"

	ec1v1 "github.com/walteh/ec1/gen/proto/golang/ec1/v1poc1"
)

// Helper function to convert value to pointer

// StartLinuxVM starts the Linux VM via the management server
func StartLinuxVM(ctx context.Context, demoCtx *DemoContext, qcow2PathObj *Image) (string, error) {
	fmt.Println("Starting Linux VM via management server...")

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
			{
				HostPort:  ptr(int32(9091)),
				GuestPort: ptr(int32(9091)),
			},
		},
	}

	// Prepare resources configuration
	resources := &ec1v1.Resources{
		Memory: ptr("2Gi"),
		Cpu:    ptr("2"),
	}

	// Start the VM
	vmInfo, err := StartVM(ctx, demoCtx, "linux-vm", qcow2PathObj.Qcow2Path, qcow2PathObj.CloudInitSeedISOPath, resources, networkConfig)
	if err != nil {
		return "", fmt.Errorf("starting Linux VM: %w", err)
	}

	vmIP := vmInfo.GetIpAddress()
	fmt.Printf("Linux VM started successfully with IP: %s\n", vmIP)

	// Simple wait to allow VM to boot
	fmt.Println("Waiting for VM to boot...")
	time.Sleep(10 * time.Second)

	// Basic connectivity test
	fmt.Printf("Testing connectivity to VM at %s...\n", vmIP)
	connected := waitForConnection(ctx, vmIP, 30*time.Second)
	if !connected {
		fmt.Println("Warning: Could not establish connectivity to VM")
		fmt.Println("Using assigned IP address anyway")
	} else {
		fmt.Println("Successfully connected to VM")
	}

	return vmIP, nil
}

// StartVM is a helper function to start a VM through the agent
func StartVM(
	ctx context.Context,
	demoCtx *DemoContext,
	name, diskPath, ciPath string,
	resources *ec1v1.Resources,
	networkConfig *ec1v1.VMNetworkConfig,
) (*ec1v1.VMInfo, error) {
	fmt.Printf("Starting VM %s with disk %s...\n", name, diskPath)

	// Check if both files exist
	if _, err := os.Stat(diskPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("disk image does not exist: %s", diskPath)
	}

	if _, err := os.Stat(ciPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("cloud-init ISO does not exist: %s", ciPath)
	}

	// Create a unique VM ID
	vmID := "vm-" + name

	// Prepare the VM start request

	// Send the request to the agent
	fmt.Println("Sending StartVM request to agent...")
	resp, err := demoCtx.AgentClient.StartVM(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("starting VM: %w", err)
	}

	// Get the IP address from the response
	ipAddress := resp.Msg.GetIpAddress()
	if ipAddress == "" {
		// If the agent didn't return an IP, use a static IP for demo
		ipAddress = "192.168.64.10"
		fmt.Printf("No IP address returned from agent, using default: %s\n", ipAddress)
	}

	// Create a VM info object with the response data
	vmInfo := &ec1v1.VMInfo{
		VmId:         ptr(vmID),
		Name:         ptr(name),
		Status:       ptr(ec1v1.VMStatus_VM_STATUS_RUNNING),
		IpAddress:    ptr(ipAddress),
		ResourcesMax: resources,
	}

	return vmInfo, nil
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
