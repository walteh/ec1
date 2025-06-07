package harpoon_initramfs_arm64

import _ "embed"
import "github.com/walteh/ec1/pkg/binembed"

//go:embed initramfs.cpio.gz.xz
var BinaryXZ []byte

const BinaryXZChecksum = "d747fe710cbd16dc9365d55a6f4037eeb8cb722d4a9a0a4dc2df034b77fda88b"

func init() {
	binembed.RegisterXZ(BinaryXZChecksum, BinaryXZ)
}
