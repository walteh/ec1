package harpoon_harpoond_amd64

import _ "embed"
import "github.com/walteh/ec1/pkg/binembed"

//go:embed harpoond.xz
var BinaryXZ []byte

const BinaryXZChecksum = "d791f03d6cc835cd67a5d5cb96cada79b8595f537f38aa6354aa4a3e57501f49"

func init() {
	binembed.RegisterXZ(BinaryXZChecksum, BinaryXZ)
}
