package harpoon_initramfs_arm64

import _ "embed"
import "github.com/walteh/ec1/pkg/binembed"

//go:embed initramfs.cpio.gz.xz
var BinaryXZ []byte

const BinaryXZChecksum = "4ff17233e323ea703d9b19d4eeb5e8e1b42657f3b99e4080974d78460f73f5dd"

func init() {
	binembed.RegisterXZ(BinaryXZChecksum, BinaryXZ)
}
