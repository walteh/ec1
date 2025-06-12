package harpoon_harpoond_arm64

import _ "embed"
import "github.com/walteh/ec1/pkg/binembed"

//go:embed harpoond.xz
var BinaryXZ []byte

const BinaryXZChecksum = "a5150bf5253e78e0a082798e8975f7c790e5c00a4776f226cf18359d58d5c73a"

func init() {
	binembed.RegisterXZ(BinaryXZChecksum, BinaryXZ)
}
