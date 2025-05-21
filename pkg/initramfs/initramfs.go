package initramfs

import (
	"bytes"
	"context"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"github.com/kdomanski/iso9660"
	"github.com/mholt/archives"
	"github.com/u-root/u-root/pkg/cpio"
	"gitlab.com/tozd/go/errors"
	"go.pdmccormick.com/initramfs"

	"github.com/walteh/ec1/gen/binembed/lgia_linux_arm64"
	"github.com/walteh/ec1/pkg/host"

	mycpio "github.com/walteh/ec1/pkg/cpio"
)

var (
	// Embedded, statically‑compiled init+gRPC binary (as XZ data)
	initBin             = lgia_linux_arm64.BinaryXZ
	decompressedInitBin = []byte{}
	initMutex           = sync.Mutex{}
	customInitPath      = "init"
)

func LoadInitBinToMemory(ctx context.Context) ([]byte, error) {
	initMutex.Lock()
	defer initMutex.Unlock()

	if len(decompressedInitBin) > 0 {
		return decompressedInitBin, nil
	}

	arc, err := (&archives.Xz{}).OpenReader(bytes.NewReader(initBin))
	if err != nil {
		return nil, errors.Errorf("opening xz reader: %w", err)
	}

	decompressedInitBind, err := io.ReadAll(arc)
	if err != nil {
		return nil, errors.Errorf("reading uncompressed init bin: %w", err)
	}

	slog.InfoContext(ctx, "loaded init bin to memory", "size", len(decompressedInitBind))

	decompressedInitBin = decompressedInitBind

	return decompressedInitBin, nil
}

type discarder struct {
	r   io.Reader
	pos int64
	dup *os.File
}

func NewDiscarder(r io.Reader) *discarder {
	df, err := os.CreateTemp("", "initramfs-discarder")
	if err != nil {
		panic(err)
	}
	return &discarder{
		r:   r,
		pos: 0,
		dup: df,
	}
}

// ReadAt implements ReadAt for a discarder.
// It is an error for the offset to be negative.
func (r *discarder) ReadAt(p []byte, off int64) (int, error) {

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

func PrepareEmptyIso(ctx context.Context, dir string) (string, error) {
	iso, err := iso9660.NewWriter()
	if err != nil {
		return "", errors.Errorf("creating iso writer: %w", err)
	}
	decompressedInitBinData, err := LoadInitBinToMemory(ctx)
	if err != nil {
		return "", errors.Errorf("uncompressing init binary: %w", err)
	}

	err = iso.AddFile(bytes.NewReader(decompressedInitBinData), "init")
	if err != nil {
		return "", errors.Errorf("adding init file to iso: %w", err)
	}

	tmp, err := os.Create(filepath.Join(dir, "init.ec1.iso"))
	if err != nil {
		return "", errors.Errorf("creating temp file: %w", err)
	}
	defer tmp.Close()

	err = iso.WriteTo(tmp, "init.ec1.volume")
	if err != nil {
		return "", errors.Errorf("writing iso: %w", err)
	}

	return tmp.Name(), nil
}

func PrepareEmptyInitramfs(ctx context.Context, dir string) (string, error) {

	decompressedInitBinData, err := LoadInitBinToMemory(ctx)
	if err != nil {
		return "", errors.Errorf("uncompressing init binary: %w", err)
	}

	f, err := os.Create(filepath.Join(dir, "initramfs.cpio.gz"))
	if err != nil {
		return "", errors.Errorf("creating initramfs.cpio.gz: %w", err)
	}
	defer f.Close()

	img := cpio.Newc.Writer(f) // wrap gzip+cpio internally

	records := []cpio.Record{}
	customInitRecord := cpio.StaticFile("init", string(decompressedInitBinData), 0755)
	records = append(records, customInitRecord)

	// Write all records to the output file
	if err := cpio.WriteRecords(img, records); err != nil {
		return "", errors.Errorf("writing CPIO records: %w", err)
	}

	if err := cpio.WriteTrailer(img); err != nil {
		return "", errors.Errorf("writing CPIO trailer: %w", err)
	}

	return f.Name(), nil
}

func PrepareInitramfsCpio(ctx context.Context, initramfsPath string) (string, error) {
	outputPath := initramfsPath + ".ec1"

	inputFile, err := os.Open(initramfsPath)
	if err != nil {
		return "", errors.Errorf("opening initramfs file %s: %w", initramfsPath, err)
	}
	defer inputFile.Close()

	rdr, err := InjectInitBinaryToInitramfsCpio(ctx, inputFile)
	if err != nil {
		return "", errors.Errorf("identifying and decompressing initramfs file %s: %w", initramfsPath, err)
	}

	// Create a new file next to the original with .ec1 suffix
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return "", errors.Errorf("creating output file %s: %w", outputPath, err)
	}
	defer outputFile.Close()

	_, err = io.Copy(outputFile, rdr)
	if err != nil {
		return "", errors.Errorf("copying initramfs file %s: %w", initramfsPath, err)
	}

	return outputPath, nil
}

// the archiveing situaiton ijn this function is rough
// for some reason, xz compressed cpio files HAVE to be decompressed in process by a stream
// otherwise they fail, so we can't break up the

func InjectInitBinaryToInitramfsCpioAlt(ctx context.Context, rdr io.Reader, compression archives.Compression) (io.ReadCloser, error) {

	decompressedInitBinData, err := LoadInitBinToMemory(ctx)
	if err != nil {
		return nil, errors.Errorf("uncompressing init binary: %w", err)
	}

	// create a temp file
	tmp, err := os.CreateTemp("", "initramfs-init")
	if err != nil {
		return nil, errors.Errorf("creating temp file: %w", err)
	}
	defer func() {
		tmp.Close()
		os.Remove(tmp.Name())
	}()

	_, err = tmp.Write(decompressedInitBinData)
	if err != nil {
		return nil, errors.Errorf("writing temp file: %w", err)
	}

	fi, err := tmp.Stat()
	if err != nil {
		return nil, errors.Errorf("getting temp file stat: %w", err)
	}

	files := []archives.FileInfo{}

	files = append(files, archives.FileInfo{
		NameInArchive: "init",
		FileInfo:      fi,
		LinkTarget:    "",
		Header:        nil,
		Open: func() (fs.File, error) {
			tmp.Seek(0, io.SeekStart)
			return tmp, nil
		},
	})

	out, err := mycpio.New().Insert(ctx, rdr, files, map[string]string{
		"init": "init.real",
	})
	if err != nil {
		return nil, errors.Errorf("extracting initramfs file: %w", err)
	}

	return out, nil

	// all, err := io.ReadAll(rdr)
	// if err != nil {
	// 	return nil, errors.Errorf("reading initramfs file: %w", err)
	// }

	// write the whole thing to a temp file
	// tmp, err := os.CreateTemp("", "initramfs-cpio")
	// if err != nil {
	// 	return nil, errors.Errorf("creating temp file: %w", err)
	// }
	// defer tmp.Close()

	// _, err = io.Copy(tmp, rdr)
	// if err != nil {
	// 	return nil, errors.Errorf("copying initramfs file: %w", err)
	// }

	// // tmp.Close()

	// tmp, err = os.Open(tmp.Name())
	// if err != nil {
	// 	return nil, errors.Errorf("opening temp file: %w", err)
	// }
	// defer tmp.Close()

	cpioReader := cpio.Newc.Reader(tmp)

	// Read all records from the input CPIO
	records, err := cpio.ReadAllRecords(cpioReader)
	if err != nil {
		return nil, errors.Errorf("reading CPIO records: %w", err)
	}

	var foundInit bool = false

	// Filter out any existing init.ec1 files
	var filteredRecords []cpio.Record
	for _, rec := range records {
		// fmt.Println("rec.Name", rec.Name)
		if rec.Name == customInitPath {
			rec.Name = "init.real"
			foundInit = true
		}
		filteredRecords = append(filteredRecords, rec)
	}

	if !foundInit {
		return nil, errors.Errorf("init.ec1 not found in initramfs")
	}

	// Add our custom init file
	customInitRecord := cpio.StaticRecord(decompressedInitBinData, cpio.Info{
		Name: customInitPath,
		Mode: 0755,
	})
	filteredRecords = append(filteredRecords, customInitRecord)

	pipeReader, pipeWriter := io.Pipe()

	outputWriter := io.WriteCloser(pipeWriter)

	// if compression != nil {
	// 	outputWriterd, err := compression.OpenWriter(outputWriter)
	// 	if err != nil {
	// 		return nil, errors.Errorf("opening compression writer: %w", err)
	// 	}
	// 	outputWriter = outputWriterd
	// }

	cpioWriter := cpio.Newc.Writer(outputWriter)

	// recs := cpio.ArchiveFromRecords(filteredRecords)

	// cpioWriter.Write(filteredRecords)

	go func() {
		defer pipeWriter.Close()

		slog.InfoContext(ctx, "going to write CPIO records", "count", len(filteredRecords))

		// Write all records to the output file
		if err := cpio.WriteRecords(cpioWriter, filteredRecords); err != nil {
			slog.ErrorContext(ctx, "writing CPIO records", "error", err)
		}

		// Write trailer to finalize the archive
		if err := cpio.WriteTrailer(cpioWriter); err != nil {
			slog.ErrorContext(ctx, "writing CPIO trailer", "error", err)
		}

		slog.InfoContext(ctx, "wrote CPIO records", "count", len(filteredRecords))

		if err := outputWriter.Close(); err != nil {
			slog.ErrorContext(ctx, "closing output writer", "error", err)
		}

		slog.InfoContext(ctx, "closed output writer for initramfs")

	}()

	slog.InfoContext(ctx, "custom init added to initramfs", "customInitPath", "/"+customInitPath)

	return pipeReader, nil
}

func init() {
	initramfs.CompressReaders[initramfs.Gzip] = func(r io.Reader) (io.Reader, error) { return (&archives.Gz{}).OpenReader(r) }
}

func PrepareKernel(ctx context.Context, kernelPath string) (string, error) {
	// Create a new file next to the original with .ec1 suffix
	outputPath := kernelPath + ".ec1"

	fle, err := os.Open(kernelPath)
	if err != nil {
		return "", errors.Errorf("opening kernel: %w", err)
	}
	defer fle.Close()

	// just check and decompress if needed

	// ext, err := vmlinuz.NewKernelExtractor(fle, false)
	// rdr, err := ext.ExtractAll(ctx)
	// if err != nil {
	// 	return "", errors.Errorf("extracting kernel: %w", err)
	// }

	rdr, err := host.ExtractVmlinuxNative(ctx, kernelPath)
	if err != nil {
		return "", errors.Errorf("extracting kernel: %w", err)
	}

	// Copy the kernel to the output file
	dstFile, err := os.Create(outputPath)
	if err != nil {
		return "", errors.Errorf("creating output file %s: %w", outputPath, err)
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, rdr)
	if err != nil {
		return "", errors.Errorf("copying kernel: %w", err)
	}

	return outputPath, nil
}
