package initramfs_aarch64

import _ "embed"

//go:embed initramfs.cpio.gz

var BinaryGZ []byte


const BinaryGZChecksum = "e7852f92875f423b6ddcbcd0d36f148875d008d0e056967cd044d2200c181e45"

