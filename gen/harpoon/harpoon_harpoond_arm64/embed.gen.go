package harpoon_harpoond_arm64

import _ "embed"
import "github.com/walteh/ec1/pkg/binembed"

//go:embed harpoond.xz
var BinaryXZ []byte

const BinaryXZChecksum = "c2de37e6f0c6a86879e46a012f52eecbe5577c5dcb3478161d7a4aa32c530359"

func init() {
	binembed.RegisterXZ(BinaryXZChecksum, BinaryXZ)
}
