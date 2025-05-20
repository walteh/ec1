package host

import (
	"bytes"
	"debug/elf"
	"debug/pe"
	"io"
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

// isPEFile checks if a file is a valid PE executable
func isPEFile(r io.ReaderAt) (bool, error) {
	_, err := pe.NewFile(r)
	if err != nil {
		// Not a PE file
		return false, nil
	}
	return true, nil
}

// isELFFile checks if a file is a valid ELF executable
func isELFFile(r io.ReaderAt) (bool, error) {
	_, err := elf.NewFile(r)
	if err != nil {
		// Not an ELF file
		return false, nil
	}
	return true, nil
}

// IsKernelUncompressed checks if the provided file is an uncompressed kernel
// that can be directly used by the Virtualization Framework.
func IsKernelUncompressed(filename string) (bool, error) {
	file, err := os.Open(filename)
	if err != nil {
		return false, err
	}
	defer file.Close()

	// Check if it's a PE file, which is the format used by EFI executables
	// (this is the format of the Fedora FCOS kernel)
	isPE, err := isPEFile(file)
	if err != nil {
		return false, err
	}
	if isPE {
		// PE files are inherently uncompressed executable formats
		return true, nil
	}

	// Check if it's an uncompressed ELF file
	isELF, err := isELFFile(file)
	if err != nil {
		return false, err
	}
	if isELF {
		// ELF files are inherently uncompressed executable formats
		return true, nil
	}

	// Check if it's an ARM64 boot image using the original method
	buf := make([]byte, 2048)
	_, err = file.Seek(0, 0) // Rewind the file
	if err != nil {
		return false, err
	}
	_, err = file.Read(buf)
	if err != nil {
		return false, err
	}

	return isUncompressedArm64Kernel(buf), nil
}
