# libkrun Go Bindings

This package provides Go bindings for libkrun, a library for running microVMs with minimal overhead. The package supports multiple variants of libkrun with different capabilities and includes optional high-performance vmnet networking on macOS.

## Libkrun Variants

This package supports three different variants of libkrun:

### 1. Generic (`libkrun`) - Default

-   **Build tag**: `libkrun`
-   **pkg-config**: `libkrun`
-   **Description**: Generic variant compatible with all Virtualization-capable systems
-   **Available functions**: All except SEV-specific and EFI-specific functions
-   **Use case**: General microVM workloads

### 2. SEV (`libkrun-sev`) - AMD SEV Support

-   **Build tag**: `libkrun_sev`
-   **pkg-config**: `libkrun-sev`
-   **Description**: Variant including support for AMD SEV (SEV, SEV-ES and SEV-SNP) memory encryption and remote attestation
-   **Requirements**: SEV-capable CPU
-   **Limitations**:
    -   `SetRoot()` not available
    -   `SetMappedVolumes()` not available
-   **Additional functions**: `SetSEVConfig()` for TEE configuration

### 3. EFI (`libkrun-efi`) - OVMF/EDK2 Support

-   **Build tag**: `libkrun_efi`
-   **pkg-config**: `libkrun-efi`
-   **Description**: Variant that bundles OVMF/EDK2 for booting a distribution-provided kernel
-   **Platform**: Only available on macOS
-   **Additional functions**: `GetShutdownEventFD()` for orderly shutdown

## VMNet Integration (macOS)

For optimal networking performance on macOS, this package includes optional vmnet integration that provides better performance than the default TSI backend.

### Requirements

-   **Build tag**: `vmnet_entitlement`
-   **Platform**: macOS only
-   **Entitlements**: Requires `com.apple.vm.networking` entitlement
-   **Helper**: Requires vmnet-helper installed at `/opt/vmnet-helper/bin/vmnet-helper`

### VMNet Operating Modes

#### Shared Mode (NAT)

-   Allows VMs to reach the Internet through NAT
-   VMs can communicate with the host
-   VMs can communicate with other shared mode VMs on the same subnet

#### Bridged Mode

-   Bridges VM network interface with a physical interface
-   VM appears as a separate device on the physical network
-   Requires specifying the physical interface name (e.g., "en0")

#### Host-Only Mode

-   VMs can communicate with the host and other host-only VMs
-   Optional isolation prevents VM-to-VM communication
-   No external network access

### VMNet Usage

```go
// Shared mode (NAT) - simplest option
err := kctx.SetVMNetNetworkShared(ctx, []string{"8080:80"})

// Bridged mode - VM gets its own IP on the physical network
err := kctx.SetVMNetNetworkBridged(ctx, "en0", []string{"8080:80"})

// Host-only mode with isolation
err := kctx.SetVMNetNetworkHost(ctx, true, []string{"8080:80"})

// Custom configuration
config := VMNetConfig{
    OperationMode:   vmnet.OperationModeShared,
    StartAddress:    &"192.168.100.1",
    EndAddress:      &"192.168.100.254",
    SubnetMask:      &"255.255.255.0",
    PortMap:         []string{"8080:80", "9090:90"},
    Verbose:         true,
}
err := kctx.SetVMNetNetwork(ctx, config)
```

## Installation

### macOS

```bash
# Generic variant
brew install libkrun

# EFI variant (if available)
brew install libkrun-efi

# For vmnet integration, install vmnet-helper
# (This typically requires additional setup and entitlements)
```

### Linux

Check your distribution's package manager for libkrun packages.

## Usage

### Basic Usage (Stub Implementation)

By default, without any build tags, the package uses a stub implementation that returns `ErrLibkrunNotAvailable`. This allows development and testing without libkrun installed.

```go
go run ./pkg/libkrun/example
```

### Generic Variant

```bash
go run -tags libkrun ./pkg/libkrun/example
```

### SEV Variant

```bash
go run -tags libkrun_sev ./pkg/libkrun/example
```

### EFI Variant

```bash
go run -tags libkrun_efi ./pkg/libkrun/example
```

### With VMNet (requires entitlement)

```bash
go run -tags "libkrun vmnet_entitlement" ./pkg/libkrun/example
```

## API Overview

The package provides a struct-based API that replaces long parameter lists with configuration structs:

### Core Configuration Structs

```go
// Basic VM configuration
type VMConfig struct {
    NumVCPUs uint8
    RAMMiB   uint32
}

// Disk configuration
type DiskConfig struct {
    BlockID  string
    Path     string
    Format   DiskFormat
    ReadOnly bool
}

// Network configuration
type NetworkConfig struct {
    PasstFD     *int      // nil for TSI backend
    GvproxyPath *string   // nil for TSI backend
    MAC         *[6]uint8 // nil for auto-generated
    PortMap     []string  // host:guest port mappings
}

// Process configuration
type ProcessConfig struct {
    ExecPath string
    Args     []string
    Env      []string
    WorkDir  *string // nil for default
}
```

### Variant-Specific Configuration

#### SEV Configuration (SEV variant only)

```go
type SEVConfig struct {
    TEEConfigFile *string // Path to TEE configuration file
}
```

#### VMNet Configuration (vmnet_entitlement build tag only)

```go
type VMNetConfig struct {
    InterfaceID     *string                 // nil for auto-generated
    OperationMode   vmnet.OperationMode     // shared, bridged, or host
    StartAddress    *string                 // nil for default
    EndAddress      *string                 // nil for default
    SubnetMask      *string                 // nil for default
    SharedInterface *string                 // required for bridged mode
    EnableIsolation *bool                   // for host mode
    Verbose         bool                    // enable verbose logging
    PortMap         []string                // host:guest port mappings
}
```

### Example Usage

```go
ctx := context.Background()

// Set log level ONCE per process (libkrun limitation)
if err := libkrun.SetLogLevel(ctx, libkrun.LogLevelInfo); err != nil {
    // Handle error
}

// Create context
kctx, err := libkrun.CreateContext(ctx)
if err != nil {
    // Handle error (might be ErrLibkrunNotAvailable)
    return
}
defer kctx.Free(ctx)

// Configure VM
vmConfig := libkrun.VMConfig{
    NumVCPUs: 2,
    RAMMiB:   1024,
}
kctx.SetVMConfig(ctx, vmConfig)

// Configure process
processConfig := libkrun.ProcessConfig{
    ExecPath: "/bin/echo",
    Args:     []string{"echo", "Hello World"},
    Env:      []string{"PATH=/bin"},
}
kctx.SetProcess(ctx, processConfig)

// Start VM (would actually start the microVM)
// kctx.StartEnter(ctx)
```

## Important Limitations

### Logging Initialization

⚠️ **Critical**: libkrun's internal Rust logger can only be initialized once per process. Always call `SetLogLevel()` only once at application startup:

```go
// ✅ Good: Call once at startup
func main() {
    ctx := context.Background()
    libkrun.SetLogLevel(ctx, libkrun.LogLevelInfo)
    // ... rest of application
}

// ❌ Bad: Multiple calls will cause panic
libkrun.SetLogLevel(ctx, libkrun.LogLevelDebug)
libkrun.SetLogLevel(ctx, libkrun.LogLevelInfo) // PANIC!
```

### File Validation

Many libkrun functions expect real file paths. Test with care:

```go
// This will fail with EINVAL (-22) if files don't exist
config := KernelConfig{
    Path:    "/path/to/kernel",
    Format:  KernelFormatELF,
    Cmdline: "console=ttyS0",
}
```

## Variant-Specific Function Availability

| Function               | Generic | SEV | EFI | VMNet | Notes                |
| ---------------------- | ------- | --- | --- | ----- | -------------------- |
| `SetRoot()`            | ✅      | ❌  | ✅  | ✅    | Not available in SEV |
| `SetMappedVolumes()`   | ✅      | ❌  | ✅  | ✅    | Not available in SEV |
| `SetSEVConfig()`       | ❌      | ✅  | ❌  | ❌    | SEV-specific         |
| `GetShutdownEventFD()` | ❌      | ❌  | ✅  | ❌    | EFI-specific         |
| `SetVMNetNetwork*()`   | ❌      | ❌  | ❌  | ✅    | VMNet-specific       |

## Testing

The package includes comprehensive tests that handle all variants:

```bash
# Test with stub implementation (default)
go test ./pkg/libkrun

# Test with generic libkrun
go test -tags libkrun ./pkg/libkrun

# Test with SEV variant
go test -tags libkrun_sev ./pkg/libkrun

# Test with EFI variant
go test -tags libkrun_efi ./pkg/libkrun

# Test with VMNet integration
go test -tags "libkrun vmnet_entitlement" ./pkg/libkrun
```

Tests automatically skip when libkrun is not available and log appropriate messages for variant-specific functionality.

## Build Configuration

The package uses Go build tags to select the appropriate implementation:

-   **No tags**: Stub implementation (`libkrun_stub.go`)
-   **`libkrun`**: Generic implementation (`libkrun.go`, `libkrun_generic.go`)
-   **`libkrun_sev`**: SEV implementation (`libkrun.go`, `libkrun_sev.go`)
-   **`libkrun_efi`**: EFI implementation (`libkrun.go`, `libkrun_efi.go`)
-   **`vmnet_entitlement`**: VMNet integration (`vmnet_darwin.go` or `vmnet_stub.go`)

## Error Handling

The package provides consistent error handling across all variants:

-   `ErrLibkrunNotAvailable`: Returned when libkrun is not available (stub implementation)
-   `ErrVMNetNotAvailable`: Returned when vmnet functionality is not available
-   Variant-specific errors: When calling functions not available in the current variant
-   libkrun errors: Wrapped errors from the underlying libkrun library

## Performance

The package is designed for performance-critical microVM applications:

-   Struct-based configuration reduces function call overhead
-   Direct C bindings with minimal Go overhead
-   VMNet integration provides better networking performance than TSI on macOS
-   Comprehensive benchmarks included in test suite

## Integration with EC1

This package is part of the EC1 fast microVM project, designed to achieve:

-   Sub-100ms boot times
-   SSH-free command execution via init injection
-   Apple VZ backend compatibility
-   Stream performance testing

For more information, see the main EC1 project documentation.
