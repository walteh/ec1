package harpoon_harpoond_arm64

import _ "embed"
import "github.com/walteh/ec1/pkg/binembed"

//go:embed harpoond.xz
var BinaryXZ []byte

const BinaryXZChecksum = "fa9a10352240407d29cc6907c6536a142a6e1fb9105d28360c2d2f79735834c1"

func init() {
	binembed.RegisterXZ(BinaryXZChecksum, BinaryXZ)
}
