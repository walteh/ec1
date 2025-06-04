package harpoon_initramfs_arm64

import _ "embed"
import "github.com/walteh/ec1/pkg/binembed"

//go:embed initramfs.cpio.gz.xz
var BinaryXZ []byte

const BinaryXZChecksum = "5f04f7db1633131b513ebadd19e24f98fa72f314611d47512642a28f963f8210"

func init() {
	binembed.RegisterXZ(BinaryXZChecksum, BinaryXZ)
}
