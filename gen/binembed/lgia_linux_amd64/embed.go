package lgia_linux_amd64

import _ "embed"

//go:embed lgia

var BinaryXZ []byte

//go:embed lgia.xz.sha256

var BinaryXZChecksum []byte

