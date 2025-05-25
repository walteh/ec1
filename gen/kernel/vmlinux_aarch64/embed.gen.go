package vmlinux_aarch64

import _ "embed"

//go:embed vmlinux.xz

var BinaryXZ []byte


const BinaryXZChecksum = "dccbce1a50c1afea9eebaacc65af913c9e9fd796af654717954921582b6c9a73"

const Version = "6.15-rc7"

//go:embed config-6.15-rc7

var Config []byte


const ConfigChecksum = "b87eaf74cb23aeeca0e7844ff4e55b42712f3b7c2c814692a8caa17e536f61d6"

