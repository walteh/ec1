package harpoon_harpoond_arm64

import _ "embed"
import "github.com/walteh/ec1/pkg/binembed"

//go:embed harpoond.xz
var BinaryXZ []byte

const BinaryXZChecksum = "b6ef9e8f93a01e1a97dbe9d83c180008b9e133b0b05224c8cdcaefb221e7fcb9"

func init() {
	binembed.RegisterXZ(BinaryXZChecksum, BinaryXZ)
}
