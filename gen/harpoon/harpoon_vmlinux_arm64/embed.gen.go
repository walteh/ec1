package harpoon_vmlinux_arm64

import _ "embed"
import "github.com/walteh/ec1/pkg/binembed"

//go:embed vmlinux.xz
var BinaryXZ []byte

const BinaryXZChecksum = "10ad105892aaeba353dcaef6663372173579717c27db13fdf2738f81cbc1cacd"

const Version = "6.15-rc7"

//go:embed config-6.15-rc7

var Config []byte

const ConfigChecksum = "b87eaf74cb23aeeca0e7844ff4e55b42712f3b7c2c814692a8caa17e536f61d6"

func init() {
	binembed.RegisterXZ(BinaryXZChecksum, BinaryXZ)
	binembed.RegisterXZ(ConfigChecksum, Config)
}
