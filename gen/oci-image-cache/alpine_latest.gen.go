package oci_image_cache

import _ "embed"

//go:embed alpine_latest.tar.gz
var ALPINE_LATEST_TAR_GZ []byte

const ALPINE_LATEST_TAR_GZ_CHECKSUM = "52e2de4a6c7cf96d61c279ac6feb6867325c92d90505d160628f1fd000a5c72b"

const ALPINE_LATEST OCICachedImage = "docker.io/library/alpine:latest"

func init() {
	Registry[ALPINE_LATEST] = ALPINE_LATEST_TAR_GZ
}

const ALPINE_LATEST_SIZE = "7.3M"
