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
COPY busybox.fragment.config .config.fragment
RUN KCONFIG_ALLCONFIG=.config.fragment yes "" | make oldconfig

ARG TARGET=aarch64

RUN make -j$(nproc) CFLAGS="-static -fno-PIE" LDFLAGS="-static -no-pie"


RUN mkdir -p /rootfs/bin /rootfs/sbin /rootfs/usr/sbin /rootfs/usr/bin /rootfs/usr/local/bin
RUN cp busybox /rootfs/bin/ \
	&& chroot /rootfs /bin/busybox --install -s

RUN mkdir -p /rootfs/etc/network
RUN echo -e 'auto eth0\niface eth0 inet dhcp' > /rootfs/etc/network/interfaces

# RUN apk add --no-cache file
FROM debian:bookworm-slim AS debian-build

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

# download jq
FROM debian-build AS jq-build

RUN git clone --depth 1 --branch master https://github.com/jqlang/jq.git /app
## have to skip building docs but I don't need them for my use case
RUN cd /app \
	&& git submodule init \
	&& git submodule update \
	&& autoreconf -i \
	&& ./configure --disable-valgrind --enable-all-static --disable-docs --disable-maintainer-mode --prefix=/usr/local \
	&& make -j$(nproc) LDFLAGS=-all-static \
	&& strip jq


RUN cd /app && make check || echo "check failed"

# Add this new build stage after the e2fsprogs-build stage
FROM debian-build AS lshw-build


RUN git clone https://github.com/lyonel/lshw.git /src
WORKDIR /src
# Build statically linked lshw
RUN make static
RUN strip src/lshw-static
RUN cp src/lshw-static /bin/lshw

FROM alpine:3.21 AS e2fsprogs-build
RUN apk add --no-cache build-base git autoconf automake linux-headers musl-dev
RUN git clone https://git.kernel.org/pub/scm/fs/ext2/e2fsprogs.git /src
WORKDIR /src
# Pass STATIC LDFLAGS so no interpreter is needed
RUN CFLAGS='-O2' \
	LDFLAGS='-static -s' \
	./configure --enable-static --disable-shared
RUN make -j$(nproc)
RUN strip misc/mke2fs
RUN cp misc/mke2fs /bin/

# FROM alpine:3.21 AS util-linux-build
# RUN apk add --no-cache build-base git autoconf automake linux-headers musl-dev util-linux-dev gettext-dev ncurses-dev libtool-2 bison flex
# RUN git clone https://git.kernel.org/pub/scm/utils/util-linux/util-linux.git /src
# WORKDIR /src
# # Checkout a stable release
# RUN git checkout v2.40.2
# RUN ./autogen.sh
# # Configure for static build with minimal dependencies and most useful utilities
# RUN CFLAGS='-O2' \
# 	LDFLAGS='-static -s' \
# 	./configure \
# 	--enable-static \
# 	--disable-shared \
# 	--disable-nls \
# 	--disable-rpath \
# 	--without-python \
# 	--without-systemd \
# 	--without-systemdsystemunitdir \
# 	--disable-makeinstall-chown \
# 	--disable-makeinstall-setuid \
# 	--disable-bash-completion \
# 	--disable-colors-default \
# 	--enable-static-programs=mount,umount,lsblk,fdisk,findmnt,losetup,mkswap,swapon,swapoff,dmesg,fsck,blkid,blockdev,fstrim,lscpu,lsmem,hwclock,sfdisk
# RUN make -j$(nproc)
# # Copy the most useful utilities to /bin
# RUN mkdir -p /bin && \
# 	for util in mount umount lsblk fdisk findmnt losetup mkswap swapon swapoff dmesg fsck blkid blockdev fstrim lscpu lsmem hwclock sfdisk; do \
# 		if [ -f "$util" ]; then cp "$util" /bin/; fi; \
# 		if [ -f "sys-utils/$util" ]; then cp "sys-utils/$util" /bin/; fi; \
# 		if [ -f "disk-utils/$util" ]; then cp "disk-utils/$util" /bin/; fi; \
# 		if [ -f "misc-utils/$util" ]; then cp "misc-utils/$util" /bin/; fi; \
# 		if [ -f "sys-utils/$util.static" ]; then cp "sys-utils/$util.static" /bin/"$util"; fi; \
# 		if [ -f "disk-utils/$util.static" ]; then cp "disk-utils/$util.static" /bin/"$util"; fi; \
# 		if [ -f "misc-utils/$util.static" ]; then cp "misc-utils/$util.static" /bin/"$util"; fi; \
# 	done
# # Strip all binaries
# RUN find /bin -type f -executable -exec strip {} \;

FROM scratch AS rootfsd
COPY --from=alpine/socat /usr/bin/socat /bin/socat
COPY --from=e2fsprogs-build /bin/mke2fs /bin/mke2fs
COPY --from=lshw-build /bin/lshw /hbin/lshw
# COPY --from=util-linux-build /bin/* /hbin/
COPY --from=harpoond /harpoond /bin/harpoond
COPY --from=busybox-build /rootfs/ /


# ---------- 3. pack with Go cpio writer ----------
FROM alpine:3.21 AS cpio
RUN apk add --no-cache cpio gzip
WORKDIR /work

# copy populated rootfs from previous stage
COPY --from=rootfsd / /rootfs

COPY udhcpc.default /rootfs/etc/udhcpc/default.script
COPY resolv.conf /rootfs/etc/resolv.conf

RUN chroot /rootfs ln -s bin/harpoond /init

RUN mkdir -p /dist \
	&& (cd /rootfs && find . -print0 | cpio --null -ov --format=newc > /dist/initramfs.cpio) \
	&& gzip -1 /dist/initramfs.cpio

# ---------- 4. export stage ----------
FROM scratch AS export
COPY --from=cpio /dist/initramfs.cpio.gz /initramfs.cpio.gz
