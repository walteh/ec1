package initramfs

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"time"

	"gitlab.com/tozd/go/errors"
	"go.pdmccormick.com/initramfs"
)

func NewExecHeader(filename string) initramfs.Header {
	return initramfs.Header{
		Filename: filename,
		// executable
		Mode:     initramfs.Mode_File | initramfs.GroupExecute | initramfs.UserExecute | initramfs.OtherExecute,
		Mtime:    time.Now(),
		Uid:      0,
		Gid:      0,
		NumLinks: 1,
		// DataSize: uint32(len(data)),
		Magic: initramfs.Magic_070701,
	}
}

func ExtractFilesFromCpio(ctx context.Context, pipe io.Reader) (headers map[string]initramfs.Header, data map[string][]byte, err error) {
	data = make(map[string][]byte)
	headers = make(map[string]initramfs.Header)

	ir := initramfs.NewReader(pipe)
	for {
		rec, err := ir.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, nil, errors.Errorf("reading CPIO record: %w", err)
		}
		if rec.Trailer() {
			break
		}

		// read next n bytes and return them
		buf := bytes.NewBuffer(nil)
		_, err = io.CopyN(buf, ir, int64(rec.DataSize))
		if err != nil {
			return nil, nil, errors.Errorf("copying data for %s: %w", rec.Filename, err)
		}
		data[rec.Filename] = buf.Bytes()
		headers[rec.Filename] = *rec
	}

	return headers, data, nil
}

// InjectInitBinaryToInitramfsCpio injects the init binary into a CPIO format initramfs
// It takes the original initramfs file as a reader and returns a reader with the modified file
func InjectFileToCpio(ctx context.Context, pipe io.Reader, header initramfs.Header, data []byte) (io.ReadCloser, error) {
	// Load the custom init binary

	// Create a buffer for the new CPIO file
	buf := bytes.NewBuffer(nil)

	// Create CPIO reader and writer
	ir := initramfs.NewReader(pipe)
	iw := initramfs.NewWriter(buf)

	// Process all records from the original CPIO
	for {
		rec, err := ir.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, errors.Errorf("reading CPIO record: %w", err)
		}

		// End of archive
		if rec.Trailer() {
			break
		}

		// Rename the original init to init.real
		if rec.Filename == header.Filename {
			// replace the last letter with z
			rec.Filename = rec.Filename[:len(rec.Filename)-1] + "z"
			slog.InfoContext(ctx, "renaming original init to iniz", "mode", rec.Mode)
		}

		// Write the header for this record
		err = iw.WriteHeader(rec)
		if err != nil {
			return nil, errors.Errorf("writing header for %s: %w", rec.Filename, err)
		}

		// If this record has data, copy it
		if rec.DataSize > 0 {
			_, err := io.CopyN(iw, ir, int64(rec.DataSize))
			if err != nil {
				return nil, errors.Errorf("copying data for %s: %w", rec.Filename, err)
			}
		}
	}

	header.DataSize = uint32(len(data))

	// First add our custom init file to the beginning
	if err := iw.WriteHeader(&header); err != nil {
		return nil, errors.Errorf("writing header for custom init: %w", err)
	}

	// Write the init binary data
	if _, err := io.CopyN(iw, bytes.NewReader(data), int64(len(data))); err != nil && err != io.EOF {
		return nil, errors.Errorf("writing init binary data: %w", err)
	}

	slog.InfoContext(ctx, "wrote custom init to new CPIO:", "size", len(data), "filename", header.Filename)

	// Write the trailer to finalize the CPIO
	if err := iw.WriteTrailer(); err != nil {
		return nil, errors.Errorf("writing CPIO trailer: %w", err)
	}

	slog.InfoContext(ctx, "completed writing new CPIO file")

	// Return the new CPIO file as a reader
	return io.NopCloser(bytes.NewReader(buf.Bytes())), nil
}
