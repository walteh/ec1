package harpoon_harpoond_arm64

import _ "embed"
import "github.com/walteh/ec1/pkg/binembed"

//go:embed harpoond.xz
var BinaryXZ []byte

const BinaryXZChecksum = "7fe0938fce58ac58e1521ee8aa7e6b552553dbc0a1ad5687be1951b6c13cf2f1"

func init() {
	binembed.RegisterXZ(BinaryXZChecksum, BinaryXZ)
}
