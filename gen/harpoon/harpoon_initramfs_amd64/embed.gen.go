package harpoon_initramfs_amd64

import _ "embed"
import "github.com/walteh/ec1/pkg/binembed"

//go:embed initramfs.cpio.gz.xz
var BinaryXZ []byte

const BinaryXZChecksum = "018d036ff51e29bb65c09036fb977538ff9956129bcd3c957c7fe1d142f8e31d"

func init() {
	binembed.RegisterXZ(BinaryXZChecksum, BinaryXZ)
}
