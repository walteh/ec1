//go:build darwin
// +build darwin

package hypervisor

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/Code-Hex/vz/v3"
	slogctx "github.com/veqryn/slog-context"

	ec1v1 "github.com/walteh/ec1/gen/proto/golang/ec1/v1poc1"
	"github.com/walteh/ec1/sanbox/pkg/cloud/id"
)

func NewDriver(ctx context.Context) (Driver, error) {
	// Detect platform and return appropriate driver
	return NewAppleDriver(ctx)
}

// TODO: add Code-Hex/vz

// // Pseudo-code for Mac Agent using Code-Hex/vz to start a Linux VM
// func (a *AgentServer) StartVM(ctx context.Context, req *StartVMRequest) (*StartVMResponse, error) {
//     // Prepare VM configuration (memory, disk image path, network interface)
//     config, err := vz.NewVirtualMachineConfig(opts...)
//     if err != nil { return nil, err }
//     vm, err := vz.NewVirtualMachine(config)
//     if err != nil { return nil, err }
//     if err := vm.Start(); err != nil {
//         return nil, err  // return failure if VM couldn't start
//     }
//     return &StartVMResponse{Success: true, VmId: "linux-vm-1"}, nil
// }

// AppleDriver implements the Driver interface for macOS using Virtualization.framework
type AppleDriver struct {
	// Map to keep track of running VMs
	vms     map[string]*appleVM
	vmMutex sync.RWMutex
}

// appleVM represents a running VM instance
type appleVM struct {
	id        string
	ipAddress string
	// status     ec1v1.VMStatus
	name       string
	resources  *ec1v1.Resources
	diskPath   string
	portFwds   []*ec1v1.PortForward
	networkCfg *ec1v1.VMNetworkConfig

	// The actual virtualization.framework VM instance
	vm *vz.VirtualMachine
}

func (d *appleVM) State() ec1v1.VMStatus {
	switch d.vm.State() {
	case vz.VirtualMachineStateRunning:
		return ec1v1.VMStatus_VM_STATUS_RUNNING
	case vz.VirtualMachineStateStopped:
		return ec1v1.VMStatus_VM_STATUS_STOPPED
	case vz.VirtualMachineStatePaused:
		// Not directly mapped in our enum, could use STOPPED
		return ec1v1.VMStatus_VM_STATUS_STOPPED
	case vz.VirtualMachineStateError:
		return ec1v1.VMStatus_VM_STATUS_ERROR
	case vz.VirtualMachineStateStarting:
		return ec1v1.VMStatus_VM_STATUS_STARTING
	case vz.VirtualMachineStateStopping:
		return ec1v1.VMStatus_VM_STATUS_STOPPING
	case vz.VirtualMachineStatePausing:
		return ec1v1.VMStatus_VM_STATUS_STOPPING
	case vz.VirtualMachineStateRestoring:
		return ec1v1.VMStatus_VM_STATUS_STOPPING
	case vz.VirtualMachineStateSaving:
		return ec1v1.VMStatus_VM_STATUS_STOPPING
	case vz.VirtualMachineStateResuming:
		return ec1v1.VMStatus_VM_STATUS_STOPPING
	default:
		return ec1v1.VMStatus_VM_STATUS_UNSPECIFIED
	}
}

// NewAppleDriver creates a new driver for macOS virtualization
func NewAppleDriver(ctx context.Context) (*AppleDriver, error) {
	// Check if Virtualization.framework is supported
	// if !vz.IsNestedVirtualizationSupported() {
	// 	return nil, fmt.Errorf("virtualization framework is not supported on this Mac")
	// }

	// In a real implementation, we would also check for hardware virtualization support
	// if !vz.IsHardwareVirtualizationSupported() {
	// 	return nil, fmt.Errorf("hardware virtualization is not supported on this Mac")
	// }

	return &AppleDriver{
		vms: make(map[string]*appleVM),
	}, nil
}

// Helper function to parse memory string (like "4Gi") to bytes
func parseMemory(memStr string) (uint64, error) {
	// Basic implementation - in a real driver, we'd properly parse sizes like "4Gi", "512Mi", etc.
	// For the POC, we'll assume basic values in GiB
	var memoryBytes uint64 = 4 * 1024 * 1024 * 1024 // Default 4 GiB
	return memoryBytes, nil
}

// Helper function to parse CPU count
func parseCPU(cpuStr string) (uint64, error) {
	// Basic implementation - in a real driver, we'd properly parse CPU values
	var cpuCount uint64 = 2 // Default 2 cores
	return cpuCount, nil
}

// StartVM implements the Driver interface
func (d *AppleDriver) StartVM(ctx context.Context, req *ec1v1.StartVMRequest) (*ec1v1.StartVMResponse, error) {

	id := id.NewID("vm")
	// Generate a simple VM ID
	vmID := id.String()

	// Parse memory and CPU requirements
	memoryBytes, err := parseMemory(req.ResourcesMax.GetMemory())
	if err != nil {
		return nil, fmt.Errorf("invalid memory specification: %w", err)
	}

	cpuCount, err := parseCPU(req.ResourcesMax.GetCpu())
	if err != nil {
		return nil, fmt.Errorf("invalid CPU specification: %w", err)
	}

	// Create the bin directory if it doesn't exist
	if err := os.MkdirAll("./bin", 0755); err != nil {
		slogctx.Error(ctx, "failed to create bin directory for EFI variable store", "error", err)
		return nil, fmt.Errorf("creating bin directory: %w", err)
	}

	variableStore, err := vz.NewEFIVariableStore("./bin/vz-vs.fd", vz.WithCreatingEFIVariableStore())
	if err != nil {
		// Check if EFI directory exists
		if _, statErr := os.Stat("./bin"); statErr != nil {
			slogctx.Error(ctx, "bin directory for EFI variable store doesn't exist", "error", statErr)
		}
		// Check if fd file exists or failed to be created
		if _, statErr := os.Stat("./bin/vz-vs.fd"); statErr != nil {
			slogctx.Error(ctx, "EFI variable store file not accessible", "error", statErr)
		}
		return nil, fmt.Errorf("creating variable store: %w", err)
	}
	slogctx.Info(ctx, "created EFI variable store", "path", "./bin/vz-vs.fd")

	loader, err := vz.NewEFIBootLoader(vz.WithEFIVariableStore(variableStore))
	if err != nil {
		slogctx.Error(ctx, "failed to create EFI boot loader", "error", err)
		return nil, fmt.Errorf("creating EFI boot loader: %w", err)
	}

	// Create a configuration for the virtual machine
	config, err := vz.NewVirtualMachineConfiguration(loader, uint(cpuCount), memoryBytes)
	if err != nil {
		slogctx.Error(ctx, "failed to create VM configuration", "error", err)
		return nil, fmt.Errorf("creating virtual machine configuration: %w", err)
	}

	// Set up storage
	diskImagePath := strings.TrimPrefix(req.GetDiskImage().GetPath(), "file://")
	diskExists := false
	if _, err := os.Stat(diskImagePath); err == nil {
		diskExists = true
		// Check for readable permissions
		file, openErr := os.Open(diskImagePath)
		if openErr != nil {
			slogctx.Error(ctx, "disk image not readable", "path", diskImagePath, "error", openErr)
		} else {
			// Check QCOW2 file format
			header := make([]byte, 4)
			if _, err := file.Read(header); err != nil {
				slogctx.Error(ctx, "cannot read disk image header", "path", diskImagePath, "error", err)
			} else {
				// QCOW2 files start with 'QFI\xfb'
				slogctx.Info(ctx, "disk image header", "header_bytes", fmt.Sprintf("%x", header))

				if !bytes.Equal(header, []byte{'Q', 'F', 'I', 0xfb}) {
					slogctx.Error(ctx, "disk image does not appear to be a valid QCOW2 file",
						"path", diskImagePath,
						"header_bytes", fmt.Sprintf("%x", header))
				} else {
					slogctx.Info(ctx, "disk image appears to be a valid QCOW2 file", "path", diskImagePath)
				}
			}
			file.Close()
			slogctx.Info(ctx, "disk image exists and is readable", "path", diskImagePath)
		}
	}

	if !diskExists {
		slogctx.Error(ctx, "disk image not found at path", "path", diskImagePath)
		return nil, fmt.Errorf("disk image not found at path: %s", diskImagePath)
	}

	// Create a disk device from the specified disk image
	diskAttachment, err := vz.NewDiskImageStorageDeviceAttachment(
		diskImagePath,
		false, // isReadOnly
	)
	if err != nil {
		return nil, fmt.Errorf("creating disk attachment: %w", err)
	}

	// Create a virtio block device
	blockDevice, err := vz.NewVirtioBlockDeviceConfiguration(diskAttachment)
	if err != nil {
		return nil, fmt.Errorf("creating block device: %w", err)
	}

	config.SetStorageDevicesVirtualMachineConfiguration(arr[vz.StorageDeviceConfiguration](blockDevice))

	// Set up network
	natAttachment, err := vz.NewNATNetworkDeviceAttachment()
	if err != nil {
		return nil, fmt.Errorf("creating NAT network attachment: %w", err)
	}

	// Create a virtio network device
	networkDevice, err := vz.NewVirtioNetworkDeviceConfiguration(natAttachment)
	if err != nil {
		return nil, fmt.Errorf("creating network device: %w", err)
	}
	config.SetNetworkDevicesVirtualMachineConfiguration(arr(networkDevice))

	// Create and set up a serial port
	// To fix the crash, we need to provide both read and write file handles
	readPipe, writePipe, err := os.Pipe()
	if err != nil {
		slogctx.Error(ctx, "failed to create pipes for serial console", "error", err)
		return nil, fmt.Errorf("creating pipes for serial console: %w", err)
	}
	slogctx.Info(ctx, "created pipes for VM serial console")

	// Use the pipes for the serial port attachment
	slogctx.Info(ctx, "creating serial port attachment")
	spAttachment, err := vz.NewFileHandleSerialPortAttachment(writePipe, readPipe)
	if err != nil {
		readPipe.Close()
		writePipe.Close()
		slogctx.Error(ctx, "failed to create serial port attachment", "error", err)
		return nil, fmt.Errorf("creating serial port attachment: %w", err)
	}
	slogctx.Info(ctx, "serial port attachment created successfully")

	// Start a goroutine to forward console output to stdout
	go func() {
		defer readPipe.Close()
		defer writePipe.Close()
		slogctx.Info(ctx, "started VM console reader")

		// Buffer for reading console output
		buf := make([]byte, 1024)
		bytesRead := 0
		outputCollection := ""
		lastLogTime := time.Now()

		for {
			n, err := readPipe.Read(buf)
			if err != nil {
				if err != io.EOF {
					slogctx.Error(ctx, "error reading from VM console", "error", err)
				}
				slogctx.Info(ctx, "VM console reader stopped", "total_bytes_read", bytesRead, "last_output", outputCollection)
				break
			}

			bytesRead += n
			// Log console output through structured logger
			output := string(buf[:n])
			outputCollection += output

			// Log collected output every second or when we have a reasonable amount
			if len(outputCollection) > 100 || time.Since(lastLogTime) > time.Second {
				slogctx.Info(ctx, "VM console output", "output", outputCollection, "bytes", len(outputCollection))
				outputCollection = ""
				lastLogTime = time.Now()
			}
		}
	}()

	macAddr, err := vz.NewRandomLocallyAdministeredMACAddress()
	if err != nil {
		return nil, fmt.Errorf("making MAC address: %w", err)
	}
	networkDevice.SetMACAddress(macAddr)

	// Create serial port configuration with the attachment
	serialPort, err := vz.NewVirtioConsolePortConfiguration(vz.WithVirtioConsolePortConfigurationAttachment(spAttachment))
	if err != nil {
		return nil, fmt.Errorf("creating serial port: %w", err)
	}

	consoleDevice, err := vz.NewVirtioConsoleDeviceConfiguration()
	if err != nil {
		return nil, fmt.Errorf("creating console device: %w", err)
	}

	consoleDevice.SetVirtioConsolePortConfiguration(0, serialPort)

	config.SetConsoleDevicesVirtualMachineConfiguration(arr[vz.ConsoleDeviceConfiguration](consoleDevice))

	// Validate the configuration
	if valid, err := config.Validate(); err != nil {
		slogctx.Error(ctx, "invalid VM configuration", "error", err, "valid", valid)
		return nil, fmt.Errorf("invalid VM configuration: %w", err)
	} else {
		slogctx.Info(ctx, "VM configuration validated successfully")
	}

	// Create the virtual machine
	vm, err := vz.NewVirtualMachine(config)
	if err != nil {
		return nil, fmt.Errorf("creating virtual machine: %w", err)
	}

	// Create a new VM instance record
	appleVm := &appleVM{
		id:         vmID,
		name:       req.GetName(),
		resources:  req.ResourcesMax,
		diskPath:   req.GetDiskImage().GetPath(),
		portFwds:   req.GetNetworkConfig().GetPortForwards(),
		networkCfg: req.NetworkConfig,
		vm:         vm,
	}

	// Log important VM configuration parameters
	slogctx.Info(ctx, "vm configuration details",
		"vm", appleVm.id,
		"disk_path", req.GetDiskImage().GetPath(),
		"memory_size", memoryBytes,
		"cpu_count", cpuCount)

	// Watch VM state change notifications in a goroutine
	go func() {
		stateCh := appleVm.vm.StateChangedNotify()
		var lastState vz.VirtualMachineState
		for {
			select {
			case <-ctx.Done():
				slogctx.Info(ctx, "stopping vm state watcher", "vm", appleVm.id)
				return
			case state, ok := <-stateCh:
				if !ok {
					slogctx.Warn(ctx, "vm state channel closed", "vm", appleVm.id)
					return
				}
				// Track specific state transitions for debugging
				slogctx.Info(ctx, "vm state changed",
					"vm", appleVm.id,
					"state", state,
					"previous_state", lastState)

				// Watch specifically for the quick RUNNING to STOPPED transition
				if lastState == vz.VirtualMachineStateRunning && state == vz.VirtualMachineStateStopped {
					slogctx.Error(ctx, "vm stopped immediately after running - likely boot failure",
						"vm", appleVm.id,
						"disk_path", appleVm.diskPath)

					// Try to identify specific causes for VM shutdown
					fileInfo, err := os.Stat(diskImagePath)
					if err != nil {
						slogctx.Error(ctx, "vm disk image not accessible", "vm", appleVm.id, "path", diskImagePath, "error", err)
					} else {
						slogctx.Info(ctx, "vm disk image details",
							"vm", appleVm.id,
							"path", diskImagePath,
							"size_bytes", fileInfo.Size(),
							"size_mb", fileInfo.Size()/(1024*1024),
							"permissions", fileInfo.Mode().String())

						// Check if disk image is suspiciously small (possible corruption)
						if fileInfo.Size() < 5*1024*1024 { // Less than 5MB
							slogctx.Error(ctx, "vm disk image suspiciously small - likely corrupted",
								"vm", appleVm.id,
								"path", diskImagePath,
								"size_bytes", fileInfo.Size())
						}
					}

					// Check for specific VM error if possible
					if appleVm.vm != nil {
						slogctx.Error(ctx, "vm error state details",
							"vm", appleVm.id,
							"can_request_stop", appleVm.vm.CanRequestStop(),
							"can_start", appleVm.vm.CanStart())
					}

					// Look for EFI variable store and validate its size/permissions
					if fileInfo, err := os.Stat("./bin/vz-vs.fd"); err != nil {
						slogctx.Error(ctx, "efi variable store not accessible", "vm", appleVm.id, "path", "./bin/vz-vs.fd", "error", err)
					} else {
						slogctx.Info(ctx, "efi variable store details",
							"vm", appleVm.id,
							"path", "./bin/vz-vs.fd",
							"size_bytes", fileInfo.Size(),
							"permissions", fileInfo.Mode().String())
					}
				}

				lastState = state
				if state == vz.VirtualMachineStateError || state == vz.VirtualMachineStateStopped {
					slogctx.Info(ctx, "vm reached terminal state, stopping watcher", "vm", appleVm.id, "state", state)
					return
				}
			}
		}
	}()

	// Store the VM in our map
	d.vmMutex.Lock()
	d.vms[vmID] = appleVm
	d.vmMutex.Unlock()

	// Start the VM
	err = vm.Start()
	if err != nil {
		d.vmMutex.Lock()
		// appleVm.status = ec1v1.VMStatus_VM_STATUS_ERROR
		d.vmMutex.Unlock()
		slogctx.Error(ctx, "failed to start vm", "vm", appleVm.id, "error", err)
		return nil, fmt.Errorf("starting virtual machine: %w", err)
	}

	// Get the VM configuration (display all settings for debugging)
	slogctx.Info(ctx, "vm start successful, configuration details",
		"vm", appleVm.id,
		"efi_store", "./bin/vz-vs.fd",
		"disk_path", diskImagePath)

	slogctx.Info(ctx, "vm started - logs should start soon", "vm", appleVm.id)

	// In a real implementation, we would need to wait until the VM is up
	// and obtain its IP address. For the POC, we'll simulate a successful
	// start and assign a fake IP with a brief wait
	time.Sleep(1 * time.Second)

	ip, err := waitForCallback(ctx, 12345)
	if err != nil {
		return nil, fmt.Errorf("waiting for callback: %w", err)
		// handle error
	}
	// now you have the guest's IP
	fmt.Printf("VM is up at %s\n", ip)

	// HERE: need the ip address of the vm

	return &ec1v1.StartVMResponse{
		VmId:      ptr(vmID),
		IpAddress: ptr(ip),
		Status:    ptr(appleVm.State()),
	}, nil
}

// waitForCallback blocks until the VM calls back with its IP, or times out.
func waitForCallback(ctx context.Context, port int) (string, error) {
	srv := &http.Server{Addr: fmt.Sprintf(":%d", port)}
	ipCh := make(chan string, 1)

	http.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		// e.g. guest does: curl http://$GATEWAY:12345/ready?ip=$(hostname -I)
		ip := r.URL.Query().Get("ip")
		if ip != "" {
			ipCh <- ip
		}
	})
	go srv.ListenAndServe() // ignore err for brevity

	select {
	case ip := <-ipCh:
		srv.Shutdown(ctx)
		return ip, nil
	case <-time.After(30 * time.Second):
		srv.Shutdown(ctx)
		return "", fmt.Errorf("timed out waiting for VM callback")
	case <-ctx.Done():
		srv.Shutdown(ctx)
		return "", ctx.Err()
	}
}

// getIPFromARP looks through the host's ARP table for the given MAC and returns its IP.
func getIPFromARP(mac string) (string, error) {
	// run `arp -an`
	out, err := exec.Command("arp", "-an").Output()
	if err != nil {
		return "", fmt.Errorf("failed to exec arp: %w", err)
	}

	// lines look like:
	// ? (172.16.2.2) at de:ad:be:ef:ca:fe on vmnet1 ifscope [ethernet]
	re := regexp.MustCompile(`$begin:math:text$(?P<ip>[\\d\\.]+)$end:math:text$\s+at\s+(?P<mac>[0-9a-f:]+)`)
	for _, line := range bytes.Split(out, []byte("\n")) {
		m := re.FindSubmatch(line)
		if len(m) == 3 && strings.EqualFold(string(m[2]), mac) {
			return string(m[1]), nil
		}
	}
	return "", fmt.Errorf("no ARP entry for MAC %s", mac)
}

// StopVM implements the Driver interface
func (d *AppleDriver) StopVM(ctx context.Context, req *ec1v1.StopVMRequest) (*ec1v1.StopVMResponse, error) {
	d.vmMutex.RLock()
	vm, exists := d.vms[req.GetVmId()]
	d.vmMutex.RUnlock()

	if !exists {
		return &ec1v1.StopVMResponse{
			Success: ptr(false),
			Error:   ptr(fmt.Sprintf("VM with ID %s not found", req.GetVmId())),
		}, nil
	}

	// // Mark the VM as stopping
	// d.vmMutex.Lock()
	// vm.status = ec1v1.VMStatus_VM_STATUS_STOPPING
	// d.vmMutex.Unlock()

	// Use the real VM to stop it
	var err error
	if req.GetForce() {
		// Force stop the VM
		err = vm.vm.Stop()
	} else {
		if !vm.vm.CanRequestStop() {
			return &ec1v1.StopVMResponse{
				Success: ptr(false),
				Error:   ptr(fmt.Sprintf("VM with ID %s cannot be stopped", req.GetVmId())),
			}, nil
		}
		// Request a graceful shutdown
		ok, err := vm.vm.RequestStop()
		if err != nil {
			return &ec1v1.StopVMResponse{
				Success: ptr(false),
				Error:   ptr(fmt.Sprintf("Failed to stop VM: %v", err)),
			}, nil
		}

		if !ok {
			return &ec1v1.StopVMResponse{
				Success: ptr(false),
				Error:   ptr(fmt.Sprintf("VM with ID %s cannot be stopped", req.GetVmId())),
			}, nil
		}
	}

	if err != nil {
		// d.vmMutex.Lock()
		// vm.status = ec1v1.VMStatus_VM_STATUS_ERROR
		// d.vmMutex.Unlock()
		return &ec1v1.StopVMResponse{
			Success: ptr(false),
			Error:   ptr(fmt.Sprintf("Failed to stop VM: %v", err)),
		}, nil
	}

	// For the POC, we'll simulate immediate completion
	// In a real implementation, we might wait for the state change callback
	// d.vmMutex.Lock()
	// vm.status = ec1v1.VMStatus_VM_STATUS_STOPPED
	// d.vmMutex.Unlock()

	return &ec1v1.StopVMResponse{
		Success: ptr(true),
	}, nil
}

// GetVMStatus implements the Driver interface
func (d *AppleDriver) GetVMStatus(ctx context.Context, req *ec1v1.GetVMStatusRequest) (*ec1v1.GetVMStatusResponse, error) {
	d.vmMutex.RLock()
	vm, exists := d.vms[req.GetVmId()]
	d.vmMutex.RUnlock()

	if !exists {
		return &ec1v1.GetVMStatusResponse{
			Response: &ec1v1.VMStatusResponse{
				Status: ptr(ec1v1.VMStatus_VM_STATUS_UNSPECIFIED),
				Error:  ptr(fmt.Sprintf("VM with ID %s not found", req.GetVmId())),
			},
		}, nil
	}

	// For a real implementation, we would query the VM state
	// But for POC, we'll use the cached state which is updated via delegate

	return &ec1v1.GetVMStatusResponse{
		Response: &ec1v1.VMStatusResponse{
			Status:    ptr(vm.State()),
			IpAddress: ptr(vm.ipAddress),
		},
	}, nil
}

// GetHypervisorType implements the Driver interface
func (d *AppleDriver) GetHypervisorType() ec1v1.HypervisorType {
	return ec1v1.HypervisorType_HYPERVISOR_TYPE_MAC_VIRTUALIZATION
}

// vzDelegate is a helper type to handle VM state changes
type vzDelegate struct {
	onStateChanged func(vz.VirtualMachineState)
}

func (d *vzDelegate) VirtualMachineStateDidChange(vm *vz.VirtualMachine) {
	if d.onStateChanged != nil {
		d.onStateChanged(vm.State())
	}
}
