package oci_image_cache

import _ "embed"

//go:embed alpine_socat_latest.tar.gz
var ALPINE_SOCAT_LATEST_TAR_GZ []byte

const ALPINE_SOCAT_LATEST_TAR_GZ_CHECKSUM = "7813a0c3179ff876fc90ad587ed55d20457a410ad3180ed45a1c5ed8476b85fb"

const ALPINE_SOCAT_LATEST OCICachedImage = "docker.io/alpine/socat:latest"

func init() {
	Registry[ALPINE_SOCAT_LATEST] = ALPINE_SOCAT_LATEST_TAR_GZ
}

const ALPINE_SOCAT_LATEST_SIZE = "8.3M"
