package harpoon_initramfs_arm64

import _ "embed"
import "github.com/walteh/ec1/pkg/binembed"

//go:embed initramfs.cpio.gz.xz
var BinaryXZ []byte

const BinaryXZChecksum = "1785f67f0a5005e895647dbeafa034b57d3eadd2fc852eb8037e515989a415e6"

func init() {
	binembed.RegisterXZ(BinaryXZChecksum, BinaryXZ)
}
