# VSOCK Client

A Go implementation of a VSOCK client that can execute commands received from a VSOCK connection. This is a Go equivalent of the C-based VSOCK client presented in the original request, but with a more modular, testable design.

## Description

The VSOCK client connects to a VSOCK socket using a specified CID (Context ID) and port. It can operate in two modes:

1. **Single Command Mode (default)**: Receives one command, executes it, and exits
2. **Persistent Mode**: Continuously receives and executes commands until the connection is closed

## Architecture

The client uses a modular design with clear separation of concerns:

1. **Transport Layer** - Handles the underlying connection (VSOCK, TCP, or in-memory for testing)
2. **Protocol Layer** - Manages message framing and serialization
3. **Executor Layer** - Executes commands and handles I/O streaming

This layered approach allows for:

-   Unit testing without actual VSOCK connections
-   Swapping different transport mechanisms
-   Testing on platforms without VSOCK support (like macOS)

### Protocol

The client uses a simple binary protocol for communication:

```
[TYPE][LENGTH][DATA]
```

Where:

-   **TYPE** (1 byte): Message type (0=command, 1=stdout, 2=stderr, 3=stdin, 4=exit)
-   **LENGTH** (2 bytes): Size of the payload in little-endian format
-   **DATA**: The actual content (command or output)

## Usage

```
vsock-client [CID[:PORT]] [persistent]
```

### Parameters

-   `CID` - Context ID (defaults to 2/VMADDR_CID_HOST)
-   `PORT` - Port number (defaults to 1024)
-   `persistent` - If provided, will continuously process commands

### Examples

```bash
# Connect to host on port 1024
vsock-client

# Connect to CID 3 on port 1024
vsock-client 3

# Connect to CID 3 on port 2048
vsock-client 3:2048

# Connect to CID 3 on port 2048 in persistent mode
vsock-client 3:2048 persistent
```

## Building

From the project root:

```bash
go build -o bin/vsock-client ./cmd/vsock-client
```

## Testing

The modular design allows easy testing without actual VSOCK connections:

```bash
go test ./cmd/vsock-client/...
```

## Requirements

-   Go 1.15 or later
-   Linux with VSOCK support for actual VSOCK connections
-   The `github.com/mdlayher/vsock` Go package

## VSOCK Context IDs

The client supports the following predefined Context IDs:

-   `VMADDR_CID_ANY` (0xFFFFFFFF): Any context
-   `VMADDR_CID_HYPERVISOR` (0): Hypervisor context
-   `VMADDR_CID_LOCAL` (1): Local context
-   `VMADDR_CID_HOST` (2): Host context
-   Any positive integer greater than 2 for specific VMs

## Security Considerations

**Warning**: This tool executes commands as received without validation. Use in trusted environments only.
