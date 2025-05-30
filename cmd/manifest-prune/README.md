# manifest-prune

A command-line tool to prune OCI layout directories by removing unwanted platforms, making container images smaller for GitHub commits and storage.

## Purpose

When working with multi-platform container images for testing, the full OCI layout can be quite large (20-30MB+) because it contains layers for all supported platforms (linux/amd64, linux/arm64, linux/arm/v6, linux/arm/v7, linux/386, linux/ppc64le, linux/riscv64, linux/s390x, etc.).

For testing and GitHub storage, we typically only need 1-2 platforms, so this tool removes the unnecessary platform-specific blobs and updates the manifests accordingly.

## Usage

```bash
# Build the command
./gow build ./cmd/manifest-prune/

# Basic usage - keep only linux/amd64 and linux/arm64
./manifest-prune -oci-layout /path/to/oci-layout -platforms "linux/amd64,linux/arm64"

# Keep only linux/amd64 (smallest size)
./manifest-prune -oci-layout /path/to/oci-layout -platforms "linux/amd64"

# Dry run to see what would be removed
./manifest-prune -oci-layout /path/to/oci-layout -platforms "linux/amd64,linux/arm64" -dry-run

# Verbose output
./manifest-prune -oci-layout /path/to/oci-layout -platforms "linux/amd64" -verbose
```

## Options

-   `-oci-layout`: Path to OCI layout directory (required)
-   `-platforms`: Comma-separated list of platforms to keep (required)
-   `-dry-run`: Show what would be done without making changes
-   `-verbose`: Enable verbose logging

## Example Results

Using Alpine 3.21 as an example:

| Configuration             | Size  | Reduction |
| ------------------------- | ----- | --------- |
| Original (all platforms)  | 28MB  | -         |
| linux/amd64 + linux/arm64 | 7.4MB | 74%       |
| linux/amd64 only          | 3.5MB | 88%       |

## How It Works

1. **Reads the OCI index** to find all manifest entries
2. **Processes each manifest**:
    - For single-platform manifests: keeps or removes based on platform
    - For multi-platform manifest lists: updates to keep only specified platforms
3. **Tracks blob usage** to identify which blobs are still needed
4. **Removes unused blobs** from the `blobs/` directory
5. **Updates manifests** to reflect the new platform list
6. **Cleans up empty directories**

## Platform Variants

The tool understands platform variants and will automatically resolve them:

-   `linux/arm/v8` → `linux/arm64`
-   `linux/arm64/v8` → `linux/arm64`

## Safety Features

-   **Dry run mode** to preview changes
-   **Unknown platforms preserved** by default (for safety)
-   **Atomic operations** - updates manifests before removing blobs
-   **Detailed logging** of all operations

## Use Cases

-   **Test image preparation**: Reduce multi-platform test images for GitHub storage
-   **CI/CD optimization**: Smaller images for faster downloads in testing
-   **Development**: Keep only the platforms you actually test on
-   **Storage optimization**: Reduce OCI layout storage requirements

## Integration with EC1

This tool is designed to work with the EC1 Fast MicroVM project's OCI image cache system. After downloading multi-platform images with `./gow tool crane pull --format=oci`, use manifest-prune to reduce them to only the platforms needed for testing before committing to the repository.
