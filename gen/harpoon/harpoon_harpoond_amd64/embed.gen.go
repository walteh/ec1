package harpoon_harpoond_amd64

import _ "embed"
import "github.com/walteh/ec1/pkg/binembed"

//go:embed harpoond.xz
var BinaryXZ []byte

const BinaryXZChecksum = "a353bae276d5f376108044669659c5a81c376f24c314035007a6a8788a3709f0"

func init() {
	binembed.RegisterXZ(BinaryXZChecksum, BinaryXZ)
}
