package vmlinux_aarch64

import _ "embed"

//go:embed vmlinux.xz

var BinaryXZ []byte


const BinaryXZChecksum = "b1765ddb7557dfd30c363ee411002e46697c9764b1143a5797affa60e2f80f54"

const Version = "6.15-rc7"

//go:embed config-6.15-rc7

var Config []byte


const ConfigChecksum = "b87eaf74cb23aeeca0e7844ff4e55b42712f3b7c2c814692a8caa17e536f61d6"

