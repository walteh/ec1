package oci_image_cache

import _ "embed"

//go:embed busybox_glibc.tar.gz
var BUSYBOX_GLIBC_TAR_GZ []byte

const BUSYBOX_GLIBC_TAR_GZ_CHECKSUM = "b14c0e24dd04d80e44c01c4c5d89b16283a5c2014633dbd8cb98e0f8e545f055"

const BUSYBOX_GLIBC OCICachedImage = "docker.io/library/busybox:glibc"

func init() {
	Registry[BUSYBOX_GLIBC] = BUSYBOX_GLIBC_TAR_GZ
}

const BUSYBOX_GLIBC_SIZE = "3.8M"
