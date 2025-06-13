package harpoon_harpoond_arm64

import _ "embed"
import "github.com/walteh/ec1/pkg/binembed"

//go:embed harpoond.xz
var BinaryXZ []byte

const BinaryXZChecksum = "79cc6ff264e1ec5f98bdccfec5c3a43e8a7f042513453dda16341c0d380c9bb4"

func init() {
	binembed.RegisterXZ(BinaryXZChecksum, BinaryXZ)
}
