package oci_image_cache

import _ "embed"

//go:embed oven_bun_alpine.tar.gz
var OVEN_BUN_ALPINE_TAR_GZ []byte

const OVEN_BUN_ALPINE_TAR_GZ_CHECKSUM = "124e8b57c09e14dc489df84f11519ee722bb6d241335834d9f0f7607d4c5c050"

const OVEN_BUN_ALPINE OCICachedImage = "docker.io/oven/bun:alpine"

func init() {
	Registry[OVEN_BUN_ALPINE] = OVEN_BUN_ALPINE_TAR_GZ
}

const OVEN_BUN_ALPINE_SIZE = "83M"
