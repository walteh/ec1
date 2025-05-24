# EC1 Code Signing Tool

A comprehensive code signing tool for Go developers working with Apple frameworks and macOS security requirements.

## Features

-   **Flexible entitlements**: Support for common Apple entitlements with friendly names
-   **Multiple operation modes**: Sign, execute, and test modes
-   **Backward compatibility**: Maintains compatibility with existing `gow` integration
-   **Dry-run support**: See what would be done without executing
-   **Verbose logging**: Detailed operation logging with structured output

## Quick Start

### List Available Entitlements

```bash
go tool github.com/walteh/ec1/tools/cmd/codesign --list-entitlements
```

### Sign a Binary

```bash
# Sign with virtualization entitlement
go tool github.com/walteh/ec1/tools/cmd/codesign -mode=sign -target=./myapp -entitlement=virtualization

# Sign with multiple entitlements
go tool github.com/walteh/ec1/tools/cmd/codesign -mode=sign -target=./myapp \
  -entitlement=virtualization \
  -entitlement=network-client \
  -entitlement=hypervisor
```

### Run Tests with Code Signing

```bash
# Run tests with automatic code signing
go tool github.com/walteh/ec1/tools/cmd/codesign -mode=test \
  -entitlement=virtualization \
  -entitlement=network-client \
  -- -v ./pkg/vmnet
```

### Execute with Code Signing

```bash
# Sign and execute a binary
go tool github.com/walteh/ec1/tools/cmd/codesign -mode=exec \
  -entitlement=virtualization \
  -- ./myapp arg1 arg2
```

## Common Entitlements

The tool provides friendly names for common Apple entitlements:

| Friendly Name                | Apple Entitlement                                        |
| ---------------------------- | -------------------------------------------------------- |
| `virtualization`             | `com.apple.security.virtualization`                      |
| `hypervisor`                 | `com.apple.security.hypervisor`                          |
| `network-client`             | `com.apple.security.network.client`                      |
| `network-server`             | `com.apple.security.network.server`                      |
| `allow-jit`                  | `com.apple.security.cs.allow-jit`                        |
| `allow-unsigned-executable`  | `com.apple.security.cs.allow-unsigned-executable-memory` |
| `disable-library-validation` | `com.apple.security.cs.disable-library-validation`       |

You can also use full Apple entitlement identifiers directly.

## Modes

### Sign Mode (`-mode=sign`)

Signs a binary with specified entitlements. Requires `-target` parameter.

### Exec Mode (`-mode=exec`)

Signs a binary and then executes it with provided arguments. Perfect for one-off execution.

### Test Mode (`-mode=test`)

Designed for `go test` integration. Acts as an exec wrapper that signs test binaries before execution.

## Integration with Go Workflows

### With `gow` (Legacy Compatibility)

The tool maintains backward compatibility with the existing `gow` integration:

```bash
# These still work as before
./gow test -codesign -v ./pkg/vmnet
```

### Direct with `go test`

```bash
# Use the tool directly as a test exec wrapper
go test -exec="go tool github.com/walteh/ec1/tools/cmd/codesign -mode=exec -entitlement=virtualization --" ./pkg/vmnet
```

### Advanced Test Scenarios

```bash
# Test with multiple entitlements and verbose output
go tool github.com/walteh/ec1/tools/cmd/codesign -mode=test -verbose \
  -entitlement=virtualization \
  -entitlement=network-client \
  -entitlement=hypervisor \
  -- -v -race ./...
```

## Command Line Options

-   `-mode`: Operation mode (`sign`, `exec`, `test`) [default: `sign`]
-   `-target`: File or binary to sign (required for sign mode)
-   `-entitlement`: Entitlement to add (can be repeated)
-   `-identity`: Code signing identity [default: `-` for ad-hoc signing]
-   `-force`: Force re-signing even if already signed
-   `-verbose`: Enable verbose logging
-   `-dry-run`: Show what would be done without executing
-   `-list-entitlements`: List available entitlements and exit

## Examples for EC1 Development

### Testing VZ Integration

```bash
# Test VZ virtualization with proper entitlements
go tool github.com/walteh/ec1/tools/cmd/codesign -mode=test \
  -entitlement=virtualization \
  -entitlement=hypervisor \
  -- -v ./pkg/vmnet
```

### Testing Network Components

```bash
# Test networking with client/server entitlements
go tool github.com/walteh/ec1/tools/cmd/codesign -mode=test \
  -entitlement=network-client \
  -entitlement=network-server \
  -- -v ./pkg/gvnet
```

### Debug Mode with JIT

```bash
# Enable JIT for debugging scenarios
go tool github.com/walteh/ec1/tools/cmd/codesign -mode=test \
  -entitlement=virtualization \
  -entitlement=allow-jit \
  -entitlement=disable-library-validation \
  -- -v -gcflags="all=-N -l" ./pkg/vmm
```

## Troubleshooting

### Verbose Output

Use `-verbose` to see detailed logging of all operations.

### Dry Run

Use `-dry-run` to see what commands would be executed without actually running them.

### Legacy Mode Issues

If you encounter issues with the legacy mode (used by `gow`), the tool automatically detects and handles the old argument format.

### Entitlement Issues

Use `--list-entitlements` to see all available entitlement names and their corresponding Apple identifiers.
