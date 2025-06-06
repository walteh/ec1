package harpoon_initramfs_arm64

import _ "embed"
import "github.com/walteh/ec1/pkg/binembed"

//go:embed initramfs.cpio.gz.xz
var BinaryXZ []byte

const BinaryXZChecksum = "e998e7ea9870d1b4c57c6949078221dab1ca548f98e5ae46580ab6a7adcd4bf8"

func init() {
	binembed.RegisterXZ(BinaryXZChecksum, BinaryXZ)
}
