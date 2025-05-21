package magic

import (
	"io"

	"github.com/walteh/ec1/pkg/ext/iox"
)

func ARM64LinuxKernelValidationReader(r io.Reader) (io.ReadCloser, error) {
	return iox.ByteValidationReader(ARM64MagicOffset, ARM64Magic.Bytes(), r)
}
