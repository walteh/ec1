package harpoon_initramfs_arm64

import _ "embed"
import "github.com/walteh/ec1/pkg/binembed"

//go:embed initramfs.cpio.gz.xz
var BinaryXZ []byte

const BinaryXZChecksum = "4a59f5df4aa1f44051e013e41e13acf3c799d5bb2911e20d718edb033eb2eb1f"

func init() {
	binembed.RegisterXZ(BinaryXZChecksum, BinaryXZ)
}
