package harpoon_initramfs_arm64

import _ "embed"
import "github.com/walteh/ec1/pkg/binembed"

//go:embed initramfs.cpio.gz.xz
var BinaryXZ []byte

const BinaryXZChecksum = "025d3f01048a9c9f59b75d89def641df3806cac44e24398d00adb18166b81cee"

func init() {
	binembed.RegisterXZ(BinaryXZChecksum, BinaryXZ)
}
