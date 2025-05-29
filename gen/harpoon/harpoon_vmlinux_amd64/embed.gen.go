package harpoon_vmlinux_amd64

import _ "embed"
import "github.com/walteh/ec1/pkg/binembed"

//go:embed vmlinux.xz
var BinaryXZ []byte

const BinaryXZChecksum = "4005e0547c31bd7618520c57890fa25617c51e0b642ea88c4860c115080a201a"

const Version = "6.15-rc7"

//go:embed config-6.15-rc7

var Config []byte

const ConfigChecksum = "a9827286aa935b058fc9afe3452e6ebab71bdb8fa93f79604face7d3ab675188"

func init() {
	binembed.RegisterXZ(BinaryXZChecksum, BinaryXZ)
	binembed.RegisterRaw(ConfigChecksum, Config)
}
