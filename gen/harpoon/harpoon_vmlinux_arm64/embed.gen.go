package harpoon_vmlinux_arm64

import _ "embed"
import "github.com/walteh/ec1/pkg/binembed"

//go:embed vmlinux.xz
var BinaryXZ []byte

const BinaryXZChecksum = "9f11999cccd1f90e0896b5deb04dad080e52763c55a449cd006c07f26aeffdff"

const Version = "6.15-rc7"

//go:embed config-6.15-rc7

var Config []byte

const ConfigChecksum = "669fb0dc421acbd0ac50b7bd9b6f996adf0e82e4e31ee13cadb25d5d44c2f27e"

func init() {
	binembed.RegisterXZ(BinaryXZChecksum, BinaryXZ)
	binembed.RegisterRaw(ConfigChecksum, Config)
}
