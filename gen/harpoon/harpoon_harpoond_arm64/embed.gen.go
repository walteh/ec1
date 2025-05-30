package harpoon_harpoond_arm64

import _ "embed"
import "github.com/walteh/ec1/pkg/binembed"

//go:embed harpoond.xz
var BinaryXZ []byte

const BinaryXZChecksum = "efde243817e1fc69eb48faa50fbeac176ee7fa693c90d3041dde736258a0ccfb"

func init() {
	binembed.RegisterXZ(BinaryXZChecksum, BinaryXZ)
}
