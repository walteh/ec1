# EC1: A CloudStack Alternative in Go

EC1 is a lightweight, Go-based alternative to Apache CloudStack – providing VM and network orchestration without the legacy bloat. It uses modern Go libraries, ConnectRPC (a gRPC + Protocol Buffers framework), and simple Go services for a lean cloud management experience.

## Overview

This project is a proof-of-concept (POC) demonstrating that we can manage virtual machines (including nested VMs and networking) with minimal complexity using modern Go libraries.

Architecture:

-   **Management Server**: Central orchestration component that manages VM lifecycle across hosts
-   **Agent**: Runs on each host to execute VM operations using the local hypervisor
-   **Hypervisors**: Support for Apple Virtualization.framework (macOS) and KVM (Linux)

## Features

-   Multi-host VM orchestration with nested virtualization
-   Support for different hypervisors (macOS virtualization and KVM)
-   Networking with port forwarding
-   Simple API based on ConnectRPC (gRPC)
-   Complete infrastructure as code, written in Go

## Demo

The demo showcases a full end-to-end nested virtualization flow:

1. Start a Management Server on macOS
2. Start a local Agent on macOS
3. Create a QCOW2 image for a Linux VM
4. Start a Linux VM on macOS using Apple's Virtualization.framework
5. Set up an EC1 Agent inside the Linux VM
6. Use the Linux Agent to start a nested VM
7. Run a web server in the nested VM
8. Access the web server from the host

### Running the Demo

Prerequisites:

-   macOS (with support for Virtualization.framework)
-   Go 1.21 or later
-   QEMU and related tools (`brew install qemu`)

To run the full demo:

```bash
# Run the complete demo
./go run ./cmd/demo --action demo

# To clean up previous runs
./go run ./cmd/demo --action demo --clean

# To run individual steps
./go run ./cmd/demo --action start-mgt
./go run ./cmd/demo --action start-agent
./go run ./cmd/demo --action create-image
./go run ./cmd/demo --action start-linux-vm --disk images/alpine.qcow2
./go run ./cmd/demo --action start-nested-vm
```

## Project Structure

-   `cmd/`: Command line tools
    -   `demo/`: Demo steps and entry point
    -   `mgt/`: Management server
    -   `agent/`: Agent implementation
-   `pkg/`: Core packages
    -   `hypervisor/`: Hypervisor implementations (Apple, KVM)
    -   `management/`: Management server implementation
    -   `agent/`: Agent service implementation
-   `proto/`: Protocol Buffer definitions
-   `gen/`: Generated code from Protocol Buffers

## Development

This project is developed as a proof-of-concept to demonstrate cloud management capabilities with Go. It is not intended for production use at this stage but provides a foundation for a more complete implementation.

## License

See LICENSE file.
