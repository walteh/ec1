package initramfs_aarch64

import _ "embed"

//go:embed initramfs.cpio.gz

var BinaryGZ []byte


const BinaryGZChecksum = "5eb1ab65b1786c2975762bb904a7a9667d0ab8ffb9c33a4948f11bd813209722"

