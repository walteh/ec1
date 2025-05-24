# Dr B Onboarding - EC1 Fast MicroVM Project

Welcome to the team! 🎉 This document provides your complete onboarding and current project status.

## 🎯 Mission Critical Goals

We're building the **world's fastest Firecracker-compatible microVM implementation** using Go and Apple's Virtualization Framework. Our secret weapon: **init injection for SSH-free command execution**.

### Performance Targets (Non-negotiable)

-   **Boot time**: < 100ms from API call to VM ready
-   **Command execution**: < 10ms overhead vs bare metal
-   **Memory footprint**: < 50MB for basic workloads
-   **Test coverage**: > 85% function coverage (enforced)

## 🚀 What's Already Built (Your Foundation)

### 1. GOW - Enhanced Go Wrapper ✅ COMPLETE

**Location**: `cmd/gow/main.go`
**Performance**: 2x faster than task runners!

```bash
# Your new best friend
./gow test -function-coverage -v ./...     # Enhanced testing
./gow mod tidy                             # Workspace-aware (0.280s vs task's 0.563s!)
./gow tool [args...]                       # Error-filtered go tools
```

**Why it matters**: Development velocity is crucial. This tool eliminates friction.

### 2. Stream Performance Testing Framework ✅ COMPLETE

**Location**: `pkg/testing/tstream/`
**Achievement**: Caught 93% performance improvement in initramfs injection!

```bash
# Use this to find bottlenecks
./gow test -run TestStreamPerformance ./pkg/testing/tstream/
```

Components:

-   `TimingReader`: Automatic bottleneck detection
-   `ProgressReader`: Real-time progress with ETA
-   `ProfiledReader`: CPU/memory profiling integration
-   `StreamBenchmark`: Comparative performance testing

**Real-world impact**: Identified gzip compression as bottleneck (not CPIO processing) in initramfs injection, leading to 2.27s → 157ms improvement.

### 3. Init Injection System ✅ WORKING

**Location**: `pkg/bootloader/linux.go`
**Secret sauce**: SSH-free command execution!

```go
// PrepareInitramfsCpio - injects our custom init
// Replaces /init with our gRPC-enabled binary
// Original init becomes /init.real
// Result: Direct command execution without SSH overhead!
```

**Why this is revolutionary**: Every other microVM solution uses SSH for command execution. We execute directly through our embedded init. This is our competitive advantage.

### 4. Apple VZ Integration ✅ WORKING

**Location**: `sandbox/pkg/cloud/hypervisor/applevf/`
**Status**: Functional VM creation and management

**Your advantage**: Don't reinvent VM management - build the Firecracker API layer on top of this.

## 🎯 Your Mission: Firecracker Compatibility Layer

### What You're Building

Create a **100% Firecracker API-compatible layer** that uses our Apple VZ backend + init injection for unprecedented performance.

### Architecture Overview

```
Firecracker REST API (your layer)
        ↓
Apple VZ Backend (existing)
        ↓
Init Injection System (existing)
        ↓
Sub-100ms boot + SSH-free execution
```

### Directory Structure for Your Work

```
sandbox/pkg/cloud/hypervisor/firecracker/   # Your main workspace
├── api/                                     # REST API handlers
│   ├── handlers.go                         # Firecracker endpoint implementations
│   ├── models.go                           # Request/response structures
│   └── server.go                           # HTTP server setup
├── vm/                                     # VM lifecycle management
│   ├── manager.go                          # VM creation/deletion/management
│   ├── config.go                           # Configuration translation
│   └── lifecycle.go                        # Start/stop/pause operations
├── config/                                 # Configuration management
│   ├── firecracker.go                      # Firecracker config parsing
│   ├── applevz.go                          # Translation to Apple VZ
│   └── validation.go                       # Resource validation
└── performance/                            # Performance optimization
    ├── benchmarks_test.go                  # Boot time benchmarks
    ├── profiling.go                        # Performance monitoring
    └── optimization.go                     # Performance tuning
```

## 🛠️ Development Environment Setup

### 1. Build GOW (Your Development Tool)

```bash
cd cmd/gow && go build -o ../../gow . && cd ../..
```

### 2. Verify Everything Works

```bash
# Run the full test suite
./gow test -function-coverage ./...

# Test stream performance tools
./gow test -run TestStreamPerformance ./pkg/testing/tstream/

# Verify Apple VZ integration
./gow test ./sandbox/pkg/cloud/hypervisor/applevf/...
```

### 3. Study the Existing Code

Essential files to understand:

1. `pkg/bootloader/linux.go` - Init injection magic
2. `pkg/testing/tstream/` - Performance testing tools
3. `sandbox/pkg/cloud/hypervisor/applevf/` - Apple VZ integration
4. `gen/firecracker-swagger-go/` - Firecracker API definitions

## 📋 Your Immediate Tasks (First Week)

### Task 1: Environment Verification ⏰ Day 1

-   [ ] Build and run `./gow` successfully
-   [ ] Run `./gow test -function-coverage ./...` and achieve >85% in new code
-   [ ] Study `pkg/bootloader/linux.go` to understand init injection
-   [ ] Run Apple VZ tests to verify existing functionality

### Task 2: API Foundation ⏰ Days 2-3

-   [ ] Create `sandbox/pkg/cloud/hypervisor/firecracker/api/` structure
-   [ ] Implement basic HTTP server using `gen/firecracker-swagger-go/` models
-   [ ] Add health check endpoint (`GET /ping`)
-   [ ] Add machine configuration endpoint (`PUT /machine-config`)
-   [ ] Test with real Firecracker client tools

### Task 3: VM Integration ⏰ Days 4-5

-   [ ] Create VM manager that uses Apple VZ backend
-   [ ] Implement VM creation with init injection enabled
-   [ ] Add boot time measurement using our stream performance tools
-   [ ] Target: < 100ms boot time from API call to ready

### Task 4: Performance Optimization ⏰ End of Week 1

-   [ ] Add performance benchmarks using `pkg/testing/tstream/`
-   [ ] Profile critical paths with our tools
-   [ ] Optimize configuration translation (Firecracker → Apple VZ)
-   [ ] Document performance wins vs standard Firecracker

## 🔬 Testing Strategy

### Performance Testing (Mandatory)

Every feature must be benchmarked:

```bash
# Example benchmark structure
./gow test -run BenchmarkFirecrackerVMBoot -benchmem
./gow test -run TestFirecrackerBootTime -timeout=30s
./gow test -function-coverage ./sandbox/pkg/cloud/hypervisor/firecracker/
```

### Integration Testing

Test against real Firecracker consumers:

-   Kata Containers runtime
-   Weave Ignite
-   Firecracker-microvm CLI tools

### Coverage Requirements

-   **Function coverage > 85%** (enforced by CI)
-   All critical paths must have performance benchmarks
-   Error paths must be tested

## 🎯 Key Success Metrics

### Technical Milestones

-   [ ] **Week 1**: Basic Firecracker API responding correctly
-   [ ] **Week 2**: VM boot time < 100ms consistently
-   [ ] **Week 3**: Command execution without SSH < 10ms overhead
-   [ ] **Week 4**: Full API compatibility with existing Firecracker consumers

### Performance Goals

-   **Boot Performance**: Sub-100ms from API call to ready
-   **API Response Time**: < 5ms for configuration endpoints
-   **Memory Efficiency**: < 50MB baseline per VM
-   **Command Execution**: Direct gRPC vs SSH (massive improvement)

## 🚨 Critical Don'ts

1. **Don't bypass performance testing** - Use `pkg/testing/tstream/` for everything
2. **Don't reinvent VM management** - Build on Apple VZ layer
3. **Don't break API compatibility** - Must work with existing Firecracker consumers
4. **Don't add SSH dependencies** - We have init injection!
5. **Don't ignore test coverage** - 85% minimum enforced

## 💡 Pro Tips for Success

### Debugging Performance Issues

```bash
# Profile specific operations
./gow test -run BenchmarkFirecrackerOperation -cpuprofile=cpu.prof

# Check for bottlenecks
./gow test -run TestStreamPerformance -v

# Memory profiling
./gow test -run TestFirecrackerMemory -memprofile=mem.prof
```

### Using Our Init Injection Advantage

```go
// In your VM creation code
bootConfig := &bootloader.Config{
    InitInjection: true,  // This is our secret weapon!
    CustomInit:    "/path/to/grpc/init",
    // This eliminates SSH overhead completely
}
```

### Leveraging Existing Apple VZ Code

```go
// Don't reinvent - extend
import "github.com/walteh/ec1/sandbox/pkg/cloud/hypervisor/applevf"

// Use existing VM creation but add Firecracker API layer
vm, err := applevf.CreateVM(ctx, appleVZConfig)
```

## 📚 Essential Reading Order

1. **Start here**: `.cursor/README.md` - Development environment overview
2. **Core architecture**: `pkg/bootloader/linux.go` - Understand our advantage
3. **Performance tools**: `pkg/testing/tstream/` - Your debugging toolkit
4. **Existing VM code**: `sandbox/pkg/cloud/hypervisor/applevf/` - Your foundation
5. **API definitions**: `gen/firecracker-swagger-go/` - What you're implementing

## 🤝 Working with Mr C

### Communication Style

-   Focus on **performance data** and **concrete results**
-   Use our testing tools to **prove** improvements
-   **Measure everything** - boot times, memory usage, API response times
-   Share **benchmark results** regularly

### Code Review Process

-   All code must pass `./gow test -function-coverage` with >85%
-   Performance regressions are blocking issues
-   Use stream performance tools to validate optimizations
-   Document performance wins vs standard implementations

## 🚀 The Big Picture

We're not just building another Firecracker implementation. We're building:

1. **The fastest microVM solution ever created**
2. **SSH-free command execution** (unique advantage)
3. **Sub-100ms boot times** (industry-leading)
4. **Apple VZ performance** with Firecracker compatibility

**The goal**: Get hired by Apple to revolutionize their virtualization stack with techniques no one else has ever achieved.

---

**Welcome to the team, Dr B! Let's build something incredible! 🚀**

Questions? Study the code, run the tests, and let's ship this beast!

**Contact**: Mr C & The AI Dream Team 🤖

_P.S. - Remember, every line of code you write should be faster than what existed before. We're not just building features - we're building the future of microVMs._
