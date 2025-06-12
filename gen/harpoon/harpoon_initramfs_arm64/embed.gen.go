package harpoon_initramfs_arm64

import _ "embed"
import "github.com/walteh/ec1/pkg/binembed"

//go:embed initramfs.cpio.gz.xz
var BinaryXZ []byte

const BinaryXZChecksum = "52c11469322e6d45ab05700764c2bf38d6a91aee5335dddbeecf5907ef58f64a"

func init() {
	binembed.RegisterXZ(BinaryXZChecksum, BinaryXZ)
}
