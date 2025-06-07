package harpoon_vmlinux_arm64

import _ "embed"
import "github.com/walteh/ec1/pkg/binembed"

//go:embed vmlinux.xz
var BinaryXZ []byte

const BinaryXZChecksum = "f1655d281cb4009369c9cb023c18aa3e75e4e5a4270d522dba08a56400c44671"

const Version = "6.15-rc7"

//go:embed config-6.15-rc7

var Config []byte

const ConfigChecksum = "8a54923717942b26d602402590303d4629e147e09394803454cf3fa94ebb5e5c"

func init() {
	binembed.RegisterXZ(BinaryXZChecksum, BinaryXZ)
	binembed.RegisterRaw(ConfigChecksum, Config)
}
