# Containerd Shim Testing & Validation Plan

## Overview

This document outlines the comprehensive testing strategy for validating the Harpoon containerd shim integration. The goal is to ensure that containers run seamlessly in microVMs through the containerd interface.

## Architecture Summary

-   **1 VM per Container**: Each container gets its own dedicated microVM
-   **VMM Integration**: Uses `pkg/vmm` with Apple Virtualization Framework backend
-   **Init Injection**: Commands executed via gRPC over vsock (no SSH overhead)
-   **Containerd Compatibility**: Implements containerd shim v2 protocol

## Testing Phases

### Phase 1: Basic Functionality ✅

**Objective**: Verify the shim can be built and basic lifecycle works

**Commands**:

```bash
# Build the shim
cd cmd/containerd-shim-harpoon-v1
./gow build -o containerd-shim-harpoon-v1 .

# Verify binary
./containerd-shim-harpoon-v1 --help
```

**Success Criteria**:

-   [x] Shim builds without errors (226MB binary, ~6-8s build time)
-   [x] Binary executes and shows help (exits properly, no hanging)
-   [x] No import or compilation issues
-   [x] Performance targets met (build <30s, startup <1s)
-   [x] Code signing works on Apple Silicon
-   [x] 11/11 integration tests passing

### Phase 2: Containerd Integration 🔄

**Objective**: Test shim registration and basic container operations

**Setup**:

```bash
# Terminal 1: Start containerd
sudo containerd --config /etc/containerd/config.toml --log-level debug

# Terminal 2: Configure containerd to use our shim
sudo mkdir -p /etc/containerd
cat << EOF | sudo tee /etc/containerd/config.toml
version = 2

[plugins."io.containerd.grpc.v1.cri".containerd.runtimes.harpoon]
  runtime_type = "io.containerd.harpoon.v1"

[plugins."io.containerd.grpc.v1.cri".containerd.runtimes.harpoon.options]
  BinaryName = "$(pwd)/containerd-shim-harpoon-v1"
EOF
```

**Test Commands**:

```bash
# Pull a test image
sudo ctr images pull docker.io/library/alpine:latest

# Create container with our runtime
sudo ctr run --runtime io.containerd.harpoon.v1 \
  docker.io/library/alpine:latest \
  test-harpoon \
  /bin/sh -c "echo 'Hello from Harpoon VM!'; sleep 5"
```

**Success Criteria**:

-   [ ] Containerd recognizes the shim
-   [ ] Container creation succeeds
-   [ ] VM boots and executes command
-   [ ] Output appears correctly
-   [ ] Container exits cleanly

### Phase 3: VM Lifecycle Validation 🔄

**Objective**: Verify VM creation, execution, and cleanup

**Test Scenarios**:

1. **VM Creation**:

```bash
# Monitor VM creation
sudo ctr run --runtime io.containerd.harpoon.v1 alpine:latest vm-test echo "VM created"
```

2. **Command Execution**:

```bash
# Test various commands
sudo ctr run --runtime io.containerd.harpoon.v1 alpine:latest cmd-test /bin/sh -c "
  echo 'Testing command execution'
  ls -la /
  ps aux
  cat /proc/meminfo | head -5
"
```

3. **Process Management**:

```bash
# Test exec into running container
sudo ctr run --runtime io.containerd.harpoon.v1 -d alpine:latest long-running sleep 30
sudo ctr tasks exec --exec-id shell1 -t long-running /bin/sh
```

4. **Cleanup**:

```bash
# Verify proper cleanup
sudo ctr tasks kill long-running
sudo ctr containers delete long-running
```

**Success Criteria**:

-   [ ] VM boots in <100ms (target)
-   [ ] Commands execute correctly
-   [ ] Exit codes propagate properly
-   [ ] VM resources are cleaned up
-   [ ] No resource leaks

### Phase 4: I/O and Networking 🔄

**Objective**: Test container I/O streams and network connectivity

**Test Commands**:

```bash
# Test stdin/stdout
echo "Hello VM" | sudo ctr run --runtime io.containerd.harpoon.v1 alpine:latest io-test cat

# Test stderr
sudo ctr run --runtime io.containerd.harpoon.v1 alpine:latest stderr-test /bin/sh -c "echo 'error' >&2"

# Test networking (if implemented)
sudo ctr run --runtime io.containerd.harpoon.v1 alpine:latest net-test ping -c 3 8.8.8.8
```

**Success Criteria**:

-   [ ] stdin/stdout/stderr work correctly
-   [ ] Network connectivity functions
-   [ ] Port forwarding works (if implemented)

### Phase 5: Error Handling 🔄

**Objective**: Verify graceful error handling and recovery

**Test Scenarios**:

```bash
# Test invalid commands
sudo ctr run --runtime io.containerd.harpoon.v1 alpine:latest error-test /nonexistent/command

# Test resource limits
sudo ctr run --runtime io.containerd.harpoon.v1 alpine:latest memory-test /bin/sh -c "
  # Try to allocate large amount of memory
  dd if=/dev/zero of=/dev/null bs=1M count=1000
"

# Test signal handling
sudo ctr run --runtime io.containerd.harpoon.v1 alpine:latest signal-test sleep 100 &
sudo ctr tasks kill signal-test SIGTERM
```

**Success Criteria**:

-   [ ] Invalid commands return proper exit codes
-   [ ] Resource limits are enforced
-   [ ] Signals are handled correctly
-   [ ] Error messages are informative

### Phase 6: Performance Validation 🔄

**Objective**: Measure and validate performance targets

**Benchmarks**:

```bash
# Boot time measurement
time sudo ctr run --runtime io.containerd.harpoon.v1 alpine:latest boot-test echo "booted"

# Command execution overhead
time sudo ctr run --runtime io.containerd.harpoon.v1 alpine:latest exec-test /bin/true

# Memory usage
sudo ctr run --runtime io.containerd.harpoon.v1 alpine:latest mem-test /bin/sh -c "
  cat /proc/meminfo
  free -h
"
```

**Performance Targets**:

-   [ ] VM boot time: <100ms
-   [ ] Command execution overhead: <10ms
-   [ ] Memory footprint: <50MB idle
-   [ ] Function coverage: >85%

### Phase 7: Integration Testing 🔄

**Objective**: Test with higher-level tools (nerdctl, buildkit)

**Test Commands**:

```bash
# Test with nerdctl
nerdctl --runtime io.containerd.harpoon.v1 run --rm alpine:latest echo "nerdctl works"

# Test image building (if buildkit integration works)
nerdctl --runtime io.containerd.harpoon.v1 build -t test-image .

# Test multi-container scenarios
nerdctl --runtime io.containerd.harpoon.v1 run -d --name web nginx:alpine
nerdctl --runtime io.containerd.harpoon.v1 run --rm alpine:latest wget -O- http://web
```

**Success Criteria**:

-   [ ] nerdctl commands work
-   [ ] Image building succeeds
-   [ ] Multi-container networking works

## Debugging and Monitoring

### Log Locations

```bash
# Containerd logs
sudo journalctl -f -u containerd

# Shim logs (if implemented)
tail -f /var/log/harpoon-shim.log

# VM console logs
find /tmp -name "console.log" -exec tail -f {} \;
```

### Debug Commands

```bash
# List all containers
sudo ctr containers list

# Check task status
sudo ctr tasks list

# Get detailed container info
sudo ctr containers info <container-id>

# Check shim processes
ps aux | grep containerd-shim-harpoon

# Monitor VM processes
ps aux | grep qemu  # or equivalent for VZ
```

### Common Issues and Solutions

1. **Shim not found**:

    - Verify binary path in containerd config
    - Check file permissions
    - Ensure binary is executable

2. **VM boot failures**:

    - Check virtualization support
    - Verify image cache setup
    - Monitor VM console logs

3. **Command execution timeouts**:
    - Check vsock connectivity
    - Verify init process is running
    - Monitor network setup

## Automated Testing

### Unit Tests

```bash
# Run VMM unit tests
./gow test -function-coverage ./pkg/vmm/...

# Run shim unit tests
./gow test -function-coverage ./cmd/containerd-shim-harpoon-v1/...
```

### Integration Tests

```bash
# Run full integration test suite
./cmd/containerd-shim-harpoon-v1/test_shim.sh

# Run performance benchmarks
./gow test -bench=. ./pkg/vmm/...
```

## Success Definition

The containerd shim integration is considered successful when:

1. ✅ **Build Success**: Shim compiles without errors
2. 🔄 **Basic Functionality**: Containers can be created, started, and stopped
3. 🔄 **Performance**: Meets boot time and execution overhead targets
4. 🔄 **Compatibility**: Works with standard containerd tools (ctr, nerdctl)
5. 🔄 **Reliability**: Handles errors gracefully and cleans up resources
6. 🔄 **Integration**: Compatible with buildkit and other ecosystem tools

## Next Steps

After successful validation:

1. **Optimization**: Improve boot times and memory usage
2. **Features**: Add volume mounts, networking, port forwarding
3. **Documentation**: Create user guides and API documentation
4. **CI/CD**: Set up automated testing pipeline
5. **Distribution**: Package for easy installation (Homebrew, etc.)

## Commands for Claude/Cursor

To help with validation, use these commands:

```bash
# Quick build and test
cd cmd/containerd-shim-harpoon-v1 && ./gow build . && ./test_shim.sh

# Run specific test
sudo ctr run --runtime io.containerd.harpoon.v1 alpine:latest test-$(date +%s) echo "test"

# Check logs
sudo journalctl -f -u containerd | grep harpoon

# Monitor VMs
watch -n 1 'ps aux | grep -E "(containerd-shim-harpoon|vm-)"'
```

This testing plan provides a systematic approach to validating the containerd shim integration and ensures all critical functionality works correctly.
