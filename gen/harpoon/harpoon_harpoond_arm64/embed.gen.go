package harpoon_harpoond_arm64

import _ "embed"
import "github.com/walteh/ec1/pkg/binembed"

//go:embed harpoond.xz
var BinaryXZ []byte

const BinaryXZChecksum = "f5ad39433a92c69727e334b18a8fbc677cb753cbbc8bb25f1d1a8c2eb608ed74"

func init() {
	binembed.RegisterXZ(BinaryXZChecksum, BinaryXZ)
}
