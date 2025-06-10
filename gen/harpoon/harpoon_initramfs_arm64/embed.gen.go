package harpoon_initramfs_arm64

import _ "embed"
import "github.com/walteh/ec1/pkg/binembed"

//go:embed initramfs.cpio.gz.xz
var BinaryXZ []byte

const BinaryXZChecksum = "98f4798c671c2381e255c06edfe9ff056bfda2dc3058776feddfd195f080e9d0"

func init() {
	binembed.RegisterXZ(BinaryXZChecksum, BinaryXZ)
}
