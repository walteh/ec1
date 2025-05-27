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


RUN mkdir -p /rootfs/bin /rootfs/sbin /rootfs/usr/sbin /rootfs/usr/bin /rootfs/usr/local/bin
RUN cp busybox /rootfs/bin/ \
	&& chroot /rootfs /bin/busybox --install -s

RUN mkdir -p /rootfs/etc/network
RUN echo -e 'auto eth0\niface eth0 inet dhcp' > /rootfs/etc/network/interfaces
RUN apk add --no-cache file

# download jq
FROM debian:bookworm-slim AS jq-build

ENV DEBIAN_FRONTEND=noninteractive \
	DEBCONF_NONINTERACTIVE_SEEN=true \
	LC_ALL=C.UTF-8 \
	LANG=C.UTF-8

RUN apt-get update \
	&& apt-get install -y \
		build-essential \
		autoconf \
		libtool \
		git \
		bison \
		flex \
		wget \
	&& rm -rf /var/lib/apt/lists/*

RUN git clone --depth 1 --branch master https://github.com/jqlang/jq.git /app
## have to skip building docs but I don't need them for my use case
RUN cd /app \
	&& git submodule init \
	&& git submodule update \
	&& autoreconf -i \
	&& ./configure --disable-valgrind --enable-all-static --disable-docs --disable-maintainer-mode --prefix=/usr/local \
	&& make -j$(nproc) LDFLAGS=-all-static \
	&& strip jq

# FROM alpine:3.21 AS jq-build

# RUN apk add --no-cache \
# 	git \
# 	autoconf \
# 	automake \
# 	libtool \
# 	build-base \
# 	bison \
# 	flex \
# 	wget

# RUN git clone https://github.com/jqlang/jq.git --depth 1 /app

# RUN cd /app && git submodule update --init \
# 	&& autoreconf -i \
# 	&& ./configure --disable-valgrind --enable-all-static --disable-docs --disable-maintainer-mode --prefix=/usr/local \
# 	&& make -j$(nproc) LDFLAGS=-all-static \
# 	&& strip jq
# testing onigurama will fail, this line just shows the error
RUN cd /app && make check || echo "check failed"

FROM scratch AS rootfs
COPY --from=alpine/socat /usr/bin/socat /bin/socat
COPY --from=busybox-build /usr/bin/file /bin/file
COPY --from=busybox-build /rootfs/ /
COPY --from=jq-build /app/jq /bin/jq

# ---------- 3. pack with Go cpio writer ----------
FROM alpine:3.21 AS cpio
RUN apk add --no-cache cpio gzip
WORKDIR /work

# copy populated rootfs from previous stage
COPY --from=rootfs / /rootfs

COPY init /rootfs/init
COPY ec1-container-init /rootfs/ec1-container-init
COPY udhcpc.default /rootfs/etc/udhcpc/default.script
COPY resolv.conf /rootfs/etc/resolv.conf

# Make ec1-container-init executable
RUN chmod +x /rootfs/ec1-container-init

RUN mkdir -p /dist \
	&& (cd /rootfs && find . -print0 | cpio --null -ov --format=newc > /dist/initramfs.cpio) \
	&& gzip -1 /dist/initramfs.cpio

# ---------- 4. export stage ----------
FROM scratch AS export
COPY --from=cpio /dist/initramfs.cpio.gz /initramfs.cpio.gz
