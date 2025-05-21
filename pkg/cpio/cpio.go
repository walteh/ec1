package cpio

import (
	"bytes"
	"context"
	"io"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/mholt/archives"
	"gitlab.com/tozd/go/errors"

	kcpio "kraftkit.sh/cpio"
)

var (
	_ archives.Archival      = (*CpioArchival)(nil)
	_ archives.Archiver      = (*CpioArchival)(nil)
	_ archives.ArchiverAsync = (*CpioArchival)(nil)
	_ archives.Extractor     = (*CpioArchival)(nil)
	// _ archives.Inserter      = (*CpioArchival)(nil)
)

// CPIO mode constants
const (
	S_IFMT  = 0170000 // file type mask
	S_IFLNK = 0120000 // symbolic link
)

func init() {
	archives.RegisterFormat(New())
}

// CpioArchival implements CPIO format support for mholt/archives.
type CpioArchival struct{}

// New returns a new CpioArchival that can archive and extract CPIO files.
func New() *CpioArchival {
	return &CpioArchival{}
}

// Archive implements archives.Archiver by creating a CPIO archive with the given files.
func (c *CpioArchival) Archive(ctx context.Context, out io.Writer, files []archives.FileInfo) error {
	w := kcpio.NewWriter(out)
	defer w.Close()

	for _, f := range files {
		select {
		case <-ctx.Done():
			return errors.Errorf("context canceled: %w", ctx.Err())
		default:
		}

		h, err := kcpio.FileInfoHeader(f.FileInfo, f.LinkTarget)
		if err != nil {
			return errors.Errorf("creating header for %s: %w", f.Name(), err)
		}
		h.Name = f.Name() // full path expected by archives.FileInfo

		if err := w.WriteHeader(h); err != nil {
			return errors.Errorf("writing header for %s: %w", f.Name(), err)
		}

		if h.Size > 0 && !f.IsDir() && h.Linkname == "" {
			file, err := f.Open()
			if err != nil {
				return errors.Errorf("opening file %s: %w", f.Name(), err)
			}

			_, err = io.Copy(w, file)
			file.Close()
			if err != nil {
				return errors.Errorf("copying data for %s: %w", f.Name(), err)
			}
		}
	}

	return nil
}

// Insert implements archives.Inserter by adding new files to an existing CPIO archive.
func (c *CpioArchival) Insert(ctx context.Context, archive io.Reader, files []archives.FileInfo, renames map[string]string) (io.ReadCloser, error) {
	// For CPIO archives, insertion is implemented by:
	// 1. Reading all existing entries from the original archive
	// 2. Creating a new archive in memory with both existing and new entries
	// 3. Writing the new archive back to the original location

	// First, seek to the beginning to read the existing archive
	if r, ok := archive.(io.ReadSeeker); ok {
		if _, err := r.Seek(0, io.SeekStart); err != nil {
			return nil, errors.Errorf("seeking to start of archive: %w", err)
		}
	}

	// Store existing entries by name for quick lookup
	existingEntries := make(map[string]*kcpio.Header)
	fileData := make(map[string][]byte)

	// Read all existing entries with the underlying library
	reader := kcpio.NewReader(archive)
	for {
		select {
		case <-ctx.Done():
			return nil, errors.Errorf("context canceled: %w", ctx.Err())
		default:
		}

		header, _, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, errors.Errorf("reading CPIO header: %w", err)
		}

		// Skip the trailer entry
		if header.Name == "TRAILER!!!" {
			continue
		}

		// Store the header information
		existingEntries[header.Name] = header

		// For regular files, store their data
		if header.Size > 0 && !header.FileInfo().IsDir() && header.Linkname == "" {
			data := make([]byte, header.Size)
			if _, err := io.ReadFull(reader, data); err != nil {
				return nil, errors.Errorf("reading file data for %s: %w", header.Name, err)
			}
			fileData[header.Name] = data
		}
	}

	// Create a buffer for the new archive
	var newArchive bytes.Buffer

	pipeReader, pipeWriter := io.Pipe()

	// Create a new CPIO writer
	writer := kcpio.NewWriter(&newArchive)
	defer writer.Close()

	// Write all existing entries to the new archive
	for name, header := range existingEntries {
		if err := writer.WriteHeader(header); err != nil {
			return nil, errors.Errorf("writing header for existing file %s: %w", name, err)
		}

		// Write file data if it exists
		if data, ok := fileData[name]; ok && len(data) > 0 {
			if _, err := writer.Write(data); err != nil {
				return nil, errors.Errorf("writing data for existing file %s: %w", name, err)
			}
		}
	}

	for from, to := range renames {
		current, exists := existingEntries[from]
		if !exists {
			return nil, errors.Errorf("file %s does not exist in archive", from)
		}

		delete(existingEntries, from)

		current.Name = to
		current.NameSize = int64(len(to))

		existingEntries[to] = current
	}

	// Add new files, skipping any that conflict with existing entries
	for _, file := range files {
		// Skip if file already exists in the archive
		if _, exists := existingEntries[file.Name()]; exists {
			continue
		}

		// Create and write header
		header, err := kcpio.FileInfoHeader(file.FileInfo, file.LinkTarget)
		if err != nil {
			return nil, errors.Errorf("creating header for %s: %w", file.Name(), err)
		}
		header.Name = file.Name()

		if err := writer.WriteHeader(header); err != nil {
			return nil, errors.Errorf("writing header for %s: %w", file.Name(), err)
		}

		// Write file data if it's a regular file with content
		if header.Size > 0 && !file.IsDir() && header.Linkname == "" {
			fileReader, err := file.Open()
			if err != nil {
				return nil, errors.Errorf("opening file %s: %w", file.Name(), err)
			}

			_, err = io.Copy(writer, fileReader)
			fileReader.Close()
			if err != nil {
				return nil, errors.Errorf("copying data for %s: %w", file.Name(), err)
			}
		}
	}

	// Close the writer to flush any remaining data
	if err := writer.Close(); err != nil {
		return nil, errors.Errorf("closing CPIO writer: %w", err)
	}

	// Seek back to beginning of original archive to overwrite it
	if r, ok := archive.(io.ReadSeeker); ok {
		if _, err := r.Seek(0, io.SeekStart); err != nil {
			return nil, errors.Errorf("seeking to start of archive: %w", err)
		}
	}

	// Truncate the file to ensure we don't have leftover data
	truncatable, ok := archive.(interface{ Truncate(size int64) error })
	if ok {
		if err := truncatable.Truncate(0); err != nil {
			return nil, errors.Errorf("truncating archive: %w", err)
		}
	}

	go func() {
		defer pipeWriter.Close()
		// Write the new archive to the original file
		if _, err := io.Copy(pipeWriter, &newArchive); err != nil {
			pipeWriter.CloseWithError(errors.Errorf("writing new archive: %w", err))
		}

	}()

	return pipeReader, nil
}

// CpioFile implements fs.File for CPIO file entries.
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

// Read implements io.Reader.
func (c CpioFile) Read(p []byte) (n int, err error) {
	return c.reader.Read(p)
}

// Extract implements archives.Extractor by extracting a CPIO archive.
func (c *CpioArchival) Extract(ctx context.Context, r io.Reader, handle archives.FileHandler) error {
	cr := kcpio.NewReader(r)
	for {
		select {
		case <-ctx.Done():
			return errors.Errorf("context canceled: %w", ctx.Err())
		default:
		}

		hdr, _, err := cr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return errors.Errorf("reading CPIO header: %w", err)
		}

		// Handle directory entries
		if hdr.FileInfo().IsDir() {
			err = handle(ctx, archives.FileInfo{
				FileInfo:      hdr.FileInfo(),
				Header:        hdr,
				NameInArchive: hdr.Name,
				LinkTarget:    hdr.Linkname,
				Open: func() (fs.File, error) {
					return CpioFile{
						reader: io.NopCloser(bytes.NewReader(nil)),
						stat:   hdr.FileInfo(),
					}, nil
				},
			})
			if err != nil {
				return errors.Errorf("handling directory %s: %w", hdr.Name, err)
			}
			continue
		}

		// Handle symbolic links - check if linkname is present
		if hdr.Linkname != "" {
			err = handle(ctx, archives.FileInfo{
				FileInfo:      hdr.FileInfo(),
				Header:        hdr,
				NameInArchive: hdr.Name,
				LinkTarget:    hdr.Linkname,
				Open: func() (fs.File, error) {
					return CpioFile{
						reader: io.NopCloser(bytes.NewReader(nil)),
						stat:   hdr.FileInfo(),
					}, nil
				},
			})
			if err != nil {
				return errors.Errorf("handling symlink %s: %w", hdr.Name, err)
			}
			continue
		}

		// Handle regular files
		dat := make([]byte, hdr.Size)
		_, err = io.ReadFull(cr, dat)
		if err != nil {
			return errors.Errorf("reading file data for %s: %w", hdr.Name, err)
		}

		err = handle(ctx, archives.FileInfo{
			FileInfo:      hdr.FileInfo(),
			Header:        hdr,
			NameInArchive: hdr.Name,
			LinkTarget:    hdr.Linkname,
			Open: func() (fs.File, error) {
				// Return a new reader each time to ensure seekability
				return CpioFile{
					reader: io.NopCloser(bytes.NewReader(dat)),
					stat:   hdr.FileInfo(),
				}, nil
			},
		})
		if err != nil {
			return errors.Errorf("handling file %s: %w", hdr.Name, err)
		}
	}
}

// ArchiveAsync implements archives.ArchiverAsync.
func (c *CpioArchival) ArchiveAsync(ctx context.Context, output io.Writer, jobs <-chan archives.ArchiveAsyncJob) error {
	w := kcpio.NewWriter(output)
	defer w.Close()

	for {
		select {
		case <-ctx.Done():
			return errors.Errorf("context canceled: %w", ctx.Err())
		case job, ok := <-jobs:
			if !ok {
				return nil // Channel closed, we're done
			}

			err := func() error {
				file := job.File
				h, err := kcpio.FileInfoHeader(file.FileInfo, file.LinkTarget)
				if err != nil {
					return errors.Errorf("creating header for %s: %w", file.Name(), err)
				}
				h.Name = file.Name() // full path expected by archives.FileInfo

				if err := w.WriteHeader(h); err != nil {
					return errors.Errorf("writing header for %s: %w", file.Name(), err)
				}

				if h.Size > 0 && !file.IsDir() && h.Linkname == "" {
					fileReader, err := file.Open()
					if err != nil {
						return errors.Errorf("opening file %s: %w", file.Name(), err)
					}
					defer fileReader.Close()

					_, err = io.Copy(w, fileReader)
					if err != nil {
						return errors.Errorf("copying data for %s: %w", file.Name(), err)
					}
				}
				return nil
			}()

			if err != nil {
				job.Result <- err
				continue
			}

			job.Result <- nil
		}
	}
}

// Extension implements archives.Archival.
func (c *CpioArchival) Extension() string {
	return "cpio"
}

// Match implements archives.Archival.
func (c *CpioArchival) Match(ctx context.Context, filename string, stream io.Reader) (archives.MatchResult, error) {
	// CPIO magic numbers
	magicODC := []byte{0x63, 0x70, 0x69, 0x6F}             // "cpio"
	magicNEW := []byte{0x30, 0x37, 0x30, 0x37, 0x30, 0x31} // "070701"
	magicCRC := []byte{0x30, 0x37, 0x30, 0x37, 0x30, 0x32} // "070702"

	result := archives.MatchResult{}

	// Check file extension first
	ext := filepath.Ext(filename)
	if ext == ".cpio" || strings.HasSuffix(filename, ".cpio") {
		result.ByName = true
	}

	// Read first 6 bytes to check for signature
	buf := make([]byte, 6)
	n, err := stream.Read(buf)
	if err != nil && err != io.EOF {
		return archives.MatchResult{}, errors.Errorf("reading header: %w", err)
	}
	if n < 4 {
		return result, nil
	}

	// Check for ODC format
	if n >= 4 && bytes.Equal(buf[:4], magicODC) {
		result.ByStream = true
	}

	// Check for NEW or CRC format
	if n >= 6 && (bytes.Equal(buf[:6], magicNEW) || bytes.Equal(buf[:6], magicCRC)) {
		result.ByStream = true
	}

	return result, nil
}

// MediaType implements archives.Archival.
func (c *CpioArchival) MediaType() string {
	return "application/x-cpio"
}

// Register registers the CPIO format with the mholt/archives package.
func Register() {
	archives.RegisterFormat(New())
}
