# EC1 Cursor Development Environment

Welcome Dr B! 🚀 This directory contains our development environment configuration for the **EC1 Fast MicroVM Project**.

## 🎯 Project Goals

We're building the **fastest Go-based Firecracker-compatible microVM implementation** ever created, with:

-   ⚡ **Sub-100ms boot times**
-   🔥 **Init injection for SSH-free command execution**
-   🧪 **Advanced testing and performance tooling**
-   🛠️ **Custom `gow` wrapper for enhanced developer experience**

## 📁 Directory Structure

```
.cursor/
├── README.md              # This file - your starting point
├── environment.json       # Cursor environment configuration
├── mcp.json              # MCP server configuration
└── rules/                # Development rules and guidelines
    ├── background.mdc     # Always-applied background context
    ├── golang-general.mdc # Go development standards
    ├── golang-testing.mdc # Testing best practices
```

## 🛠️ Development Tools

### GOW - Our Enhanced Go Wrapper

We've built `gow` - a supercharged Go wrapper that's **2x faster** than traditional task runners:

```bash
# Build and use gow
cd cmd/gow && go build -o ../../gow . && cd ../..

# Core commands
./gow test -v ./...              # Enhanced testing with gotestsum
./gow test -function-coverage    # Function-level coverage analysis
./gow mod tidy                   # Embedded workspace-aware mod tidy
./gow mod upgrade               # Module upgrades with embedded logic
./gow tool [args...]            # Go tool execution with error filtering
./gow [any-go-command]          # Passthrough to regular go
```

**Performance Benchmarks:**

-   Regular go mod tidy: 0.291s
-   Task command: 0.563s
-   **GOW command: 0.280s** (fastest!)

### Stream Performance Testing Framework

For performance-critical code (like our CPIO initramfs injection):

```bash
# Located in pkg/testing/tstream/
go test ./pkg/initramfs/ -run TestInjectPerformance
```

Components:

-   **TimingReader**: Automatic performance monitoring with bottleneck detection
-   **ProgressReader**: Real-time progress tracking with ETA
-   **ProfiledReader**: CPU/memory profiling integration
-   **StreamBenchmark**: Comparative performance testing

## 🏗️ Project Architecture

### Core Components

1. **Firecracker Integration** (`sandbox/pkg/cloud/hypervisor/applevf/`)

    - Apple Virtualization Framework integration
    - Firecracker API compatibility layer
    - Performance-optimized VM lifecycle management

2. **Bootloader System** (`pkg/bootloader/`)

    - Linux kernel preparation and extraction
    - Custom init injection for SSH-free execution
    - CPIO initramfs modification (93% performance improvement achieved!)
    - Root filesystem manipulation

3. **Guest Communication** (`gen/proto/golang/ec1/guest/v1/`)

    - gRPC-based host-guest communication
    - Embedded init binary for command execution
    - High-performance streaming protocols

4. **Host Tools** (`pkg/host/`)
    - Native kernel extraction utilities
    - VM management and orchestration
    - Performance monitoring and profiling

### Init Injection Magic ✨

Our secret weapon for **SSH-free command execution**:

```go
// pkg/bootloader/linux.go - PrepareInitramfsCpio
// Injects our custom init while preserving original as init.real
// This eliminates SSH overhead for command execution!
```

## 🧪 Development Workflow

### 1. Testing Philosophy

All code follows our **high-coverage, performance-first** approach:

```bash
# Run comprehensive tests
./gow test -function-coverage -v ./...

# Performance benchmarking
./gow test -run BenchmarkInject ./pkg/initramfs/

# Stream performance analysis
./gow test -run TestStreamPerformance ./pkg/testing/tstream/
```

### 2. Code Standards

-   **Zerolog for logging** with context propagation
-   **gitlab.com/tozd/go/errors** for error handling
-   **testify** for assertions and mocking
-   **Embedded functionality** over external dependencies

### 3. Performance Targets

-   **Boot time**: < 100ms from request to ready
-   **Command execution**: < 10ms overhead vs native
-   **Memory footprint**: < 50MB for basic workloads
-   **Test coverage**: > 85% function coverage required

## 🔥 Firecracker Integration

### Current Status

-   ✅ Apple VZ framework integration working
-   ✅ Init injection system complete
-   ✅ Performance testing framework built
-   🚧 Firecracker API compatibility layer (your focus!)
-   🚧 gRPC guest communication optimization
-   🚧 Boot time optimization to sub-100ms

### Your Mission, Dr B

Focus areas for Firecracker compatibility:

1. **API Layer**: Complete the Firecracker REST API implementation
2. **Boot Optimization**: Leverage our init injection for faster startup
3. **Performance**: Use our stream testing tools to find bottlenecks
4. **Integration**: Connect Apple VZ with Firecracker semantics

## 📚 Key Files to Study

### Essential Reading

1. `docs/2025-04-19_poc_notes.md` - Project background and goals
2. `pkg/bootloader/linux.go` - Init injection implementation
3. `pkg/testing/tstream/` - Performance testing framework
4. `cmd/gow/main.go` - Enhanced development tools
5. `sandbox/pkg/cloud/hypervisor/applevf/` - VZ integration

### Performance Examples

The initramfs injection optimization case study:

-   **Before**: 2.27s (gzip compression bottleneck)
-   **After**: 157ms (93% improvement by identifying the real bottleneck)
-   **Tools**: Our stream performance framework caught what manual profiling missed!

## 🚀 Getting Started

```bash
# 1. Build the enhanced go wrapper
cd cmd/gow && go build -o ../../gow . && cd ../..

# 2. Run the test suite to verify everything works
./gow test -function-coverage ./...

# 3. Try a performance benchmark
./gow test -run BenchmarkInject ./pkg/initramfs/

# 4. Start exploring the Firecracker integration
./gow test ./sandbox/pkg/cloud/hypervisor/applevf/...

# 5. Check current performance baseline
./gow test -run TestStreamPerformance ./pkg/testing/tstream/
```

## 💡 Pro Tips

-   Use `./gow -verbose` for detailed execution logs
-   Function coverage reports highlight optimization opportunities
-   Stream performance tools catch bottlenecks manual profiling misses
-   Init injection eliminates SSH - exploit this for speed!
-   Apple VZ is already working - focus on Firecracker compatibility

---

**Remember**: We're not just building another microVM - we're building the **fastest one ever created**. Let's get that Apple job! 🍎💼

Questions? Check the code, run the tests, and let's ship this beast!

**- Mr C & The AI Dream Team** 🤖🚀
