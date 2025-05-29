package harpoon_vmlinux_amd64

import _ "embed"
import "github.com/walteh/ec1/pkg/binembed"

//go:embed vmlinux.xz
var BinaryXZ []byte

const BinaryXZChecksum = "ccc523b36c8013e07c8d3f172701c4f7302aa90a0532043e68eb5084d110fe1c"

const Version = "6.15-rc7"

//go:embed config-6.15-rc7

var Config []byte

const ConfigChecksum = "fdc56e6dcc3eaded7082406c77f82598af416fb9e39da833f1cf8cb1ae098f4a"

func init() {
	binembed.RegisterXZ(BinaryXZChecksum, BinaryXZ)
	binembed.RegisterXZ(ConfigChecksum, Config)
}
