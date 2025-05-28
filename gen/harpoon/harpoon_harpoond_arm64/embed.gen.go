package harpoon_harpoond_arm64

import _ "embed"
import "github.com/walteh/ec1/pkg/binembed"

//go:embed harpoond.xz
var BinaryXZ []byte

const BinaryXZChecksum = "f34924690c4f92734283a3933725dd11a765f4bd74142b6843f6a9858af0e669"

func init() {
	binembed.RegisterXZ(BinaryXZChecksum, BinaryXZ)
}
