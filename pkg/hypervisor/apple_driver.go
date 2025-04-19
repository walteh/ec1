package hypervisor

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/Code-Hex/vz/v3"
	ec1v1 "github.com/walteh/ec1/gen/proto/golang/ec1/v1poc1"
)

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
	if !vz.IsNestedVirtualizationSupported() {
		return nil, fmt.Errorf("virtualization framework is not supported on this Mac")
	}

	// In a real implementation, we would also check for hardware virtualization support
	// if !vz.IsHardwareVirtualizationSupported() {
	// 	return nil, fmt.Errorf("hardware virtualization is not supported on this Mac")
	// }

	return &AppleDriver{
		vms: make(map[string]*appleVM),
	}, nil
}

func ptr[T any](v T) *T { return &v }

func arr[T any](v ...T) []T { return v }

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
	// Generate a simple VM ID
	vmID := fmt.Sprintf("mac-vm-%s", req.GetName())

	// Parse memory and CPU requirements
	memoryBytes, err := parseMemory(req.ResourcesMax.GetMemory())
	if err != nil {
		return nil, fmt.Errorf("invalid memory specification: %w", err)
	}

	cpuCount, err := parseCPU(req.ResourcesMax.GetCpu())
	if err != nil {
		return nil, fmt.Errorf("invalid CPU specification: %w", err)
	}

	loader, err := vz.NewEFIBootLoader()
	if err != nil {
		return nil, fmt.Errorf("creating EFI boot loader: %w", err)
	}

	// Create a configuration for the virtual machine
	config, err := vz.NewVirtualMachineConfiguration(loader, uint(cpuCount), memoryBytes)

	// Set up storage
	diskImagePath := req.GetDiskImagePath()
	diskExists := false
	if _, err := os.Stat(diskImagePath); err == nil {
		diskExists = true
	}

	if !diskExists {
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
	spAttachment, err := vz.NewFileHandleSerialPortAttachment(os.Stdout, nil)
	if err != nil {
		return nil, fmt.Errorf("creating serial port attachment: %w", err)
	}

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

	// // Set a boot loader for EFI boot
	// bootLoader, err := vz.NewEFIBootLoader()
	// if err != nil {
	// 	return nil, fmt.Errorf("creating EFI boot loader: %w", err)
	// }
	// config.SetBootLoader(bootLoader)

	// Validate the configuration
	if _, err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid VM configuration: %w", err)
	}

	// Create the virtual machine
	vm, err := vz.NewVirtualMachine(config)
	if err != nil {
		return nil, fmt.Errorf("creating virtual machine: %w", err)
	}

	// Create a new VM instance record
	appleVm := &appleVM{
		id:        vmID,
		name:      req.GetName(),
		resources: req.ResourcesMax,
		diskPath:  req.GetDiskImagePath(),
		// status:     ec1v1.VMStatus_VM_STATUS_STARTING,
		networkCfg: req.NetworkConfig,
		vm:         vm,
	}

	vm.State()

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
		return nil, fmt.Errorf("starting virtual machine: %w", err)
	}

	// In a real implementation, we would need to wait until the VM is up
	// and obtain its IP address. For the POC, we'll simulate a successful
	// start and assign a fake IP with a brief wait
	time.Sleep(1 * time.Second)

	// For the POC, we'll assign a static IP
	appleVm.ipAddress = "192.168.64.10" // Simulated IP for the POC

	return &ec1v1.StartVMResponse{
		VmId:      ptr(vmID),
		IpAddress: ptr(appleVm.ipAddress),
		Status:    ptr(appleVm.State()),
	}, nil
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
			Status: ptr(ec1v1.VMStatus_VM_STATUS_UNSPECIFIED),
			Error:  ptr(fmt.Sprintf("VM with ID %s not found", req.GetVmId())),
		}, nil
	}

	// For a real implementation, we would query the VM state
	// But for POC, we'll use the cached state which is updated via delegate

	return &ec1v1.GetVMStatusResponse{
		Status:    ptr(vm.State()),
		IpAddress: ptr(vm.ipAddress),
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
