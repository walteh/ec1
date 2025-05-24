# libkrun Go Bindings

This package provides Go bindings for [libkrun](https://github.com/containers/libkrun), a library for creating and managing lightweight virtual machines (microVMs) using Firecracker-compatible APIs.

## Features

-   **Context Management**: Create and manage libkrun configuration contexts
-   **VM Configuration**: Set vCPUs, RAM, and other VM parameters
-   **Filesystem Setup**: Configure root filesystem and mount points
-   **Process Execution**: Set executable and environment for VM workloads
-   **Logging Integration**: Full `zerolog` integration for debugging
-   **Build Flexibility**: Automatic fallback to stub implementation when libkrun unavailable

## Installation

### Prerequisites

First, install libkrun on your system:

```bash
# macOS
brew install libkrun

# Ubuntu/Debian
sudo apt-get install libkrun-dev

# Fedora
sudo dnf install libkrun-devel
```

### Building with libkrun

To build with actual libkrun support:

```bash
# Build with libkrun support
./gow build -tags="cgo,libkrun" ./pkg/libkrun/example/

# Run tests with libkrun support
./gow test -tags="cgo,libkrun" ./pkg/libkrun/
```

### Building without libkrun (Development)

For development when libkrun is not available, the package automatically uses stub implementations:

```bash
# Build with stub implementation (default)
./gow build ./pkg/libkrun/example/

# Run tests with stub implementation
./gow test ./pkg/libkrun/
```

## Usage

### Basic Example

```go
package main

import (
    "context"
    "os"

    "github.com/rs/zerolog"
    "github.com/walteh/ec1/pkg/libkrun"
)

func main() {
    // Setup context with logger
    ctx := context.Background()
    logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
    ctx = logger.WithContext(ctx)

    // Set libkrun log level
    libkrun.SetLogLevel(ctx, 3) // Info level

    // Create configuration context
    kctx, err := libkrun.CreateContext(ctx)
    if err != nil {
        logger.Fatal().Err(err).Msg("failed to create context")
    }
    defer kctx.Free(ctx)

    // Configure VM
    err = kctx.SetVMConfig(ctx, 1, 512) // 1 vCPU, 512 MiB RAM
    if err != nil {
        logger.Fatal().Err(err).Msg("failed to set VM config")
    }

    // Set root filesystem
    err = kctx.SetRoot(ctx, "/path/to/rootfs")
    if err != nil {
        logger.Fatal().Err(err).Msg("failed to set root")
    }

    // Set executable to run
    err = kctx.SetExec(ctx, "/bin/echo",
        []string{"echo", "Hello World"},
        []string{"PATH=/bin:/usr/bin"})
    if err != nil {
        logger.Fatal().Err(err).Msg("failed to set executable")
    }

    // Start the VM (this will take control and exit when VM shuts down)
    err = kctx.StartEnter(ctx)
    if err != nil {
        logger.Fatal().Err(err).Msg("failed to start VM")
    }
}
```

### API Reference

#### Context Management

-   `CreateContext(ctx context.Context) (*Context, error)` - Create a new configuration context
-   `(*Context).Free(ctx context.Context) error` - Free the configuration context

#### Configuration

-   `SetLogLevel(ctx context.Context, level uint32) error` - Set libkrun log level (0=Off, 1=Error, 2=Warn, 3=Info, 4=Debug, 5=Trace)
-   `(*Context).SetVMConfig(ctx context.Context, numVCPUs uint8, ramMiB uint32) error` - Set basic VM parameters
-   `(*Context).SetRoot(ctx context.Context, rootPath string) error` - Set root filesystem path
-   `(*Context).SetExec(ctx context.Context, execPath string, argv []string, envp []string) error` - Set executable and environment

#### Execution

-   `(*Context).StartEnter(ctx context.Context) error` - Start and enter the VM (this function takes control)

## Testing

Run the test suite:

```bash
# Test with stub implementation (safe for development)
./gow test -v ./pkg/libkrun/

# Test with actual libkrun (requires libkrun installation)
./gow test -v -tags="cgo,libkrun" ./pkg/libkrun/
```

Run the example:

```bash
# Run example with stub implementation
./gow run ./pkg/libkrun/example/

# Run example with actual libkrun
./gow run -tags="cgo,libkrun" ./pkg/libkrun/example/
```

## Build Tags

The package uses Go build tags for conditional compilation:

-   **Default**: Uses stub implementation, safe for development without libkrun
-   **`cgo,libkrun`**: Uses actual CGO bindings to libkrun library

## Error Handling

All functions return errors using the `gitlab.com/tozd/go/errors` package for proper error wrapping and context. The stub implementation returns `ErrLibkrunNotAvailable` for all operations.

## Logging

The package integrates with `zerolog` for structured logging. Pass a context with a logger using `zerolog.Ctx(ctx)` to get detailed debug information about libkrun operations.

## Performance Considerations

-   Context creation/destruction is lightweight
-   VM startup through `StartEnter()` is a blocking operation that takes control of the process
-   Memory management is handled automatically through Go's garbage collector and proper C memory cleanup

## Integration with EC1

This package is designed to integrate with the EC1 fast microVM project, providing:

-   Sub-100ms boot times when properly configured
-   Init injection for SSH-free command execution
-   Apple VZ backend compatibility
-   Stream performance testing integration

For more information about EC1, see the main project documentation.
