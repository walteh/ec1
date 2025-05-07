#!/bin/bash
set -e

DISK_PATH="/Users/usr/Developer/github/walteh/ec1/build/nocloud_alpine-3.21.2-aarch64-uefi-cloudinit-r0.qcow2"
EFI_PATH="./bin/vz-vs.fd" # Using OVMF instead of the custom vz EFI store

# Run QEMU with similar settings to what we're using with vz
# Run QEMU with modified settings to fix the serial console issue
qemu-system-aarch64 \
	-M virt \
	-accel hvf \
	-cpu host \
	-smp 2 \
	-m 2G \
	-drive file=${DISK_PATH},format=qcow2 \
	-nographic \
	-serial stdio \
	-monitor none \
	-d unimp,guest_errors \
	-D qemu_debug.log \
	-boot menu=on

# If the above fails, you might need OVMF firmware files
# brew install ovmf
# or download them from: https://github.com/tianocore/edk2/releases
