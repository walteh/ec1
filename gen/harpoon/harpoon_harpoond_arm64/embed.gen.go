package harpoon_harpoond_arm64

import _ "embed"
import "github.com/walteh/ec1/pkg/binembed"

//go:embed harpoond.xz
var BinaryXZ []byte

const BinaryXZChecksum = "f2cbcb82b492d1752d14c354c01ffbbff718ffcf43a7c6ce59c68c0c2c42af88"

func init() {
	binembed.RegisterXZ(BinaryXZChecksum, BinaryXZ)
}
