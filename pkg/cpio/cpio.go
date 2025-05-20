package cpio

import (
	"bytes"
	"context"
	"io"
	"io/fs"

	"github.com/mholt/archives"

	kcpio "kraftkit.sh/cpio"
)

var (
	_ archives.Archival      = CpioArchival{}
	_ archives.Archiver      = CpioArchival{}
	_ archives.ArchiverAsync = CpioArchival{}
)

type CpioArchival struct {
}

func (c CpioArchival) Archive(ctx context.Context, out io.Writer, files []archives.FileInfo) error {
	w := kcpio.NewWriter(out)
	defer w.Close()

	for _, f := range files {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		h, err := kcpio.FileInfoHeader(f.FileInfo, "")
		if err != nil {
			return err
		}
		h.Name = f.Name() // full path expected by archives.FileInfo

		if err := w.WriteHeader(h); err != nil {
			return err
		}
		if h.Size > 0 {
			r, err := f.Open()
			if err != nil {
				return err
			}
			defer r.Close()
			if _, err := io.Copy(w, r); err != nil {
				return err
			}
		}
	}
	return nil
}

type CpioFile struct {
	reader io.ReadCloser
	stat   fs.FileInfo
}

// Close implements fs.File.
func (c CpioFile) Close() error {
	return c.reader.Close()
}

// Stat implements fs.File.
func (c CpioFile) Stat() (fs.FileInfo, error) {
	return c.stat, nil
}

var _ fs.File = CpioFile{}

func (c CpioFile) Read(p []byte) (n int, err error) {
	return c.reader.Read(p)
}

func (c CpioArchival) Extract(ctx context.Context, r io.Reader, handle archives.FileHandler) error {
	cr := kcpio.NewReader(r)
	for {
		hdr, _, err := cr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		dat := make([]byte, hdr.Size)
		_, err = io.ReadFull(cr, dat)
		if err != nil {
			return err
		}

		r := bytes.NewReader(dat)

		cp := CpioFile{reader: io.NopCloser(r), stat: hdr.FileInfo()}

		err = handle(ctx, archives.FileInfo{
			FileInfo:      hdr.FileInfo(),
			Header:        hdr,
			NameInArchive: hdr.Name,
			LinkTarget:    hdr.Linkname,
			Open: func() (fs.File, error) {
				return cp, nil
			},
		})
		if err != nil {
			return err
		}
	}
}

// ArchiveAsync implements archives.ArchiverAsync.
func (c CpioArchival) ArchiveAsync(ctx context.Context, output io.Writer, jobs <-chan archives.ArchiveAsyncJob) error {
	panic("unimplemented")

}

// Extension implements archives.Archival.
func (c CpioArchival) Extension() string {
	return "cpio"
}

// Match implements archives.Archival.
func (c CpioArchival) Match(ctx context.Context, filename string, stream io.Reader) (archives.MatchResult, error) {
	panic("unimplemented")
}

// MediaType implements archives.Archival.
func (c CpioArchival) MediaType() string {
	panic("unimplemented")
}
