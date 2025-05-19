package host

import (
	"bytes"
	"os"
)

// from https://github.com/h2non/filetype/blob/cfcd7d097bc4990dc8fc86187307651ae79bf9d9/matchers/document.go#L159-L174
func compareBytes(slice, subSlice []byte, startOffset int) bool {
	sl := len(subSlice)

	if startOffset+sl > len(slice) {
		return false
	}

	s := slice[startOffset : startOffset+sl]
	return bytes.Equal(s, subSlice)
}

// patterns and offsets are coming from https://github.com/file/file/blob/master/magic/Magdir/linux
func isUncompressedArm64Kernel(buf []byte) bool {
	pattern := []byte{0x41, 0x52, 0x4d, 0x64}
	offset := 0x38

	return compareBytes(buf, pattern, offset)
}

func IsKernelUncompressed(filename string) (bool, error) {
	file, err := os.Open(filename)
	if err != nil {
		return false, err
	}
	defer file.Close()

	buf := make([]byte, 2048)
	_, err = file.Read(buf)
	if err != nil {
		return false, err
	}

	return isUncompressedArm64Kernel(buf), nil
}
