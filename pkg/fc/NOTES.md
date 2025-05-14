# Firecracker API Implementation Notes

This document contains implementation notes, known issues, and enhancement ideas for the Firecracker API implementation in the `fc` package.

## Overview

The `fc` package implements the Firecracker REST API using the hypervisors package as the backend. It serves as a translation layer between Firecracker API operations and hypervisor operations, allowing VMs to be managed through the standard Firecracker API while supporting different hypervisor backends.

## API Implementation Status

| Category              | Status          | Notes                                    |
| --------------------- | --------------- | ---------------------------------------- |
| VM Lifecycle          | Partial         | Core start/stop/pause implemented        |
| Machine Configuration | Complete        | All parameters mapped                    |
| Boot Source           | Complete        | Kernel, initrd, boot args supported      |
| Block Devices         | Partial         | Basic configuration only                 |
| Network Interfaces    | Partial         | Basic configuration only                 |
| Memory Balloon        | Not Implemented | Requires balloon device support          |
| Snapshots             | Not Implemented | Requires snapshot support in hypervisors |
| MMDS                  | Not Implemented | Requires metadata service support        |

## Implementation Details

### VM Lifecycle

The implementation treats VM creation as a multi-step process:

1. The client creates a VM configuration through multiple API calls
2. The implementation tracks the pending configuration until all required parts are present
3. When required configurations are available, a VM is created via the hypervisor
4. Once created, the VM can be managed via sync actions (start, stop, pause, etc.)

### State Management

The implementation uses a `vmRegistry` to track active VMs and a `pendingVMConfigs` map to track VM configurations that are in the process of being created. This approach handles the fact that Firecracker API divides VM configuration across multiple API calls.

### Configuration Mapping

API parameters are mapped to hypervisor constructs:

-   **Machine Configuration**: Maps to `NewVMOptions`
-   **Boot Source**: Maps to `bootloader.Config`
-   **Drives**: Maps to virtio devices (implementation pending)
-   **Network Interfaces**: Maps to virtio devices (implementation pending)

## Enhancement Ideas for Hypervisors Package

### 1. VM Lookup Methods

Adding methods to find VMs by ID would simplify integration:

```go
// Add to Hypervisor interface
GetVirtualMachine(id string) (VM, bool)
ListVirtualMachines() []VM
```

### 2. Device Management APIs

Standardized APIs for device management would simplify device creation:

```go
// Block device helper
CreateBlockDevice(path string, readOnly bool) (virtio.VirtioDevice, error)

// Network device helper
CreateNetworkDevice(macAddr string, tapName string) (virtio.VirtioDevice, error)
```

### 3. Configuration Management

A builder pattern for VM options would make configuration cleaner:

```go
vmOpts := hypervisors.NewVMOptionsBuilder().
    WithVCPUs(2).
    WithMemory(strongunits.GiB(4)).
    WithBlockDevice("/path/to/disk.img", false).
    WithNetworkDevice("52:54:00:12:34:56", "tap0").
    Build()
```

### 4. Enhanced VM State Management

More detailed state information:

```go
type VirtualMachineError struct {
    Code    string
    Message string
    Details map[string]string
}

// Add to VirtualMachine interface
LastError() *VirtualMachineError
StateHistory() []VirtualMachineStateChange
```

### 5. Direct Reboot Support

Add direct reboot support to avoid stop-then-start pattern:

```go
// Add to VirtualMachine interface
Reboot(ctx context.Context) error
CanReboot(ctx context.Context) bool
```

## Known Issues and Limitations

1. **Hot-plug Operations**: Currently no support for modifying VM configuration after creation
2. **Device Conversion**: Need better mapping between Firecracker device models and virtio devices
3. **Error Handling**: Limited error details from hypervisor operations
4. **State Transitions**: Some state transitions might not be properly validated
5. **Cleanup**: No automatic cleanup of pending configurations that become stale

## Integration Testing Notes

For testing the Firecracker API implementation:

1. Use the standard Firecracker API tests from AWS
2. Create dedicated test VMs with minimal configurations
3. Test each API endpoint individually, then in sequence
4. Verify proper error handling with invalid inputs
5. Test concurrent API operations to ensure thread safety

## Future Work

1. Complete implementation of all Firecracker API operations
2. Add vsock device support for VM communication
3. Implement snapshot and restore operations
4. Add support for Firecracker MMDS (Microvm Metadata Service)
5. Implement proper cleanup of resources on VM termination
6. Add comprehensive metrics and monitoring
