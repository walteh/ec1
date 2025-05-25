# ---------- 1. compile a static BusyBox with networking ----------
FROM alpine:3.21 AS busybox-build
RUN apk add --no-cache build-base wget tar linux-headers musl-dev musl-utils
ARG BUSYBOX_VER=1.36.1
WORKDIR /build
# Grab and unpack BusyBox source
RUN wget -qO- https://busybox.net/downloads/busybox-${BUSYBOX_VER}.tar.bz2 \
	| tar -xj
WORKDIR /build/busybox-${BUSYBOX_VER}

# 1  baseline defconfig
RUN make defconfig

# 2  apply overrides from repo
# COPY busybox.config .config
COPY busybox.fragment.config .config.fragment
RUN KCONFIG_ALLCONFIG=.config.fragment yes "" | make oldconfig

ARG TARGET=aarch64
# 3  compile static + strip
# Force static + no‑PIE in case your toolchain adds PIE by default
RUN make -j$(nproc) CFLAGS="-static -fno-PIE" LDFLAGS="-static -no-pie"

# --- create rootfs tree right here ---

# RUN cp busybox /rootfs/bin/
# RUN /build/busybox-${BUSYBOX_VER}/busybox --install -s \
# 		-busyboxdir=/rootfs/bin
RUN mkdir -p /rootfs/bin
RUN cp busybox /rootfs/bin/ && \
    chroot /rootfs /bin/busybox --install -s

RUN mkdir -p  /rootfs/etc/network
RUN echo -e 'auto eth0\niface eth0 inet dhcp' > /rootfs/etc/network/interfaces

# RUN mkdir -p /rootfs/lib
# RUN cp /build/busybox-${BUSYBOX_VER}/usr/lib/ld-musl-aarch64.so.1 /rootfs/lib/
# RUN ldd busybox | awk '{print $3}' | xargs -I {} cp {} /rootfs/lib/

######## export stage – *no* RUN lines, so no shell required ########
FROM scratch AS rootfs
COPY --from=busybox-build /rootfs/ /

# ---------- 2. assemble rootfs ----------
# FROM scratch AS rootfs
# COPY --from=busybox-build /build/busybox-*/busybox /bin/busybox
# SHELL ["/bin/busybox", "sh", "-c"]
# RUN /bin/busybox --install -s

# # --- your extras ---
# COPY init /init
# # COPY vsock-agent /usr/bin/agent example
# COPY udhcpc.default /etc/udhcpc/default.script
# # minimal NIC config so udhcpc works inside VM
# RUN mkdir -p /etc/network \
# 	&& echo "auto eth0\niface eth0 inet dhcp" > /etc/network/interfaces

# CMD ["[]"]

# ---------- 3. pack with Go cpio writer ----------
FROM alpine:3.21 AS cpio
RUN apk add --no-cache cpio gzip
WORKDIR /work

# copy populated rootfs from previous stage
COPY --from=rootfs / /rootfs

COPY init /rootfs/init
COPY udhcpc.default /rootfs/etc/udhcpc/default.script
COPY resolv.conf /rootfs/etc/resolv.conf

RUN mkdir -p /dist \
	&& (cd /rootfs && find . -print0 | cpio --null -ov --format=newc > /dist/initramfs.cpio) \
	&& gzip -1 /dist/initramfs.cpio

# ---------- 4. export stage ----------
FROM scratch AS export
COPY --from=cpio /dist/initramfs.cpio.gz /initramfs.cpio.gz
# (optional) raw rootfs for inspection
# COPY --from=rootfs / /rootfs
