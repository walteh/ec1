package initramfs_aarch64

import _ "embed"

//go:embed initramfs.cpio.gz

var BinaryGZ []byte


const BinaryGZChecksum = "bc4918d5563884499b1033c64fa018c118d2464bef2c4b25384711ff1ca4743b"

