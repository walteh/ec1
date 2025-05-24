# EC1 Cursor Development Environment

Welcome to the **EC1 Fast MicroVM Project** - building the world's fastest Firecracker-compatible microVM implementation! 🚀

## 🎯 Project Mission

We're creating a **sub-100ms boot time microVM** with **SSH-free command execution** using:

-   ⚡ **Apple Virtualization Framework** for high-performance VM management
-   🔥 **Firecracker REST API** compatibility
-   🚨 **Init injection system** for direct command execution (our secret weapon!)
-   🧪 **Advanced performance testing** framework

## 📁 Directory Structure

```
.cursor/
├── README.md                    # This file - comprehensive project guide
├── environment.json             # Cursor environment configuration
├── docker/                      # Container setup for background agents
├── rules/                       # Development standards and conventions
│   ├── background.mdc          # Always-applied project context
│   ├── firecracker-integration.mdc  # Firecracker API implementation rules
│   ├── golang-general.mdc      # Go development standards
│   └── golang-testing.mdc      # Testing best practices
└── legacy/                     # Historical documentation
    ├── dr-b-*.md              # Dr B background agent docs (archived)
    └── BACKGROUND_AGENT_*.md   # Previous agent setup guides
```

## 🛠️ Development Tools

### GOW - Enhanced Go Wrapper (NEW!)

We've transitioned from `./go` to `./gow` - a lightweight wrapper that uses Go's tool system:

```bash
# The new gow wrapper (55 bytes!)
#!/usr/bin/env bash
set -euo pipefail
go tool gow "$@"
```

**Core Commands:**

```bash
./gow test -v ./...                      # Standard go test with enhancements
./gow test -function-coverage            # Function-level coverage analysis
./gow test -codesign                     # Test with code signing (macOS)
./gow test -ide                          # IDE-compatible output
./gow mod tidy                           # Workspace-aware mod tidy
./gow mod upgrade                        # Upgrade dependencies
./gow tool [args...]                     # Execute go tools
./gow dap                               # Debug adapter protocol
./gow retab                             # Code formatting tool
```

**Performance Benchmarks:**

-   `go mod tidy`: 0.291s
-   Task runner: 0.563s
-   **GOW**: 0.280s (fastest!)

### Stream Performance Testing Framework

Located in `pkg/testing/tstream/` - catches performance bottlenecks automatically:

```go
// Components:
- TimingReader      // Automatic bottleneck detection
- ProgressReader    // Real-time progress with ETA
- ProfiledReader    // CPU/memory profiling integration
- StreamBenchmark   // Comparative performance testing
```

**Real Impact:** Identified gzip compression bottleneck in initramfs injection → 93% improvement (2.27s → 157ms)!

## 🏗️ Project Architecture

### 1. Firecracker API Implementation (`pkg/firecracker/`)

```
pkg/firecracker/
├── api.go          # MAIN API implementation (FirecrackerMicroVM)
├── NOTES.md        # Implementation status and enhancement ideas
├── server.go       # HTTP server setup (TODO)
└── handlers/       # Individual endpoint handlers (TODO)
```

**Current Status:**

-   ✅ Basic API structure with Apple VZ backend integration
-   ✅ VM lifecycle operations (start/stop/pause)
-   ✅ Machine configuration endpoints
-   ✅ Snapshot support (save/restore)
-   🚧 Network and storage device management
-   🚧 Full Firecracker compatibility

### 2. Init Injection System (`pkg/bootloader/`)

Our **secret weapon** for SSH-free command execution:

```go
// PrepareInitramfsCpio - Injects custom init while preserving original
// Result: Direct gRPC command execution without SSH overhead!
```

### 3. VMM Layer (`pkg/vmm/`)

Abstraction layer for virtual machine management:

-   Generic VM interface
-   Apple VZ implementation
-   Libkrun support (experimental)
-   State management and lifecycle

### 4. Apple VZ Backend (`pkg/vmm/vf/`)

Native macOS virtualization integration:

-   High-performance VM operations
-   Native memory balloon support
-   Snapshot/restore capabilities
-   Direct kernel/initramfs loading

## 🧪 Development Standards

### Go Conventions

**Imports:**

-   Always use `./gow` instead of `go` directly
-   Never modify `go.mod`/`go.sum` manually - use `./gow mod tidy`
-   Assume broken imports need reference adjustment, not creation

**Logging:**

-   Use `slog` exclusively with context propagation
-   Pass `context.Context` as first parameter
-   Use `slog.InfoContext(ctx, "message", attrs...)`

**Error Handling:**

-   Use `gitlab.com/tozd/go/errors` for all errors
-   Use `errors.Errorf("action: %w", err)` for wrapping
-   Error messages describe what was being attempted

**Testing:**

-   Maintain >85% function coverage (enforced)
-   Use `testify` for assertions and mocking
-   Use `./gow test -function-coverage` to verify
-   Performance benchmarks for critical paths

### Firecracker Integration Rules

**Performance First:**

-   Boot time target: <100ms
-   Command execution: <10ms overhead
-   Memory usage: <50MB baseline
-   Use stream testing tools for optimization

**API Compatibility:**

-   100% Firecracker REST API compatible
-   Use `gen/firecracker-swagger-go/` definitions
-   Test with real Firecracker clients
-   Document any extensions clearly

## 🚀 Getting Started

### 1. Build the GOW tool

```bash
# GOW is pre-built at the root
chmod +x ./gow
./gow version
```

### 2. Run the test suite

```bash
# Comprehensive testing with coverage
./gow test -function-coverage ./...

# Performance benchmarks
./gow test -run Benchmark ./pkg/initramfs/

# Stream performance validation
./gow test ./pkg/testing/tstream/
```

### 3. Explore key components

```bash
# Firecracker API implementation
./gow test ./pkg/firecracker/

# VMM abstraction layer
./gow test ./pkg/vmm/

# Apple VZ backend
./gow test ./pkg/vmm/vf/
```

## 📊 Performance Targets

-   **Boot time**: <100ms from API call to VM ready
-   **API latency**: <5ms for configuration endpoints
-   **Command execution**: <10ms overhead vs native
-   **Memory footprint**: <50MB for basic workloads
-   **Test coverage**: >85% function coverage

## 🔥 Firecracker Implementation Status

### Completed ✅

-   Basic API structure and server setup
-   VM lifecycle management (create/start/stop)
-   Machine configuration (CPU/memory)
-   Memory balloon operations
-   Snapshot save/restore
-   Integration with Apple VZ backend

### In Progress 🚧

-   Network interface management
-   Block device configuration
-   MMDS (metadata service)
-   Full API compatibility testing
-   Performance optimization

### TODO 📋

-   vsock device support
-   Entropy device configuration
-   CPU templates
-   Metrics and logging endpoints
-   Hot-plug operations

## 💡 Pro Tips

### Performance Debugging

```bash
# CPU profiling
./gow test -run BenchmarkVMBoot -cpuprofile=cpu.prof

# Memory profiling
./gow test -run TestMemoryUsage -memprofile=mem.prof

# Stream performance analysis
./gow test -v ./pkg/testing/tstream/
```

### Using Init Injection

```go
// Leverage our SSH-free execution
bootloader := &bootloader.LinuxBootloader{
    InitInjection: true,  // Our competitive advantage!
    CustomInit:    embeddedGRPCInit,
}
```

### Testing with Real Firecracker Clients

```bash
# Start the API server
./gow run ./pkg/firecracker/cmd/server/

# Test with Firecracker SDK
firecracker-go-sdk --api-socket /tmp/firecracker.sock

# Test with Kata Containers
kata-runtime --firecracker-binary /path/to/our/api
```

## 🎯 Current Focus Areas

1. **Firecracker API Completion** - Achieve 100% API compatibility
2. **Performance Optimization** - Sub-100ms boot consistently
3. **Integration Testing** - Validate with real-world clients
4. **Documentation** - API guides and performance reports

## 🚨 Important Notes

-   **Always use `./gow`** instead of `go` directly
-   **Test coverage >85%** is enforced by CI
-   **Performance regressions** are blocking issues
-   **Init injection** is our key differentiator - leverage it!

---

**Questions?** Check the code, run the tests, and let's build the fastest microVM ever created! 🚀

**Project Lead**: Mr C & The AI Team 🤖
