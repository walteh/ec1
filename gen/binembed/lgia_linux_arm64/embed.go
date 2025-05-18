package lgia_linux_arm64

import _ "embed"

//go:embed lgia.xz

var BinaryXZ []byte

//go:embed lgia.xz.sha256

var BinaryXZChecksum []byte

