package harpoon_initramfs_arm64

import _ "embed"
import "github.com/walteh/ec1/pkg/binembed"

//go:embed initramfs.cpio.gz.xz
var BinaryXZ []byte

const BinaryXZChecksum = "6f8069d1f922832a42268401b04d5c92c42b5b783baf218728108c3e1598ec34"

func init() {
	binembed.RegisterXZ(BinaryXZChecksum, BinaryXZ)
}
