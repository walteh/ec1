package harpoon_vmlinux_amd64

import _ "embed"
import "github.com/walteh/ec1/pkg/binembed"

//go:embed vmlinux.xz
var BinaryXZ []byte

const BinaryXZChecksum = "29d79a62b749c939e3eff2d20ab97dbed33a842739b7adfeefea412962cba9d6"

const Version = "6.15-rc7"

//go:embed config-6.15-rc7

var Config []byte

const ConfigChecksum = "d0e00f55ba4159d115d7b19cfc3f507569b05d0295896d2dbcb8c645f7887bc5"

func init() {
	binembed.RegisterXZ(BinaryXZChecksum, BinaryXZ)
	binembed.RegisterRaw(ConfigChecksum, Config)
}
