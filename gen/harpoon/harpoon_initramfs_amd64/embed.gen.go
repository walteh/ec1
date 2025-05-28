package harpoon_initramfs_amd64

import _ "embed"
import "github.com/walteh/ec1/pkg/binembed"

//go:embed initramfs.cpio.gz.xz
var BinaryXZ []byte

const BinaryXZChecksum = "8e5a2a4de0873787a9994c3e70d2a6a39bb4ff03849520f91afb950ef30664d2"

func init() {
	binembed.RegisterXZ(BinaryXZChecksum, BinaryXZ)
}
