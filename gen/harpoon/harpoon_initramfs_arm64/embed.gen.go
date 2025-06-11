package harpoon_initramfs_arm64

import _ "embed"
import "github.com/walteh/ec1/pkg/binembed"

//go:embed initramfs.cpio.gz.xz
var BinaryXZ []byte

const BinaryXZChecksum = "e2dbfd4bc4faace7669918063d22d3a993eafa54ab9cb936bdd453debb6e3047"

func init() {
	binembed.RegisterXZ(BinaryXZChecksum, BinaryXZ)
}
