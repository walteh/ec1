package harpoon_initramfs_arm64

import _ "embed"
import "github.com/walteh/ec1/pkg/binembed"

//go:embed initramfs.cpio.gz.xz
var BinaryXZ []byte

const BinaryXZChecksum = "75ec7627a3e4cf37045499f1b50c811cdd3616fb30adaafad0e5abd6c1d6da2c"

func init() {
	binembed.RegisterXZ(BinaryXZChecksum, BinaryXZ)
}
