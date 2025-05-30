package harpoon_initramfs_arm64

import _ "embed"
import "github.com/walteh/ec1/pkg/binembed"

//go:embed initramfs.cpio.gz.xz
var BinaryXZ []byte

const BinaryXZChecksum = "9b492b09b996ece14f5b20cc48668e63ad0b1490bde3e7496cc2fd72b608b60a"

func init() {
	binembed.RegisterXZ(BinaryXZChecksum, BinaryXZ)
}
