package harpoon_initramfs_amd64

import _ "embed"
import "github.com/walteh/ec1/pkg/binembed"

//go:embed initramfs.cpio.gz.xz
var BinaryXZ []byte

const BinaryXZChecksum = "00a8124d938c700a311c6c1b5ec59243cfb10c01ea3c2ca200730ff99e7a3bf2"

func init() {
	binembed.RegisterXZ(BinaryXZChecksum, BinaryXZ)
}
