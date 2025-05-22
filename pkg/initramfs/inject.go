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

// InjectInitBinaryToInitramfsCpio injects the init binary into a CPIO format initramfs
// It takes the original initramfs file as a reader and returns a reader with the modified file
func InjectInitBinaryToInitramfsCpio(ctx context.Context, rdr io.Reader) (io.ReadCloser, error) {
	// Load the custom init binary
	decompressedInitBinData, err := LoadInitBinToMemory(ctx)
	if err != nil {
		return nil, errors.Errorf("uncompressing init binary: %w", err)
	}

	// Decompress the initramfs if it's compressed
	// rdr, compressed, err := archivesx.IdentifyAndDecompress(ctx, "", rdr)
	// if err != nil {
	// 	return nil, errors.Errorf("identifying and decompressing initramfs file: %w", err)
	// }

	// if !compressed {
	// 	return nil, errors.Errorf("this function only works with compressed initramfs files")
	// }

	slog.InfoContext(ctx, "custom init added to initramfs", "customInitPath", "/"+customInitPath)

	// Create a buffer for the new CPIO file
	buf := bytes.NewBuffer(nil)

	// Close the reader when done
	defer func() {
		if closer, ok := rdr.(io.ReadCloser); ok {
			closer.Close()
		}
	}()

	// Create CPIO reader and writer
	ir := initramfs.NewReader(rdr)
	iw := initramfs.NewWriter(buf)

	// First add our custom init file to the beginning
	if err = iw.WriteHeader(&initramfs.Header{
		Filename: "init",
		// executable
		Mode:     initramfs.Mode_File | initramfs.GroupExecute | initramfs.UserExecute | initramfs.OtherExecute,
		Mtime:    time.Now(),
		Uid:      0,
		Gid:      0,
		NumLinks: 1,
		DataSize: uint32(len(decompressedInitBinData)),
		Magic:    initramfs.Magic_070701,
	}); err != nil {
		return nil, errors.Errorf("writing header for custom init: %w", err)
	}

	// Write the init binary data
	if _, err := io.Copy(iw, bytes.NewReader(decompressedInitBinData)); err != nil {
		return nil, errors.Errorf("writing init binary data: %w", err)
	}

	slog.InfoContext(ctx, "wrote custom init to new CPIO")

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
		if rec.Filename == customInitPath {
			rec.Filename = "init.real"

			slog.InfoContext(ctx, "renaming original init to init.real", "mode", rec.Mode)
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

	// Write the trailer to finalize the CPIO
	if err = iw.WriteTrailer(); err != nil {
		return nil, errors.Errorf("writing CPIO trailer: %w", err)
	}

	slog.InfoContext(ctx, "completed writing new CPIO file")

	// Return the new CPIO file as a reader
	return io.NopCloser(bytes.NewReader(buf.Bytes())), nil
}
