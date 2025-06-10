package harpoon_harpoond_arm64

import _ "embed"
import "github.com/walteh/ec1/pkg/binembed"

//go:embed harpoond.xz
var BinaryXZ []byte

const BinaryXZChecksum = "d5bae39dc2adf1d304faf1d120f74a4d0f166101d1d809c26f1744cc8fd0c2ed"

func init() {
	binembed.RegisterXZ(BinaryXZChecksum, BinaryXZ)
}
