package harpoon_harpoond_arm64

import _ "embed"
import "github.com/walteh/ec1/pkg/binembed"

//go:embed harpoond.xz
var BinaryXZ []byte

const BinaryXZChecksum = "681e3af7d20c9dd21c6de9446f3a287ef1186cbf2231c2b23201d55f20ad053c"

func init() {
	binembed.RegisterXZ(BinaryXZChecksum, BinaryXZ)
}
