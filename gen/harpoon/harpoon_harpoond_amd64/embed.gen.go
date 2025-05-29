package harpoon_harpoond_amd64

import _ "embed"
import "github.com/walteh/ec1/pkg/binembed"

//go:embed harpoond.xz
var BinaryXZ []byte

const BinaryXZChecksum = "c9d527e7fc786fdeb21908570c11a6913d9a115cebb4ae896f730614959c4d83"

func init() {
	binembed.RegisterXZ(BinaryXZChecksum, BinaryXZ)
}
