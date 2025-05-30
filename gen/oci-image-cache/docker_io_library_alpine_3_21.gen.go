package oci_image_cache

import _ "embed"

//go:embed docker_io_library_alpine_3_21.tar.xz
var DOCKER_IO_LIBRARY_ALPINE_3_21_TAR_XZ []byte

const DOCKER_IO_LIBRARY_ALPINE_3_21_TAR_XZ_CHECKSUM = "681d82bb7731f40f3b1698e539395aa0c2a6937ea70f7790ab357f42de5fec24"

const DOCKER_IO_LIBRARY_ALPINE_3_21_IMAGE OCICachedImage = "docker.io/library/alpine:3.21"

func init() {
	Registry[DOCKER_IO_LIBRARY_ALPINE_3_21_IMAGE] = DOCKER_IO_LIBRARY_ALPINE_3_21_TAR_XZ
}
