package harpoon_initramfs_arm64

import _ "embed"
import "github.com/walteh/ec1/pkg/binembed"

//go:embed initramfs.cpio.gz.xz
var BinaryXZ []byte

const BinaryXZChecksum = "bf81bac5ae4fa7c0fbf7cf6c5aee6ebffcff5157059f3796ee20702e615e1f39"

func init() {
	binembed.RegisterXZ(BinaryXZChecksum, BinaryXZ)
}
