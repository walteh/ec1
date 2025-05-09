//go:build linux

package hypervisor

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"

	libvirt "github.com/digitalocean/go-libvirt"
	ec1v1 "github.com/walteh/ec1/gen/proto/golang/ec1/v1poc1"
)

// TODO: add KVM/QEMU driver

// import libvirt "libvirt.org/libvirt-go"
// ...
// conn, err := libvirt.NewConnect("qemu:///system")
// if err != nil { return error... }
// // Define domain XML for nested VM (or use an existing XML template)
// dom, err := conn.DomainCreateXML(nestedVmXML, 0)  // 0 = default flags
// if err != nil { ... }
// defer dom.Free()
// // If needed, get its IP or wait for boot...

// KVMDriver implements the Driver interface for Linux using KVM via libvirt
type KVMDriver struct {
	// Map to keep track of running VMs
	vms     map[string]*kvmVM
	vmMutex sync.RWMutex

	// Connection to libvirt
	l    *libvirt.Libvirt
	conn net.Conn
}

// kvmVM represents a running VM instance
type kvmVM struct {
	id         string
	ipAddress  string
	name       string
	resources  *ec1v1.Resources
	diskPath   string
	portFwds   []*ec1v1.PortForward
	networkCfg *ec1v1.VMNetworkConfig

	// Libvirt domain ID and name
	domain     *libvirt.Domain
	domainName string
}

func NewDriver(ctx context.Context) (Driver, error) {
	return NewKVMDriver(ctx)
}

// NewKVMDriver creates a new driver for KVM virtualization
func NewKVMDriver(ctx context.Context) (*KVMDriver, error) {
	// Connect to libvirt (QEMU system)
	conn, err := net.DialTimeout("unix", "/var/run/libvirt/libvirt-sock", 2*time.Second)
	if err != nil {
		return nil, fmt.Errorf("connecting to libvirt socket: %w", err)
	}

	l := libvirt.New(conn)
	if err := l.Connect(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("establishing libvirt connection: %w", err)
	}

	return &KVMDriver{
		vms:  make(map[string]*kvmVM),
		l:    l,
		conn: conn,
	}, nil
}

// StartVM implements the Driver interface
func (d *KVMDriver) StartVM(ctx context.Context, req *ec1v1.StartVMRequest) (*ec1v1.StartVMResponse, error) {
	// Generate a simple VM ID
	vmID := fmt.Sprintf("kvm-vm-%s", req.GetName())

	// Get memory and CPU specs
	memoryStr := req.ResourcesMax.GetMemory()
	cpuStr := req.ResourcesMax.GetCpu()

	// Parse memory - using a simple parsing for POC
	memoryMB := 1024 // Default 1GB
	if memoryStr != "" {
		// Simple parsing: assume format like "1024Mi" or "1Gi"
		mem, err := strconv.Atoi(memoryStr[:len(memoryStr)-2])
		if err == nil {
			// If memory is specified in Gi, convert to Mi
			if memoryStr[len(memoryStr)-2:] == "Gi" {
				memoryMB = mem * 1024
			} else {
				memoryMB = mem
			}
		}
	}

	// For simplicity, create a domain XML definition
	// In a real implementation, we would properly configure based on all parameters
	domainXML := fmt.Sprintf(`
	<domain type='kvm'>
		<name>%s</name>
		<memory unit='MiB'>%d</memory>
		<vcpu>%s</vcpu>
		<os>
			<type arch='x86_64'>hvm</type>
			<boot dev='hd'/>
		</os>
		<devices>
			<disk type='file' device='disk'>
				<driver name='qemu' type='qcow2'/>
				<source file='%s'/>
				<target dev='vda' bus='virtio'/>
			</disk>
			<interface type='network'>
				<source network='default'/>
				<model type='virtio'/>
			</interface>
			<console type='pty'/>
		</devices>
	</domain>
	`, vmID, memoryMB, cpuStr, req.GetDiskImage().GetPath())

	// Create the domain from XML
	domain, err := d.l.DomainDefineXML(domainXML)
	if err != nil {
		return nil, fmt.Errorf("defining domain from XML: %w", err)
	}

	// Create the VM record
	vm := &kvmVM{
		id:         vmID,
		name:       req.GetName(),
		resources:  req.ResourcesMax,
		diskPath:   req.GetDiskImage().GetPath(),
		portFwds:   req.GetNetworkConfig().GetPortForwards(),
		networkCfg: req.NetworkConfig,
		domain:     &domain,
		domainName: vmID,
	}

	// Store the VM
	d.vmMutex.Lock()
	d.vms[vmID] = vm
	d.vmMutex.Unlock()

	// Start the domain
	if err := d.l.DomainCreate(domain); err != nil {
		d.vmMutex.Lock()
		delete(d.vms, vmID)
		d.vmMutex.Unlock()

		// Try to clean up the domain
		_ = d.l.DomainUndefine(domain)

		return nil, fmt.Errorf("starting domain: %w", err)
	}

	// For port forwarding, we would configure network in a real implementation
	// In a real setup, we'd likely use libvirt network filtering or a separate tool

	// In a real implementation, we would get the IP by querying the domain
	// For the POC, we assign a fake IP
	vm.ipAddress = "192.168.122.10" // Simulated IP for the POC

	// Return the response
	return &ec1v1.StartVMResponse{
		VmId:      ptr(vmID),
		IpAddress: ptr(vm.ipAddress),
		Status:    ptr(ec1v1.VMStatus_VM_STATUS_RUNNING),
	}, nil
}

// StopVM implements the Driver interface
func (d *KVMDriver) StopVM(ctx context.Context, req *ec1v1.StopVMRequest) (*ec1v1.StopVMResponse, error) {
	d.vmMutex.RLock()
	vm, exists := d.vms[req.GetVmId()]
	d.vmMutex.RUnlock()

	if !exists {
		return &ec1v1.StopVMResponse{
			Success: ptr(false),
			Error:   ptr(fmt.Sprintf("VM with ID %s not found", req.GetVmId())),
		}, nil
	}

	var err error
	if req.GetForce() {
		// Force stop the VM
		err = d.l.DomainDestroy(*vm.domain)
	} else {
		// Request a graceful shutdown
		err = d.l.DomainShutdown(*vm.domain)
	}

	if err != nil {
		return &ec1v1.StopVMResponse{
			Success: ptr(false),
			Error:   ptr(fmt.Sprintf("Failed to stop VM: %v", err)),
		}, nil
	}

	return &ec1v1.StopVMResponse{
		Success: ptr(true),
	}, nil
}

// GetVMStatus implements the Driver interface
func (d *KVMDriver) GetVMStatus(ctx context.Context, req *ec1v1.GetVMStatusRequest) (*ec1v1.GetVMStatusResponse, error) {
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

	// Get the domain info which includes state
	// 	rState, rMaxMem, rMemory, rNrVirtCPU, rCPUTime, err := d.l.DomainGetInfo(*vm.domain)

	rState, _, _, _, _, err := d.l.DomainGetInfo(*vm.domain)
	if err != nil {
		return &ec1v1.GetVMStatusResponse{
			Response: &ec1v1.VMStatusResponse{
				Status: ptr(ec1v1.VMStatus_VM_STATUS_ERROR),
				Error:  ptr(fmt.Sprintf("Failed to get VM state: %v", err)),
			},
		}, nil
	}

	// Map libvirt state to our state
	status := mapLibvirtState(libvirt.DomainState(rState))

	return &ec1v1.GetVMStatusResponse{
		Response: &ec1v1.VMStatusResponse{
			Status:    ptr(status),
			IpAddress: ptr(vm.ipAddress),
		},
	}, nil
}

// Map libvirt domain state to our VM status
func mapLibvirtState(state libvirt.DomainState) ec1v1.VMStatus {
	switch state {
	case libvirt.DomainRunning:
		return ec1v1.VMStatus_VM_STATUS_RUNNING
	case libvirt.DomainBlocked:
		return ec1v1.VMStatus_VM_STATUS_RUNNING
	case libvirt.DomainPaused:
		return ec1v1.VMStatus_VM_STATUS_STOPPED
	case libvirt.DomainShutdown:
		return ec1v1.VMStatus_VM_STATUS_STOPPING
	case libvirt.DomainShutoff:
		return ec1v1.VMStatus_VM_STATUS_STOPPED
	case libvirt.DomainCrashed:
		return ec1v1.VMStatus_VM_STATUS_ERROR
	case libvirt.DomainPmsuspended:
		return ec1v1.VMStatus_VM_STATUS_STOPPED
	default:
		return ec1v1.VMStatus_VM_STATUS_UNSPECIFIED
	}
}

// GetHypervisorType implements the Driver interface
func (d *KVMDriver) GetHypervisorType() ec1v1.HypervisorType {
	return ec1v1.HypervisorType_HYPERVISOR_TYPE_KVM
}

// Close cleanly closes the libvirt connection
func (d *KVMDriver) Close() error {
	if d.l != nil {
		if err := d.l.Disconnect(); err != nil {
			return fmt.Errorf("disconnecting from libvirt: %w", err)
		}
	}

	if d.conn != nil {
		if err := d.conn.Close(); err != nil {
			return fmt.Errorf("closing connection: %w", err)
		}
	}

	return nil
}
