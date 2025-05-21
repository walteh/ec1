package osx

import (
	"io"
	"os"
)

var _ io.ReaderAt = &TempFileRewindReader{}

type TempFileRewindReader struct {
	r   io.Reader
	pos int64
	dup *os.File
}

func NewTempFileRewindReader(r io.Reader) *TempFileRewindReader {
	df, err := os.CreateTemp("", "initramfs-discarder")
	if err != nil {
		panic(err)
	}
	return &TempFileRewindReader{
		r:   r,
		pos: 0,
		dup: df,
	}
}

// ReadAt implements ReadAt for a discarder.
// It is an error for the offset to be negative.
func (r *TempFileRewindReader) ReadAt(p []byte, off int64) (int, error) {

	if off-r.pos < 0 {
		return r.dup.ReadAt(p, off)
		// return 0, fmt.Errorf("negative seek on discarder not allowed: off=%d, pos=%d", off, r.pos)
	}
	if off != r.pos {
		i, err := io.Copy(r.dup, io.LimitReader(r.r, off-r.pos))
		if err != nil || i != off-r.pos {
			return 0, err
		}
		r.pos += i
	}
	n, err := io.ReadFull(io.TeeReader(r.r, r.dup), p)
	if err != nil {
		return n, err
	}
	r.pos += int64(n)
	return n, err
}

func (r *TempFileRewindReader) Seek(offset int64, whence int) (int64, error) {
	return r.dup.Seek(offset, whence)
}

func (r *TempFileRewindReader) Close() error {
	// close dup and delete the file
	err := r.dup.Close()
	if err != nil {
		return err
	}
	return os.Remove(r.dup.Name())
}
