# Harpoon Kernel Configuration

This directory contains the kernel configuration for the Harpoon microVM project, organized using **config fragments** to eliminate duplication and improve maintainability.

## Configuration Structure

### Base Configuration

-   **`harpoon-base.config`** - Common configuration shared between all architectures
    -   Essential microVM features (virtio, networking, filesystems)
    -   Security features (SECCOMP, stack protection, VMAP_STACK)
    -   Minimal driver set for fast boot times
    -   Disabled unnecessary features (graphics, USB, most hardware drivers)

### Architecture-Specific Fragments

-   **`harpoon-amd64.fragment`** - x86_64-specific settings

    -   x86_64 security features (SMAP, SMEP, UMIP, CET, IBT)
    -   8250 serial console support
    -   x86_64 timer and CPU features

-   **`harpoon-arm64.fragment`** - ARM64-specific settings
    -   ARM64 security features (PAN, PTR_AUTH, BTI, HW_AFDBM)
    -   AMBA PL011 serial console support
    -   ARM64-specific errata and timer configurations

## Build Process

The Docker build process automatically merges the base configuration with the appropriate architecture fragment:

```dockerfile
# Copy base config and architecture-specific fragment
COPY harpoon-base.config ./
COPY harpoon-${TARGETARCH}.fragment ./

# Merge base config with architecture-specific fragment
RUN scripts/kconfig/merge_config.sh -m harpoon-base.config harpoon-${TARGETARCH}.fragment
```

## Local Development

For local development and testing, use the provided helper script:

```bash
# Inside the kernel build container
./merge-config.sh amd64    # Merge config for AMD64
./merge-config.sh arm64    # Merge config for ARM64
```

## Adding New Configuration Options

### For All Architectures

Add new options to `harpoon-base.config`.

### For Specific Architecture

Add new options to the appropriate `harpoon-{arch}.fragment` file.

### Creating New Architecture Support

1. Create a new fragment file: `harpoon-{newarch}.fragment`
2. Add architecture-specific configurations
3. The build system will automatically use it when `TARGETARCH={newarch}`

## Benefits of This Approach

1. **No Duplication** - Common settings are defined once in the base config
2. **Easy Maintenance** - Architecture-specific changes only affect their respective fragments
3. **Extensible** - Easy to add new architectures or configuration variants
4. **Standard Tooling** - Uses the Linux kernel's built-in `merge_config.sh` script
5. **Clear Separation** - Architecture-specific concerns are isolated

## Legacy Files

The old monolithic config files (`harpoon-amd64.config`, `harpoon-arm64.config`) can be removed once this new approach is validated.
