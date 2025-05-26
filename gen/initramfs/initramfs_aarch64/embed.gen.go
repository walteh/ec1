package initramfs_aarch64

import _ "embed"

//go:embed initramfs.cpio.gz

var BinaryGZ []byte


const BinaryGZChecksum = "16b803cb272a1f791130690dfccfbbf2c9d7be506a83cf3b218964bab414047f"

