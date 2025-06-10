package harpoon_harpoond_arm64

import _ "embed"
import "github.com/walteh/ec1/pkg/binembed"

//go:embed harpoond.xz
var BinaryXZ []byte

const BinaryXZChecksum = "60823165f4330fa78873c2fafa2785a98ae2786c21ba70b0535f5d2914735520"

func init() {
	binembed.RegisterXZ(BinaryXZChecksum, BinaryXZ)
}
