package initramfs_aarch64

import _ "embed"

//go:embed initramfs.cpio.gz

var BinaryGZ []byte


const BinaryGZChecksum = "b08da0603bae830390b8e967862e92c657d8320ef0438939d79d31053c94723b"

