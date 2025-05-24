# 🤖 Dr B Background Agent Setup Guide

## 🚀 READY TO ACTIVATE DR B!

Everything is configured for Cursor Background Agents. Here's how to launch Dr B:

## 📋 Pre-Activation Checklist

### ✅ Environment Ready

-   [x] **environment.json** configured with Go development environment
-   [x] **install.sh** script ready (complete Go setup + gow tool)
-   [x] **start.sh** script ready (development environment activation)
-   [x] **Terminal configurations** for Firecracker development
-   [x] **Development aliases** for maximum efficiency
-   [x] **Performance monitoring** terminals configured

### ✅ Mission Documents Ready

-   [x] **Dr B agent instructions** (`.cursor/dr-b-agent-instructions.md`)
-   [x] **Complete onboarding** (`docs/dr-b-onboarding.md`)
-   [x] **Development environment** (`.cursor/README.md`)
-   [x] **Firecracker rules** (`.cursor/rules/firecracker-integration.mdc`)

## 🎯 Activation Steps

### 1. Enable Background Agents in Cursor

-   **Turn OFF privacy mode** (required for background agents)
-   Ensure you have **Max Mode** access (required for background agents)

### 2. Launch Background Agent

```
Cmd + ' (or Ctrl + ')  # Open background agents list
```

### 3. GitHub Setup (First Time)

-   Grant **read-write access** to `github.com/walteh/ec1` repo
-   Background agents need to clone and push branches

### 4. Launch Dr B with This Prompt:

```
🔥 I am Dr B, your background agent for the EC1 Fast MicroVM project!

My mission: Build the world's fastest Firecracker-compatible microVM implementation using our Apple VZ backend + init injection system.

Key objectives:
- Enhance the MAIN Firecracker API implementation in pkg/firecracker/
- Achieve <100ms boot times with SSH-free command execution
- Maintain >85% function coverage using our gow tool
- Build on existing Apple VZ foundation

I'll start by:
1. Enhancing the existing Firecracker API in pkg/firecracker/api.go (NOT sandbox!)
2. Implementing missing endpoints and improving VM lifecycle
3. Adding performance optimization with our stream testing tools

🚨 CRITICAL: I work in pkg/firecracker/ - the MAIN implementation, NOT sandbox/!

Please refer to .cursor/dr-b-agent-instructions.md for my complete mission brief!

Ready to revolutionize microVMs! 🚀
```

### 5. Monitor Dr B's Progress

```
Cmd + ; (or Ctrl + ;)  # View agent status and enter machine
```

## 🎯 What Dr B Will Do

### Week 1 Sprint Plan

-   **Days 1-2**: API Foundation

    -   Create Firecracker REST API structure
    -   Implement health endpoints (`/ping`)
    -   Add machine configuration (`/machine-config`)

-   **Days 3-4**: VM Integration

    -   VM lifecycle using Apple VZ backend
    -   Init injection system integration
    -   Boot time measurement

-   **Days 5-7**: Performance Optimization
    -   Stream performance testing integration
    -   Benchmark critical paths
    -   Optimize configuration translation

### Expected Deliverables

-   [ ] Firecracker API responding correctly
-   [ ] <100ms boot time consistently
-   [ ] > 85% function coverage on all code
-   [ ] Integration tests passing
-   [ ] Performance benchmarks documented

## 🔧 Dr B's Development Environment

### Tools Available

-   **GOW wrapper**: `./gow` (2x faster than alternatives)
-   **Stream performance testing**: Automatic bottleneck detection
-   **Function coverage**: Built-in >85% enforcement
-   **Development aliases**: Mission-specific shortcuts

### Key Workspaces

```bash
/workspace/pkg/firecracker/                           # PRIMARY workspace (MAIN implementation!)
/workspace/pkg/bootloader/                            # Init injection system
/workspace/pkg/testing/tstream/                       # Performance tools
/workspace/pkg/vmm/                                   # VM management foundation
```

**⚠️ CRITICAL**: Work in `pkg/firecracker/` NOT `sandbox/` - sandbox is just for testing old code!

### Performance Monitoring

-   **Terminal 1**: Stream performance validation
-   **Terminal 2**: Firecracker workspace
-   **Terminal 3**: Continuous performance monitoring

## 🎯 Success Metrics for Dr B

### Technical Targets

-   **Boot time**: <100ms from API call to VM ready
-   **API compatibility**: 100% Firecracker REST API compatible
-   **Test coverage**: >85% function coverage
-   **Performance**: Documented improvements vs standard Firecracker

### Communication Expectations

Dr B will provide:

-   Performance benchmark data
-   Function coverage reports
-   API compatibility status
-   Architecture decisions with rationale

## 🚨 Critical Success Factors

### What Makes Dr B Successful

1. **Uses our performance tools** - Stream testing catches bottlenecks
2. **Builds on Apple VZ** - Don't reinvent VM management
3. **Maintains API compatibility** - Must work with existing clients
4. **Leverages init injection** - Our secret weapon for speed!

### Red Flags to Watch For

-   Bypassing performance testing tools
-   Reinventing VM management instead of extending
-   Breaking Firecracker API compatibility
-   Ignoring test coverage requirements

## 🔥 The Secret Weapon

**Init Injection Advantage**: Every other microVM uses SSH for command execution. We execute directly through embedded gRPC init. This eliminates network overhead and gives us unprecedented performance.

Dr B should exploit this advantage in every design decision!

## 🚀 Launch Sequence

**You're ready to activate Dr B!**

1. Turn off privacy mode in Cursor
2. Hit `Cmd + '` to open background agents
3. Paste the launch prompt above
4. Watch Dr B revolutionize microVMs!

**Target**: Build the fastest microVM implementation ever created and land that Apple job! 🍎

---

**Status**: 🔥 READY FOR LAUNCH! 🚀
