package iox

import (
	"bytes"
	"io"

	"gitlab.com/tozd/go/errors"
)

func ByteValidationReader(offset int, want []byte, r io.Reader) (io.ReadCloser, error) {
	// Check if we can discard bytes efficiently
	if seeker, ok := r.(io.ReadSeeker); ok {

		currOffset, err := seeker.Seek(0, io.SeekCurrent)
		if err != nil {
			return nil, errors.Errorf("seeking to offset %d: %w", offset, err)
		}

		defer func() { _, _ = seeker.Seek(currOffset, io.SeekStart) }()

		// If we can seek, just jump to the offset
		_, err = seeker.Seek(int64(offset), io.SeekStart)
		if err != nil {
			return nil, errors.Errorf("seeking to offset %d: %w", offset, err)
		}

		// Read the bytes we want to check
		check := make([]byte, len(want))
		_, err = io.ReadFull(r, check)
		if err != nil {
			return nil, errors.Errorf("reading bytes at offset %d: %w", offset, err)
		}

		// Validate the bytes
		if !bytes.Equal(check, want) {
			return nil, errors.Errorf("invalid bytes at offset %d (want='0x%x':'%s', got='0x%x':'%s')", offset, want, want, check, check)
		}

		return PreservedNopCloser(seeker), nil
	}

	// this sucks if the offset is huge, but effectively not sure of a way around it

	check := make([]byte, offset+len(want))

	l, err := io.ReadFull(r, check)
	if err != nil {
		return nil, errors.Errorf("reading bytes from offset 0 to %d (found %d bytes): %w", offset+len(want), l, err)
	}

	got := check[offset : offset+len(want)]
	// Validate the bytes
	if !bytes.Equal(got, want) {
		return nil, errors.Errorf("invalid bytes at offset %d (want='0x%x':'%s', got='0x%x':'%s')", offset, want, want, got, got)
	}

	// Create a MultiReader to combine the check bytes and the rest of the input
	combined := io.MultiReader(bytes.NewReader(check), r)

	return PreservedNopCloser(combined), nil
}
