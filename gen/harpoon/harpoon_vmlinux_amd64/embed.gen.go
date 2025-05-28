package harpoon_vmlinux_amd64

import _ "embed"
import "github.com/walteh/ec1/pkg/binembed"

//go:embed vmlinux.xz
var BinaryXZ []byte

const BinaryXZChecksum = "d47feffa9c734b01663dccc72b9ce40794c84906cd5501d4a4a346e0cc4688ea"

const Version = "6.15-rc7"

//go:embed config-6.15-rc7

var Config []byte

const ConfigChecksum = "a8779b60104aabd009fb06dffc16d63e98acb6e0c45a0ccd837026f09864e2c3"

func init() {
	binembed.RegisterXZ(BinaryXZChecksum, BinaryXZ)
	binembed.RegisterXZ(ConfigChecksum, Config)
}
