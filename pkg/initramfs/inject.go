package initramfs

import (
	"bytes"
	"context"
	"io"
	"log"
	"log/slog"
	"time"

	"gitlab.com/tozd/go/errors"
	"go.pdmccormick.com/initramfs"

	"github.com/walteh/ec1/pkg/ext/archivesx"
)

func InjectInitBinaryToInitramfsCpio(ctx context.Context, rdr io.Reader) (io.ReadCloser, error) {

	decompressedInitBinData, err := LoadInitBinToMemory(ctx)
	if err != nil {
		return nil, errors.Errorf("uncompressing init binary: %w", err)
	}

	rdr, compressed, err := archivesx.IdentifyAndDecompress(ctx, "", rdr)
	if err != nil {
		return nil, errors.Errorf("identifying and decompressing initramfs file: %w", err)
	}

	if !compressed {
		return nil, errors.Errorf("this function only works with compressed initramfs files")
	}

	slog.InfoContext(ctx, "custom init added to initramfs", "customInitPath", "/"+customInitPath)

	buf := bytes.NewBuffer(nil)
	defer rdr.(io.ReadCloser).Close()

	ir := initramfs.NewReader(rdr)
	iw := initramfs.NewWriter(buf)

	if err = iw.WriteHeader(&initramfs.Header{
		Filename: "init",
		Mode:     0755,
		Mtime:    time.Now(),
		Uid:      0,
		Gid:      0,
		Magic:    "070707",
		NumLinks: 1,
		DataSize: uint32(len(decompressedInitBinData)),
	}); err != nil {
		return nil, errors.Errorf("writing header: %w", err)
	}

	slog.InfoContext(ctx, "wrote header")

	if _, err := io.Copy(iw, bytes.NewReader(decompressedInitBinData)); err != nil {
		return nil, errors.Errorf("Copy %s: %w", "init", err)
	}

	for _, rec := range ir.All() {
		if rec.Trailer() {
			break
		}
		if rec.Filename == customInitPath {
			rec.Filename = "init.real"
		}

		slog.InfoContext(ctx, "writing header", "filename", rec.Filename, "size", rec.DataSize)
		err := iw.WriteHeader(&initramfs.Header{
			Filename: rec.Filename,
			Mode:     rec.Mode,
			DataSize: rec.DataSize,
			Mtime:    rec.Mtime,
			Uid:      rec.Uid,
			Gid:      rec.Gid,
			Magic:    rec.Magic,
			NumLinks: rec.NumLinks,
		})
		if err != nil {
			return nil, errors.Errorf("writing header: %w", err)
		}

		slog.InfoContext(ctx, "wrote header", "filename", rec.Filename, "size", rec.DataSize)

		if rec.DataSize > 0 {
			if _, err := io.Copy(iw, ir); err != nil {
				return nil, errors.Errorf("Copy %s: %w", rec.Filename, err)
			}
		}
	}

	slog.InfoContext(ctx, "wrote all records")

	slog.InfoContext(ctx, "wrote init")

	if err = iw.WriteTrailer(); err != nil {
		return nil, errors.Errorf("writing trailer: %w", err)
	}

	slog.InfoContext(ctx, "wrote initramfs")

	isCompressed, compressType, err := ir.ContinueCompressed(nil)
	if err != nil {
		if err == io.EOF {

			return io.NopCloser(bytes.NewReader(buf.Bytes())), nil
		}

		return nil, errors.Errorf("ContinueCompressed: %w", err)
	}

	slog.InfoContext(ctx, "ContinueCompressed", "isCompressed", isCompressed, "compressType", compressType)

	if isCompressed {
		log.Printf("Found %s compressed stream", compressType)
	}

	return io.NopCloser(bytes.NewReader(buf.Bytes())), nil
}
