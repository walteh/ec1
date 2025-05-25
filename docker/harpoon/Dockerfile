 
FROM ubuntu:24.10

# Prevent interactive prompts during package installation
ENV DEBIAN_FRONTEND=noninteractive

# Install build dependencies
RUN apt-get update && apt-get install -y \
	libncurses-dev gawk flex bison openssl libssl-dev dkms libelf-dev libudev-dev libpci-dev libiberty-dev autoconf llvm

RUN apt-get install -y \
	curl \
	bc

# Set working directory
WORKDIR /build

# Copy the pui-pui project
COPY . .

# Make the build tool executable
RUN chmod +x puipui-linux-tool

# Set environment variables for reproducible builds
ENV SOURCE_DATE_EPOCH=1640995200

# Build the kernel and initramfs
RUN ./puipui-linux-tool

# Create output directory and copy artifacts
RUN mkdir -p /output \
	&& find . -name "puipui_linux_v*.tar.gz" -exec cp {} /output/ \; \
	&& find . -path "*/artifact/*" -type f -exec cp {} /output/ \;

# Export stage - allows copying files out of the container
FROM scratch AS export
COPY --from=0 /output /

# Final stage with just the artifacts
FROM ubuntu:24.10 AS final
COPY --from=0 /output /artifacts
WORKDIR /artifacts

# Show what was built
RUN ls -la /artifacts

CMD ["bash", "-c", "echo", "PUI PUI Linux build complete. Artifacts:", "&&", "ls", "-la", "/artifacts"]
