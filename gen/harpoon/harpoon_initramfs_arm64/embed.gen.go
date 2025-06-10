package harpoon_initramfs_arm64

import _ "embed"
import "github.com/walteh/ec1/pkg/binembed"

//go:embed initramfs.cpio.gz.xz
var BinaryXZ []byte

const BinaryXZChecksum = "e3558f8321a29048a9112884897694a6ea30fcb6fe6014eb4528e74fd3af5da1"

func init() {
	binembed.RegisterXZ(BinaryXZChecksum, BinaryXZ)
}
