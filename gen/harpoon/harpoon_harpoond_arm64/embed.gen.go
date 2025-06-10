package harpoon_harpoond_arm64

import _ "embed"
import "github.com/walteh/ec1/pkg/binembed"

//go:embed harpoond.xz
var BinaryXZ []byte

const BinaryXZChecksum = "8fe6daf8ce7096897e4cc23927a19f28e7f9f0996b442884e4107e7e103b0299"

func init() {
	binembed.RegisterXZ(BinaryXZChecksum, BinaryXZ)
}
