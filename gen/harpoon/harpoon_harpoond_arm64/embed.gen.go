package harpoon_harpoond_arm64

import _ "embed"
import "github.com/walteh/ec1/pkg/binembed"

//go:embed harpoond.xz
var BinaryXZ []byte

const BinaryXZChecksum = "7fc5d412db3403a843ff815cd3f2b075120fdbe899e53dfde56426824df06390"

func init() {
	binembed.RegisterXZ(BinaryXZChecksum, BinaryXZ)
}
