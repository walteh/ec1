package harpoon_initramfs_arm64

import _ "embed"
import "github.com/walteh/ec1/pkg/binembed"

//go:embed initramfs.cpio.gz.xz
var BinaryXZ []byte

const BinaryXZChecksum = "8601b41e16bc8f9c444559dadb8c4687c83fcb829c04979dc65655de7e4ad3b9"

func init() {
	binembed.RegisterXZ(BinaryXZChecksum, BinaryXZ)
}
