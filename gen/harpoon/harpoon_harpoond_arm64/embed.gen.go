package harpoon_harpoond_arm64

import _ "embed"
import "github.com/walteh/ec1/pkg/binembed"

//go:embed harpoond.xz
var BinaryXZ []byte

const BinaryXZChecksum = "eee68c8ceb82e9027035d298c7b0a2b987a4802faaed39c8c6ba628d1d31115f"

func init() {
	binembed.RegisterXZ(BinaryXZChecksum, BinaryXZ)
}
