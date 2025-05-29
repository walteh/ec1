package harpoon_initramfs_arm64

import _ "embed"
import "github.com/walteh/ec1/pkg/binembed"

//go:embed initramfs.cpio.gz.xz
var BinaryXZ []byte

const BinaryXZChecksum = "3714c5c0dac2ed99ed33f282720c51f941edb4874c13420b53d5f90200631586"

func init() {
	binembed.RegisterXZ(BinaryXZChecksum, BinaryXZ)
}
