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

const ALPINE_LATEST_TAR_GZ_CHECKSUM = "03ecfe66f4f5a5651bab7aa0dcd7bc18f85e333d9aceadea93bdb80ecd35155f"
const ALPINE_LATEST OCICachedImage = "docker.io/library/alpine:latest"
const ALPINE_LATEST_SIZE = "7.6M"

func init() { register(ALPINE_LATEST.String(), "alpine_latest.tar.gz") }

const BUSYBOX_GLIBC_TAR_GZ_CHECKSUM = "7cdde60d4de8aa3f6f5bce4ea9935142585d5769dbc8cfd2fe9ecff7843e2e53"
const BUSYBOX_GLIBC OCICachedImage = "docker.io/library/busybox:glibc"
const BUSYBOX_GLIBC_SIZE = "3.8M"

func init() { register(BUSYBOX_GLIBC.String(), "busybox_glibc.tar.gz") }

const DEBIAN_BOOKWORM_SLIM_TAR_GZ_CHECKSUM = "9fb3a8f976501d77e8dd48260ba4b552c3b360489a9f121ed9b159ad2104aee4"
const DEBIAN_BOOKWORM_SLIM OCICachedImage = "docker.io/library/debian:bookworm-slim"
const DEBIAN_BOOKWORM_SLIM_SIZE = "53M"

func init() { register(DEBIAN_BOOKWORM_SLIM.String(), "debian_bookworm_slim.tar.gz") }

const OVEN_BUN_ALPINE_TAR_GZ_CHECKSUM = "640f1f301710193c591a542c34c9c1d3b472258a407c42faa595605c5b16015d"
const OVEN_BUN_ALPINE OCICachedImage = "docker.io/oven/bun:alpine"
const OVEN_BUN_ALPINE_SIZE = "83M"

func init() { register(OVEN_BUN_ALPINE.String(), "oven_bun_alpine.tar.gz") }

const ALPINE_SOCAT_LATEST_TAR_GZ_CHECKSUM = "cc9ec4933fcebe096547fddbbf46c457397b2bda7321103c7c3f5159e2d55d0a"
const ALPINE_SOCAT_LATEST OCICachedImage = "docker.io/alpine/socat:latest"
const ALPINE_SOCAT_LATEST_SIZE = "8.6M"

func init() { register(ALPINE_SOCAT_LATEST.String(), "alpine_socat_latest.tar.gz") }
