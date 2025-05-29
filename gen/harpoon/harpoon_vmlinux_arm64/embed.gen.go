package harpoon_vmlinux_arm64

import _ "embed"
import "github.com/walteh/ec1/pkg/binembed"

//go:embed vmlinux.xz
var BinaryXZ []byte

const BinaryXZChecksum = "180604635c3b7aec05cea3de52f41c6540172fedb8a7e839684cccebd672d110"

const Version = "6.15-rc7"

//go:embed config-6.15-rc7

var Config []byte

const ConfigChecksum = "14031fdbf74287c53a2a24c9c7a90c0a520e146e1f510b1112a28e8dae3e577c"

func init() {
	binembed.RegisterXZ(BinaryXZChecksum, BinaryXZ)
	binembed.RegisterXZ(ConfigChecksum, Config)
}
