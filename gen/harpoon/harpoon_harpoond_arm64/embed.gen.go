package harpoon_harpoond_arm64

import _ "embed"
import "github.com/walteh/ec1/pkg/binembed"

//go:embed harpoond.xz
var BinaryXZ []byte

const BinaryXZChecksum = "cf27ae71148de88af624765c9801e8db486f6b29b1c34305e6d4a3905c109a43"

func init() {
	binembed.RegisterXZ(BinaryXZChecksum, BinaryXZ)
}
