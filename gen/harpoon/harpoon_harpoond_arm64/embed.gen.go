package harpoon_harpoond_arm64

import _ "embed"
import "github.com/walteh/ec1/pkg/binembed"

//go:embed harpoond.xz
var BinaryXZ []byte

const BinaryXZChecksum = "d22040397956421b4c2e22bcd762d84d23d221a8e625fa09888a63eb2e29c888"

func init() {
	binembed.RegisterXZ(BinaryXZChecksum, BinaryXZ)
}
