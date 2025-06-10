package harpoon_initramfs_arm64

import _ "embed"
import "github.com/walteh/ec1/pkg/binembed"

//go:embed initramfs.cpio.gz.xz
var BinaryXZ []byte

const BinaryXZChecksum = "6f4f24a137cb0d708b59b5897d524e018c67a60f6c16d6c34ada523bac9e38bc"

func init() {
	binembed.RegisterXZ(BinaryXZChecksum, BinaryXZ)
}
