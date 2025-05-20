// Package unzbootgo provides a pure Go implementation for extracting Linux kernels
// from EFI applications that carry the kernel image in compressed form.
package unzbootgo

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/mholt/archives"
	"gitlab.com/tozd/go/errors"
)

const (
	// ARM64MagicOffset is the offset of the ARM64 magic signature in a kernel file
	ARM64MagicOffset = 56
	// LoadImageMaxBytes is the maximum size for decompressed kernel image
	LoadImageMaxBytes = 256 << 20 // 256MB

	// GzipMagic is the magic number for gzip compression
	// GzipMagic = []int{31, 139}
)

// EFI PE/COFF magic signatures
var (
	// MSDOSMagic is the MS-DOS stub magic number
	MSDOSMagic = []byte("MZ")
	// ZimgMagic identifies Linux EFI zboot images
	ZimgMagic = []byte("zimg")
	// LinuxMagic is the Linux header magic
	LinuxMagic = []byte{0xcd, 0x23, 0x82, 0x81}
	// ARM64Magic identifies an ARM64 kernel
	ARM64Magic = []byte("ARM\x64")
)

// LinuxEFIZbootHeader represents the header of a Linux EFI zboot image
type LinuxEFIZbootHeader struct {
	MSDOSMagic      [2]byte
	Reserved0       [2]byte
	Zimg            [4]byte
	PayloadOffset   uint32
	PayloadSize     uint32
	Reserved1       [8]byte
	CompressionType [32]byte
	LinuxMagic      [4]byte
	PEHeaderOffset  uint32
}

func (h *LinuxEFIZbootHeader) CompressorFromCompressionType() (func(io.Reader) (io.ReadCloser, error), error) {
	switch dat := string(h.CompressionType[:]); {
	case strings.HasPrefix(dat, "gzip"):
		return func(r io.Reader) (io.ReadCloser, error) {
			return (&archives.Gz{}).OpenReader(r)
		}, nil
	}

	return nil, errors.New("unknown compression type")
}

// ExtractKernel extracts a Linux kernel from an EFI application.
// It takes the path to the EFI application and the output path for the kernel.
// Returns an error if extraction fails.
func ExtractKernel(inputFile, outputFile string) error {
	// Open input file
	f, err := os.Open(inputFile)
	if err != nil {
		return errors.Errorf("opening input file: %w", err)
	}
	defer f.Close()

	// Process the data
	kernelReader, err := ProcessKernel(context.Background(), f)
	if err != nil {
		return err
	}

	// Create output file
	out, err := os.Create(outputFile)
	if err != nil {
		return errors.Errorf("creating output file: %w", err)
	}
	defer out.Close()

	// Copy processed data to output file
	_, err = io.Copy(out, kernelReader)
	if err != nil {
		return errors.Errorf("writing output file: %w", err)
	}

	return nil
}

// ReaderAt combines io.ReaderAt and io.Reader interfaces
type ReadCounter struct {
	// io.ReaderAt
	count int64
	io.Reader
}

func (r *ReadCounter) Read(p []byte) (n int, err error) {
	n, err = r.Reader.Read(p)
	r.count += int64(n)
	return
}

func (r *ReadCounter) Count() int64 {
	return r.count
}

type FakeGzipReader struct {
	reader io.Reader
	count  int64
	header []byte
}

// ProcessKernel examines the provided data and if it's an EFI zboot image,
// extracts and decompresses the contained kernel. Returns either the original
// data or the extracted kernel data.
func ProcessKernel(ctx context.Context, reader io.Reader) (io.Reader, error) {

	// all, err := io.ReadAll(reader)
	// if err != nil {
	// 	return nil, errors.Errorf("reading all: %w", err)
	// }

	// Parse the header
	header := LinuxEFIZbootHeader{}
	readCounter := &ReadCounter{Reader: reader}
	// bufferedReader := NewFakeSeeker(readCounter, int64(header.PayloadOffset))
	if err := binary.Read(readCounter, binary.LittleEndian, &header); err != nil {
		return nil, errors.Errorf("reading header: %w", err)
	}

	// Verify magic numbers
	if !bytes.Equal(header.MSDOSMagic[:], MSDOSMagic) ||
		!bytes.Equal(header.Zimg[:], ZimgMagic) ||
		!bytes.Equal(header.LinuxMagic[:], LinuxMagic) {
		// Reset the reader to the beginning and return it
		if seeker, ok := reader.(io.Seeker); ok {
			_, err := seeker.Seek(0, io.SeekStart)
			if err != nil {
				return nil, errors.Errorf("seeking to beginning: %w", err)
			}
		}
		// Not an EFI zboot image, return as is
		return reader, nil
	}

	// sectionReader := io.NewSectionReader(bytes.NewReader(all), int64(header.PayloadOffset), int64(header.PayloadSize))
	countBefore := readCounter.Count()
	discarded, err := io.ReadFull(readCounter, make([]byte, int(header.PayloadOffset)-int(countBefore)))
	if err != nil {
		return nil, errors.Errorf("discarding to payload offset: %w", err)
	}
	// discarded, err := bufferedReader.Discard(int(header.PayloadOffset) - int(countBefore))
	// if err != nil {
	// 	return nil, errors.Errorf("discarding to payload offset: %w", err)
	// }
	countAfter := readCounter.Count()

	// peeked, err := bufferedReader.Peek(256)
	// if err != nil {
	// 	return nil, errors.Errorf("peeking at payload: %w", err)
	// }

	slog.InfoContext(ctx, "discarding payload offset",
		// "bytes", peeked,
		"discarded", discarded,
		"count before", countBefore,
		"count after", countAfter,
		"payload offset", header.PayloadOffset,
		"payload size", header.PayloadSize,
		// "reader size", bufferedReader.Buffered(),
	)

	// slog.InfoContext(ctx, "discarded",
	// 	"bytes discarded", discarded,
	// 	"payload offset", header.PayloadOffset,
	// 	"payload size", header.PayloadSize,
	// 	// "buffered offset", bufferedReader.Buffered(),
	// 	// "buffered size", bufferedReader.Size(),
	// 	"header", header,
	// 	"compression type", string(header.CompressionType[:]),
	// 	"pe header offset", header.PEHeaderOffset,
	// )

	// // find the gzip header
	// gzipHeaderOffset := bytes.Index(peeked, []byte{0x1f, 0x8b})
	// if gzipHeaderOffset == -1 {
	// 	return nil, errors.New("gzip header not found")
	// }

	// discard up to the gzip header
	// discarded, err = bufferedReader.Discard(gzipHeaderOffset)
	// if err != nil {
	// 	return nil, errors.Errorf("discarding to gzip header: %w", err)
	// }

	payload := io.LimitReader(readCounter, int64(header.PayloadSize))

	// Extract compressed payload
	// // payload := io.NewSectionReader(reader, int64(header.PayloadOffset), int64(header.PayloadSize))

	// // // Identify and decompress the payload
	// format, decompressedReader, err := archives.Identify(ctx, "", payload)
	// if err != nil {
	// 	return nil, errors.Errorf("identifying compression format: %w", err)
	// }

	compressor, err := header.CompressorFromCompressionType()
	if err != nil {
		return nil, errors.Errorf("getting compressor: %w", err)
	}

	// if comp, ok := format.(archives.Compression); ok {
	decompressedReader, err := compressor(payload)
	if err != nil {
		return nil, errors.Errorf("decompression failed: %w", err)
	}
	// }

	// Verify ARM64 kernel
	validationReader, err := validateARM64Kernel(decompressedReader)
	if err != nil {
		return nil, errors.Errorf("validating ARM64 kernel: %w", err)
	}

	return validationReader, nil
}

// validateARM64Kernel verifies if the decompressed data is a valid ARM64 kernel
// by checking for the ARM64 magic bytes at the expected offset
func validateARM64Kernel(r io.Reader) (io.Reader, error) {
	// Create a buffered reader to peek ahead
	br := bufio.NewReaderSize(r, ARM64MagicOffset+4)

	// Peek ahead to get the magic bytes
	peek, err := br.Peek(ARM64MagicOffset + 4)
	if err != nil {
		return nil, errors.Errorf("peeking for ARM64 magic: %w", err)
	}

	// Check for ARM64 magic
	if !bytes.Equal(peek[ARM64MagicOffset:ARM64MagicOffset+4], ARM64Magic) {
		return nil, errors.New("not a valid ARM64 kernel")
	}

	// Return the reader (with the peeked bytes still available to read)
	return br, nil
}

// decompressGzip decompresses data with gzip compression
// func decompressGzip(data []byte) ([]byte, error) {
// 	format := archiver.Gz{}
// 	reader, err := format.OpenReader(bytes.NewReader(data))
// 	if err != nil {
// 		return nil, errors.Errorf("opening gzip reader: %w", err)
// 	}
// 	defer reader.Close()

// 	buf := new(bytes.Buffer)
// 	_, err = io.Copy(buf, reader)
// 	if err != nil {
// 		return nil, errors.Errorf("reading gzip data: %w", err)
// 	}

// 	return buf.Bytes(), nil
// }

// // decompressZstd decompresses data with zstd compression
// func decompressZstd(data []byte) ([]byte, error) {
// 	format := archiver.Zstd{}
// 	reader, err := format.Open(bytes.NewReader(data))
// 	if err != nil {
// 		return nil, errors.Errorf("opening zstd reader: %w", err)
// 	}
// 	defer reader.Close()

// 	buf := new(bytes.Buffer)
// 	_, err = io.Copy(buf, reader)
// 	if err != nil {
// 		return nil, errors.Errorf("reading zstd data: %w", err)
// 	}

// 	return buf.Bytes(), nil
// }
