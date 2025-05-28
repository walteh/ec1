# Multi-stage build for bare bones VM kernel
FROM debian:bookworm-slim AS builder

# Install essential build dependencies
RUN apt-get update && apt-get install -y \
	git \
	make \
	gcc \
	flex \
	bison \
	libssl-dev \
	libelf-dev \
	bc \
	kmod \
	cpio \
	&& rm -rf /var/lib/apt/lists/*

ENV SRC_DIR=/usr/src \
	DIST_DIR=/dist \
	LINUX_DIR=/usr/src/linux \
	LINUX_REPO_URL=git://git.kernel.org/pub/scm/linux/kernel/git/stable/linux-stable.git

# Use stable kernel version instead of RC
ARG KERNEL_VERSION=6.15-rc7
# Allow choosing different kernel configs

RUN mkdir -p ${SRC_DIR} ${DIST_DIR} \
	&& git clone --depth 1 --branch v${KERNEL_VERSION} ${LINUX_REPO_URL} ${LINUX_DIR}

WORKDIR ${LINUX_DIR}

ARG TARGETARCH

ARG KERNEL_CONFIG=harpoon-${TARGETARCH}.config

# Copy kernel config
COPY ${KERNEL_CONFIG} .config

RUN make LOCALVERSION= olddefconfig

RUN <<EOF
if [ "${TARGETARCH}" = "arm64" ]; then
	make LOCALVERSION= -j$(nproc) Image
else
	make LOCALVERSION= -j$(nproc) bzImage
fi
EOF


# For bare bones VM, we only need the kernel image
RUN <<EOF
if [ "${TARGETARCH}" = "arm64" ]; then
	cp arch/arm64/boot/Image /boot/vmlinux
	cp .config /boot/config-${KERNEL_VERSION}
else
	cp arch/x86/boot/bzImage /boot/vmlinux
	cp .config /boot/config-${KERNEL_VERSION}
fi
EOF

# Export stage - this allows us to copy files out
FROM scratch AS export
COPY --from=builder /boot /

# Final stage with minimal footprint (for container image)
FROM scratch
COPY --from=builder /boot /boot
# Note: Removed modules copy since CONFIG_MODULES is disabled
