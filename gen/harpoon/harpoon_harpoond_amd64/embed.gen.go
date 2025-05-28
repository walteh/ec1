package harpoon_harpoond_amd64

import _ "embed"
import "github.com/walteh/ec1/pkg/binembed"

//go:embed harpoond.xz
var BinaryXZ []byte

const BinaryXZChecksum = "808a8e983ae9c4b90e9773958fb251e7c6fe1a797dd8aa3184643830aa83e86e"

func init() {
	binembed.RegisterXZ(BinaryXZChecksum, BinaryXZ)
}
