# EC1 MicroVM Development Environment

Welcome to the **EC1 Fast MicroVM Project** - building the world's fastest Firecracker-compatible microVM implementation! 🚀

## 🎯 Project Mission

We're creating **VMs as easy and performant as Docker containers** using:

-   ⚡ **Sub-100ms boot times** with Apple Virtualization Framework
-   🔥 **Firecracker API compatibility** for ecosystem integration
-   🚨 **Init injection system** for SSH-free command execution (our secret weapon!)
-   🧪 **Advanced performance testing** framework

## 📁 Cursor Environment Structure

```
.cursor/
├── README.md                    # This file - comprehensive guide
├── environment.json             # Terminal and build configuration
├── install.sh                   # Background agent setup script
├── start.sh                     # Environment startup script
├── Dockerfile                   # Container setup for agents
└── rules/                       # Development standards and conventions
    ├── background.mdc          # Always-applied project context
    ├── firecracker-support.mdc # Firecracker implementation guide
    ├── golang-general.mdc      # Go development standards
    ├── golang-testing.mdc      # Testing best practices
    └── sandbox.mdc             # Legacy/temporary code marker
```

## 🛠️ Development Tools

### GOW - Enhanced Go Wrapper

The project uses `./gow` - a lightweight wrapper around `go tool gow`:

```bash
./gow test -function-coverage    # Test with >85% coverage (required)
./gow test -codesign            # Test with code signing (macOS features)
./gow test -bench=.             # Run performance benchmarks
./gow mod tidy                  # Workspace-aware dependency management
./gow tool [args...]            # Execute go tools with filtering
./gow dap                       # Debug adapter protocol for VS Code
```

### Key Project Packages

-   **`pkg/firecracker/`** - Main Firecracker API implementation
-   **`pkg/vmm/`** - Virtual machine management abstraction
-   **`pkg/vmm/vf/`** - Apple VZ backend implementation
-   **`pkg/bootloader/`** - Init injection system (SSH-free execution)
-   **`pkg/testing/tstream/`** - Stream performance testing framework

## 🚀 Quick Start for Background Agents

### 1. Environment Setup

The `.cursor/install.sh` script automatically configures:

-   Go 1.24.3 with proper PATH
-   Essential development tools (gotestsum, mockery, dlv)
-   Private module access (`GOPRIVATE=github.com/walteh`)
-   Development aliases and shortcuts
-   Workspace verification

### 2. Terminal Configurations

Environment provides 5 pre-configured terminals:

-   **firecracker-dev**: Firecracker API development
-   **vmm-dev**: VMM abstraction layer
-   **performance-tools**: Stream testing framework
-   **init-injection**: Boot loader and init system
-   **full-coverage**: Complete test suite with coverage

### 3. Key Development Commands

```bash
# Navigate to workspaces
firecracker    # → pkg/firecracker/
vmm           # → pkg/vmm/
bootloader    # → pkg/bootloader/
performance   # → pkg/testing/tstream/

# Testing shortcuts
gowtest                    # Function coverage testing
fulltest                   # Full project test suite
benchmark-firecracker      # Firecracker performance tests
benchmark-vmm             # VMM performance tests

# Development workflow
./gow test -function-coverage ./...  # Verify >85% coverage
./gow test -bench=. ./pkg/firecracker/  # Performance validation
./gow test -tags=integration ./...   # Integration tests
```

## 📊 Performance Standards

### Non-Negotiable Targets

-   **VM Boot Time**: <100ms from API call to ready
-   **Command Execution**: <10ms overhead (via gRPC init, NOT SSH)
-   **Memory Footprint**: <50MB for basic workloads
-   **API Latency**: <5ms for configuration endpoints
-   **Test Coverage**: >85% function coverage (enforced)

### Performance Testing

Use the stream performance framework for I/O validation:

```go
import "github.com/walteh/ec1/pkg/testing/tstream"

reader := tstream.NewTimingReader(dataReader)
// Automatic bottleneck detection and reporting
```

## 🏗️ Architecture Principles

### 1. VMM Abstraction Pattern

```go
// DO: Use VMM abstraction layer
import "github.com/walteh/ec1/pkg/vmm"
vm, err := hypervisor.NewVirtualMachine(ctx, id, options, bootloader)

// DON'T: Access backends directly
import "github.com/walteh/ec1/pkg/vmm/vf" // ❌ Direct VZ access
```

### 2. Init Injection Advantage

```go
// Our secret weapon - SSH-free execution
bootloader := &bootloader.LinuxBootloader{
    VmlinuzPath:   vmi.KernelPath(),
    InitrdPath:    vmi.InitramfsPath(), // Contains gRPC init
    KernelCmdLine: vmi.KernelArgs(),
    InitInjection: true, // Enables direct command execution
}
```

### 3. Firecracker API Compatibility

-   Use `gen/firecracker-swagger-go/` models exclusively
-   Maintain 100% JSON response compatibility
-   Test with real Firecracker clients (Kata, SDK)

## 🧪 Development Standards

### Go Conventions

-   **Always use `./gow`** instead of `go` directly
-   **slog for logging** with context propagation: `slog.InfoContext(ctx, ...)`
-   **gitlab.com/tozd/go/errors** for error handling: `errors.Errorf("action: %w", err)`
-   **Struct-based configuration** over multiple parameters
-   **Generics for abstractions** (e.g., `FirecrackerMicroVM[V vmm.VirtualMachine]`)

### Testing Requirements

-   **>85% function coverage mandatory** (CI enforced)
-   **Performance benchmarks** for critical paths
-   **Use project testing utilities**: `pkg/testing/tstream/`, `tlog/`, `tctx/`
-   **Build tags awareness**: Some tests require `-tags libkrun_efi`, etc.

### Error Handling Patterns

-   **Non-critical feature failures**: Log warning and continue gracefully
-   **Context-first functions**: `func MyFunc(ctx context.Context, ...) error`
-   **Descriptive error messages**: What was being attempted, not what failed

## 🔥 Current Implementation Status

### Firecracker API (pkg/firecracker/)

```
✅ VM lifecycle (create/start/stop/pause)
✅ Machine configuration (CPU/memory)
✅ Memory balloon operations
✅ Snapshot save/restore
✅ Instance information API
✅ Apple VZ backend integration

🚧 Network interface management
🚧 Block device configuration
🚧 HTTP REST server
🚧 Full API compatibility testing

❌ MMDS (metadata service)
❌ vsock devices
❌ CPU templates
❌ Metrics endpoints
```

## 📚 Essential Documentation

-   **[docs/firecracker-support.md](../docs/firecracker-support.md)** - Complete strategy and alternatives analysis
-   **[pkg/firecracker/NOTES.md](../pkg/firecracker/NOTES.md)** - Implementation status details
-   **Rules in `.cursor/rules/`** - Development standards and patterns

## 🚨 Critical Guidelines

1. **Never bypass performance testing** - Use `pkg/testing/tstream/`
2. **Never access VZ directly** - Use VMM abstraction (`pkg/vmm/`)
3. **Never break API compatibility** - Must work with existing Firecracker clients
4. **Never add SSH dependencies** - We have init injection!
5. **Never drop below 85% coverage** - CI will block deployment

## 💡 Pro Tips for Background Agents

### Performance Debugging

```bash
# CPU profiling
./gow test -run BenchmarkVMBoot -cpuprofile=cpu.prof

# Memory profiling
./gow test -run TestMemoryUsage -memprofile=mem.prof

# Stream performance analysis
./gow test -v ./pkg/testing/tstream/
```

### Development Workflow

```bash
# Start with health check
./gow test -function-coverage ./...

# Focus on specific area
firecracker && ./gow test -v ./...

# Validate performance
./gow test -bench=. ./pkg/firecracker/

# Check integration
./gow test -tags=integration ./...
```

### Build Tag Considerations

Some features require specific build tags:

```bash
./gow test -tags libkrun ./pkg/libkrun/
./gow test -tags integration ./pkg/firecracker/
./gow test -tags codesign ./...  # macOS code signing tests
```

## 🎯 Strategic Context

**Primary Goal**: Make VMs as easy and performant as Docker containers
**Platform Focus**: macOS first with Apple Virtualization Framework  
**Ecosystem Integration**: 100% Firecracker API compatibility
**Competitive Advantage**: SSH-free execution via init injection

Our init injection system eliminates the 50-100ms SSH overhead that every other microVM solution suffers from. This, combined with Apple VZ's native performance, positions us to achieve boot times and command execution speeds that exceed traditional solutions.

**Goal**: Make our Firecracker implementation on macOS the fastest in the world! 🚀

---

**Questions?** Check the rules, run the tests, and focus on making every feature faster than before!
