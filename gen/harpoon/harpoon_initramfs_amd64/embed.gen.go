package harpoon_initramfs_amd64

import _ "embed"
import "github.com/walteh/ec1/pkg/binembed"

//go:embed initramfs.cpio.gz.xz
var BinaryXZ []byte

const BinaryXZChecksum = "9d1151037192d6b62c9c081efbf32cc63f6c02a10d3ec75229861e1218a501f8"

func init() {
	binembed.RegisterXZ(BinaryXZChecksum, BinaryXZ)
}
