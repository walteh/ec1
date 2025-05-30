package oci_image_cache

import _ "embed"

//go:embed alpine_socat_latest.tar.gz
var ALPINE_SOCAT_LATEST_TAR_GZ []byte

const ALPINE_SOCAT_LATEST_TAR_GZ_CHECKSUM = "a1c82627d9f23df6f263a22e3039fd9b68b58d36631efaac395059dd8ab1b961"

const ALPINE_SOCAT_LATEST OCICachedImage = "docker.io/alpine/socat:latest"

func init() {
	Registry[ALPINE_SOCAT_LATEST] = ALPINE_SOCAT_LATEST_TAR_GZ
}

const ALPINE_SOCAT_LATEST_SIZE = "8.3M"
