package harpoon_initramfs_arm64

import _ "embed"
import "github.com/walteh/ec1/pkg/binembed"

//go:embed initramfs.cpio.gz.xz
var BinaryXZ []byte

const BinaryXZChecksum = "b7e99915de4d5be6f1b04a60fc6d89e58d80558c0a810aad68a069d7b513a7af"

func init() {
	binembed.RegisterXZ(BinaryXZChecksum, BinaryXZ)
}
