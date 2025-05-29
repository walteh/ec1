package harpoon_harpoond_arm64

import _ "embed"
import "github.com/walteh/ec1/pkg/binembed"

//go:embed harpoond.xz
var BinaryXZ []byte

const BinaryXZChecksum = "908faff7a15d6e95e617e10a98111c6bb06315cfbddde8b9b8e12f6b95b01dd0"

func init() {
	binembed.RegisterXZ(BinaryXZChecksum, BinaryXZ)
}
