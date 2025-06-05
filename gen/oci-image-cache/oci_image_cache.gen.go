package oci_image_cache

import (
	"embed"
	"github.com/walteh/ec1/pkg/testing/toci"
)

type OCICachedImage string

func (me OCICachedImage) String() string {
	return string(me)
}

//go:embed layouts/*
var s embed.FS

func register(imageName string, fileName string) {
	toci.MustRegisterImage(imageName, "layouts/"+fileName, s)
}

const ALPINE_LATEST_TAR_GZ_CHECKSUM = "b81f9eead5a3c93c4a95784dd6ff81f2be2582c094f2dc15ab7516f5e7d143eb"
const ALPINE_LATEST OCICachedImage = "docker.io/library/alpine:latest"
const ALPINE_LATEST_SIZE = "7.5M"

func init() { register(ALPINE_LATEST.String(), "alpine_latest.tar.gz") }

const BUSYBOX_GLIBC_TAR_GZ_CHECKSUM = "ecf1f4dff863e763f63e9313681b1717764a1994dd6e9d9675c95927f7343661"
const BUSYBOX_GLIBC OCICachedImage = "docker.io/library/busybox:glibc"
const BUSYBOX_GLIBC_SIZE = "3.8M"

func init() { register(BUSYBOX_GLIBC.String(), "busybox_glibc.tar.gz") }

const DEBIAN_BOOKWORM_SLIM_TAR_GZ_CHECKSUM = "0f0d641e6c32f0a8a5324e05ea097e883ba29104b701c0262274381d143610af"
const DEBIAN_BOOKWORM_SLIM OCICachedImage = "docker.io/library/debian:bookworm-slim"
const DEBIAN_BOOKWORM_SLIM_SIZE = "53M"

func init() { register(DEBIAN_BOOKWORM_SLIM.String(), "debian_bookworm_slim.tar.gz") }

const OVEN_BUN_ALPINE_TAR_GZ_CHECKSUM = "d2df0a975545b58b78bf527af980f01952ed810eb82c3dd77a77b35371f4af1a"
const OVEN_BUN_ALPINE OCICachedImage = "docker.io/oven/bun:alpine"
const OVEN_BUN_ALPINE_SIZE = "83M"

func init() { register(OVEN_BUN_ALPINE.String(), "oven_bun_alpine.tar.gz") }

const ALPINE_SOCAT_LATEST_TAR_GZ_CHECKSUM = "369681f37cb44479bd593a96f87eec7027c758d99425ef539a70f5b9213ad300"
const ALPINE_SOCAT_LATEST OCICachedImage = "docker.io/alpine/socat:latest"
const ALPINE_SOCAT_LATEST_SIZE = "8.6M"

func init() { register(ALPINE_SOCAT_LATEST.String(), "alpine_socat_latest.tar.gz") }
