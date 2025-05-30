package oci_image_cache

import _ "embed"

//go:embed debian_bookworm_slim.tar.gz
var DEBIAN_BOOKWORM_SLIM_TAR_GZ []byte

const DEBIAN_BOOKWORM_SLIM_TAR_GZ_CHECKSUM = "2875b0f9af753898c9bcf3f0866a0d39aac6074f27b4850722bc9d849d0ef253"

const DEBIAN_BOOKWORM_SLIM OCICachedImage = "docker.io/library/debian:bookworm-slim"

func init() {
	Registry[DEBIAN_BOOKWORM_SLIM] = DEBIAN_BOOKWORM_SLIM_TAR_GZ
}

const DEBIAN_BOOKWORM_SLIM_SIZE = "54M"
