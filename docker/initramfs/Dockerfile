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

RUN make -j$(nproc) CFLAGS="-static -fno-PIE" LDFLAGS="-static -no-pie"

RUN mkdir -p /rootfs/bin
RUN cp busybox /rootfs/bin/ \
	&& chroot /rootfs /bin/busybox --install -s

RUN mkdir -p /rootfs/etc/network
RUN echo -e 'auto eth0\niface eth0 inet dhcp' > /rootfs/etc/network/interfaces


FROM scratch AS rootfs
COPY --from=alpine/socat /usr/bin/socat /bin/socat
COPY --from=busybox-build /rootfs/ /

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
