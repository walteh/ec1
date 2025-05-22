package initramfs

import (
	_ "unsafe"

	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"math"
	"slices"
	"strconv"
	"sync"
	"time"

	"gitlab.com/tozd/go/errors"
	"go.pdmccormick.com/initramfs"
)

func NewExecHeader(filename string) initramfs.Header {
	return initramfs.Header{
		Filename:     filename,
		FilenameSize: uint32(len(filename) + 1), // add 1 for the null terminator
		// executable
		Mode:  initramfs.Mode_FileTypeMask | initramfs.GroupExecute | initramfs.UserExecute | initramfs.OtherExecute,
		Mtime: time.Now(),
		// Uid:      uint32(os.Getuid()),
		// Gid:      uint32(os.Getgid()),
		NumLinks: 1,
		// DataSize: uint32(len(data)),
		Magic: initramfs.Magic_070701,
	}
}

func OrderedByInode(headers map[string]*initramfs.Header) []*initramfs.Header {
	names := make([]*initramfs.Header, 0, len(headers))
	for _, header := range headers {
		names = append(names, header)
	}
	slices.SortFunc(names, func(a, b *initramfs.Header) int {
		return int(a.Inode) - int(b.Inode)
	})
	return names
}

func ExtractFilesFromCpio(ctx context.Context, pipe io.Reader) (headers map[string]*initramfs.Header, data map[string][]byte, err error) {
	data = make(map[string][]byte)
	headers = make(map[string]*initramfs.Header)

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
		if old, ok := headers[rec.Filename]; ok {
			slog.WarnContext(ctx, "duplicate filename - ignoring",
				"filename", old.Filename,
				"inode", old.Inode,
				"size", old.DataSize,
				"mode", old.Mode,
				"mtime", old.Mtime,
				"uid", old.Uid,
				"gid", old.Gid)
		}
		data[rec.Filename] = buf.Bytes()
		headers[rec.Filename] = rec
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
	header.Inode = math.MaxUint32
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

	if err := iw.Close(); err != nil {
		return nil, errors.Errorf("closing CPIO writer: %w", err)
	}

	slog.InfoContext(ctx, "completed writing new CPIO file")

	// Return the new CPIO file as a reader
	return io.NopCloser(buf), nil
}

// FastInjectFileToCpio performs the same operation as InjectFileToCpio but uses direct byte manipulation
// for significantly better performance with large files
func FastInjectFileToCpio(ctx context.Context, pipe io.Reader, header initramfs.Header, data []byte) (io.ReadCloser, error) {
	// Read the entire CPIO archive into memory
	cpioData, err := io.ReadAll(pipe)
	if err != nil {
		return nil, errors.Errorf("reading CPIO data: %w", err)
	}

	// Search for the init filename in the CPIO data
	initPos := FindFileNameByteIndex(ctx, cpioData, header.Filename)
	if initPos == -1 {
		slog.WarnContext(ctx, "could not find init file in CPIO archive")
	} else {
		// Verify we found the right position by checking if it's preceded by a path component separator
		// This helps ensure we found the correct "init" and not part of another filename
		if initPos > 0 && cpioData[initPos-1] != '/' {
			// This should be the correct position for the standalone "init" filename
			// Modify the last character to convert "init" to "iniz"
			// oldByte := cpioData[initPos+int(header.FilenameSize)-2]
			cpioData[initPos+int(header.FilenameSize)-2] = 'z'
			// slog.InfoContext(ctx, "renamed original init to iniz", "position", initPos, "old", fmt.Sprintf("%s", string([]byte{oldByte})))
		} else {
			return nil, errors.Errorf("found init but context appears incorrect")
		}
	}

	// Find the CPIO trailer (TRAILER!!!) to inject our new file before it
	trailerSignature := []byte("TRAILER!!!")
	trailerPos := bytes.LastIndex(cpioData, trailerSignature)
	if trailerPos == -1 {
		return nil, errors.Errorf("could not find CPIO trailer")
	}

	trailerHeaderPos := trailerPos - 110

	header.DataSize = uint32(len(data))
	var newInitRecord bytes.Buffer

	header.Inode = math.MaxUint32

	// fmt.Println("header.Inode", header.Inode, "replacedByte", fmt.Sprintf("%s", string([]byte{cpioData[initPos+int(header.FilenameSize)-2]})))

	n, err := header.WriteTo(&newInitRecord)
	if err != nil {
		return nil, errors.Errorf("writing init header: %w", err)
	}

	// add n % 4 bytes of padding
	padding := pad(int(n), 4)
	if padding != 0 {
		newInitRecord.Write(make([]byte, padding))
	}

	nd, err := newInitRecord.Write(data)
	if err != nil {
		return nil, errors.Errorf("writing init data: %w", err)
	}

	// add nd % 4 bytes of padding
	padding = pad(int(nd), 4)
	if padding != 0 {
		newInitRecord.Write(make([]byte, padding))
	}

	newRecord := newInitRecord.Bytes()

	pos := trailerHeaderPos

	trailingZeros := -4
	for i := len(cpioData) - 1; i >= 0; i-- {
		if cpioData[i] != 0 {
			break
		}
		trailingZeros++
	}

	totalLen := len(cpioData) + len(newRecord) - trailingZeros
	out := make([]byte, totalLen)

	// copy head
	copy(out, cpioData[:pos])
	// insert newRecord
	copy(out[pos:], newRecord)
	// copy tail
	copy(out[pos+len(newRecord):], cpioData[pos:len(cpioData)-trailingZeros])

	return io.NopCloser(bytes.NewReader(out)), nil
}

func pad(length int, align int) int {
	return (align - (length % align)) % align
}

func FindFileNameByteIndex(ctx context.Context, cpioData []byte, filename string) int {
	const headerSize = 110
	const align = 4

	for pos := 0; pos+headerSize <= len(cpioData); {
		check := string(cpioData[pos : pos+6])
		if check != "070701" {
			slog.WarnContext(ctx, "magic mismatch", "pos", pos, "check", check)
			return -1
		}

		namesize, _ := strconv.ParseUint(string(cpioData[pos+94:pos+102]), 16, 32)
		filesize, _ := strconv.ParseUint(string(cpioData[pos+54:pos+62]), 16, 32)

		nameStart := pos + headerSize
		nameEnd := nameStart + int(namesize)

		if nameEnd > len(cpioData) {
			return -1
		}

		name := string(cpioData[nameStart : nameEnd-1]) // trim null

		// pad1: align header+name up to 4 bytes
		namePad := pad(headerSize+int(namesize), align)
		// pad2: align data up to 4 bytes
		filePad := pad(int(filesize), align)

		slog.InfoContext(ctx, "data", "pos", pos, "name", name, "namesize", namesize, "namePad", namePad, "filesize", filesize, "filePad", filePad)
		if name == filename {
			return pos + headerSize
		}

		pos += headerSize + int(namesize) + namePad + int(filesize) + filePad

	}

	return -1
}

func FindFileNameOffset(r io.Reader, writer io.Writer, filename string) (int64, error) {
	const headerSize = 110
	var offset int64
	br := bufio.NewReader(r)

	for {
		// 1) Read header
		hdr := make([]byte, headerSize)
		n, err := io.ReadFull(io.TeeReader(br, writer), hdr)
		offset += int64(n)
		if err != nil {
			if err == io.EOF {
				return -1, nil
			}
			return -1, err
		}
		if string(hdr[:6]) != "070701" {
			return -1, fmt.Errorf("invalid magic at %d: %q", offset-int64(n), hdr[:6])
		}

		// 2) Parse sizes
		namesize, _ := strconv.ParseUint(string(hdr[94:102]), 16, 32)
		filesize, _ := strconv.ParseUint(string(hdr[54:62]), 16, 32)

		// 3) Read filename
		nameBuf, err := br.Peek(int(namesize))
		offset += int64(n)
		if err != nil {
			return -1, err
		}
		name := string(nameBuf[:namesize-1])

		// 4) Check for match
		if name == filename {
			return offset - int64(namesize), nil
		}

		// 5) Compute padding
		pad1 := (4 - ((headerSize + int(namesize)) % 4)) % 4
		pad2 := (4 - (int(filesize) % 4)) % 4

		// 6) Skip name padding, file data, and data padding
		for _, skip := range []int64{int64(pad1), int64(filesize), int64(pad2), int64(namesize)} {
			if skip > 0 {
				n64, _ := io.CopyN(writer, br, skip)
				offset += n64
			}
		}
	}
}

func StreamInjectSlow(ctx context.Context, src io.Reader, hdr initramfs.Header, data []byte) io.ReadCloser {
	br := bufio.NewReader(src)
	pr, pw := io.Pipe()
	bw := bufio.NewWriter(pw)

	go func() {
		defer pw.Close()
		defer bw.Flush()

		for {
			header := make([]byte, 110)
			if _, err := io.ReadFull(br, header); err != nil {
				return // propagate EOF/error via pipe
			}
			if string(header[:6]) != "070701" {
				pw.CloseWithError(fmt.Errorf("bad magic"))
				return
			}

			namesize, _ := strconv.ParseUint(string(header[94:102]), 16, 32)
			filesize, _ := strconv.ParseUint(string(header[54:62]), 16, 32)

			name := make([]byte, namesize)
			io.ReadFull(br, name)

			// rename init → iniz
			trimmed := string(name[:len(name)-1])
			if trimmed == hdr.Filename {
				name[len(name)-2] = 'z' // mutate 't'→'z'
			}

			// if trailer, inject our new record just before we forward it
			if trimmed == "TRAILER!!!" {
				writeNewRecord(bw, hdr, data) // header.WriteTo + pads + data
			}

			// copy header+name to output
			bw.Write(header)
			bw.Write(name)

			// pad1
			pad1 := (4 - ((110 + int(namesize)) % 4)) % 4
			io.CopyN(bw, br, int64(pad1))

			// copy file data + pad2
			io.CopyN(bw, br, int64(filesize))
			pad2 := (4 - (int(filesize) % 4)) % 4
			io.CopyN(bw, br, int64(pad2))

			if trimmed == "TRAILER!!!" {
				return // we're done
			}
		}
	}()

	return pr
}

func parseHex32(b []byte) uint32 {
	var v uint32
	for _, c := range b {
		v = v<<4 + uint32(hexDigit[c])
	}
	return v
}

var hexDigit = [256]uint32{
	'0': 0, '1': 1, '2': 2, '3': 3, '4': 4, '5': 5,
	'6': 6, '7': 7, '8': 8, '9': 9,
	'a': 10, 'A': 10, 'b': 11, 'B': 11,
	'c': 12, 'C': 12, 'd': 13, 'D': 13,
	'e': 14, 'E': 14, 'f': 15, 'F': 15,
}

func StreamInject(ctx context.Context, src io.Reader, hdr initramfs.Header, data []byte) io.ReadCloser {
	br := bufio.NewReader(src)
	pr, pw := io.Pipe()
	bw := bufio.NewWriterSize(pw, 128<<10)

	hdrBuf := make([]byte, 110)
	nameBuf := make([]byte, 256)
	copy(nameBuf, hdr.Filename)             // ensure capacity > want len
	want := append([]byte(hdr.Filename), 0) // target + NUL
	copyBuf := make([]byte, 64<<10)         // shared copy buffer

	go func() {
		defer pw.Close()
		defer bw.Flush()

		for {
			if _, err := io.ReadFull(br, hdrBuf); err != nil {
				return
			}
			if !bytes.Equal(hdrBuf[:6], []byte("070701")) {
				pw.CloseWithError(fmt.Errorf("bad magic"))
				return
			}

			namesize := parseHex32(hdrBuf[94:102])
			filesize := parseHex32(hdrBuf[54:62])

			if int(namesize) > cap(nameBuf) {
				nameBuf = make([]byte, namesize*2)
			}
			name := nameBuf[:namesize]
			io.ReadFull(br, name)

			if bytes.Equal(name, want) {
				name[len(name)-2] = 'z'
			}

			if bytes.Equal(name[:len(name)-1], []byte("TRAILER!!!")) {
				_ = writeNewRecord(bw, hdr, data)
			}

			bw.Write(hdrBuf)
			bw.Write(name)

			pad1 := (4 - ((110 + int(namesize)) & 3)) & 3
			io.CopyBuffer(bw, io.LimitReader(br, int64(pad1)), copyBuf)
			io.CopyBuffer(bw, io.LimitReader(br, int64(filesize)), copyBuf)
			pad2 := (4 - (int(filesize) & 3)) & 3
			io.CopyBuffer(bw, io.LimitReader(br, int64(pad2)), copyBuf)

			if bytes.Equal(name[:len(name)-1], []byte("TRAILER!!!")) {
				return
			}
		}
	}()

	return pr
}

// writeNewRecord serialises hdr (newc format) plus data and both paddings.
// It never allocates large buffers; everything is written directly to w.
func writeNewRecord(w io.Writer, hdr initramfs.Header, data []byte) error {
	const headerSize = 110 // fixed for "070701" newc archives   [oai_citation:0‡Carbs Linux Repositories](https://git.carbslinux.org/forks/toybox/tree/toys/posix/cpio.c?id=a6336b942302b92f0b65ec35299e7667b9fcbe19&utm_source=chatgpt.com)
	const align = 4        // newc entries are 4‑byte aligned      [oai_citation:1‡Apache Commons](https://commons.apache.org/proper/commons-compress/apidocs/org/apache/commons/compress/archivers/cpio/CpioArchiveEntry.html?utm_source=chatgpt.com)

	// 1. Fix‑up the header to reflect this file's payload length.
	hdr.DataSize = uint32(len(data))
	hdr.Inode = math.MaxUint32

	// 2. Write header + NUL‑terminated filename.
	if _, err := hdr.WriteTo(w); err != nil {
		return err
	}

	// 3. Pad header+name block to next 4‑byte boundary.
	//
	// totalLen so far = headerSize + hdr.FilenameSize
	pad1 := (align - ((headerSize + int(hdr.FilenameSize)) % align)) % align
	if pad1 != 0 {
		if _, err := w.Write(make([]byte, pad1)); err != nil {
			return err
		}
	}

	// 4. Write file payload.
	if _, err := w.Write(data); err != nil {
		return err
	}

	// 5. Pad payload block to next 4‑byte boundary.
	pad2 := (align - (len(data) % align)) % align
	if pad2 != 0 {
		if _, err := w.Write(make([]byte, pad2)); err != nil {
			return err
		}
	}

	return nil
}

// writeNewRecordOptimized is an optimized version of writeNewRecord with reduced allocations
func writeNewRecordOptimized(w io.Writer, hdr initramfs.Header, data []byte) error {
	const headerSize = 110 // fixed for "070701" newc archives
	const align = 4        // newc entries are 4‑byte aligned

	// 1. Fix‑up the header to reflect this file's payload length.
	hdr.DataSize = uint32(len(data))
	hdr.Inode = math.MaxUint32

	// 2. Write header + NUL‑terminated filename.
	if _, err := hdr.WriteTo(w); err != nil {
		return err
	}

	// 3. Pad header+name block to next 4‑byte boundary using pre-allocated buffer.
	pad1 := (align - ((headerSize + int(hdr.FilenameSize)) % align)) % align
	if pad1 != 0 {
		if _, err := w.Write(paddingBuf1[:pad1]); err != nil {
			return err
		}
	}

	// 4. Write file payload.
	if _, err := w.Write(data); err != nil {
		return err
	}

	// 5. Pad payload block to next 4‑byte boundary using pre-allocated buffer.
	pad2 := (align - (len(data) % align)) % align
	if pad2 != 0 {
		if _, err := w.Write(paddingBuf2[:pad2]); err != nil {
			return err
		}
	}

	return nil
}

// StreamInjectOptimized is an optimized version of StreamInject with buffer pooling and performance improvements
func StreamInjectOptimized(ctx context.Context, src io.Reader, hdr initramfs.Header, data []byte) io.ReadCloser {
	br := bufio.NewReader(src)
	pr, pw := io.Pipe()
	bw := bufio.NewWriterSize(pw, 256<<10) // Larger buffer for better throughput

	// Get buffers from pools
	hdrBuf := hdrBufPool.Get().([]byte)
	nameBuf := nameBufPool.Get().([]byte)
	copyBuf := largeCopyBufPool.Get().([]byte) // Use larger buffer for copying

	// Pre-allocate and prepare the target filename with NUL terminator
	want := make([]byte, len(hdr.Filename)+1)
	copy(want, hdr.Filename)
	want[len(hdr.Filename)] = 0

	go func() {
		defer func() {
			// Return buffers to pools
			hdrBufPool.Put(hdrBuf)
			nameBufPool.Put(nameBuf)
			largeCopyBufPool.Put(copyBuf)
			pw.Close()
			bw.Flush()
		}()

		for {
			if _, err := io.ReadFull(br, hdrBuf); err != nil {
				return
			}

			// Use pre-computed constant for magic check
			if !bytes.Equal(hdrBuf[:6], cpioMagicBytes[:]) {
				pw.CloseWithError(fmt.Errorf("bad magic"))
				return
			}

			// Use optimized hex parsing
			namesize := parseHex32Fast(hdrBuf[94:102])
			filesize := parseHex32Fast(hdrBuf[54:62])

			// Resize name buffer only if needed, with growth factor
			if int(namesize) > cap(nameBuf) {
				nameBufPool.Put(nameBuf)           // Return old buffer
				nameBuf = make([]byte, namesize*2) // Create new with growth factor
			}
			name := nameBuf[:namesize]

			if _, err := io.ReadFull(br, name); err != nil {
				return
			}

			// Optimized filename comparison
			if len(name) == len(want) && bytes.Equal(name, want) {
				name[len(name)-2] = 'z'
			}

			// Check for trailer using pre-computed constant
			if len(name) > 1 && bytes.HasPrefix(name[:len(name)-1], trailerBytes[:]) {
				_ = writeNewRecordOptimized(bw, hdr, data)
			}

			// Write header and name
			if _, err := bw.Write(hdrBuf); err != nil {
				return
			}
			if _, err := bw.Write(name); err != nil {
				return
			}

			// Optimized padding calculation using bitwise operations
			pad1 := (4 - ((110 + int(namesize)) & 3)) & 3
			if pad1 > 0 {
				if _, err := io.CopyBuffer(bw, io.LimitReader(br, int64(pad1)), copyBuf); err != nil {
					return
				}
			}

			// Copy file data
			if filesize > 0 {
				if _, err := io.CopyBuffer(bw, io.LimitReader(br, int64(filesize)), copyBuf); err != nil {
					return
				}
			}

			// Data padding
			pad2 := (4 - (int(filesize) & 3)) & 3
			if pad2 > 0 {
				if _, err := io.CopyBuffer(bw, io.LimitReader(br, int64(pad2)), copyBuf); err != nil {
					return
				}
			}

			// Check if this was the trailer
			if len(name) > 1 && bytes.HasPrefix(name[:len(name)-1], trailerBytes[:]) {
				return
			}
		}
	}()

	return pr
}

// Optimized hex parsing with bounds checking
func parseHex32Fast(b []byte) uint32 {
	if len(b) != 8 {
		// Fallback to original if wrong length
		return parseHex32(b)
	}

	var v uint32
	// Unrolled loop for better performance
	v = uint32(hexDigit[b[0]])<<28 |
		uint32(hexDigit[b[1]])<<24 |
		uint32(hexDigit[b[2]])<<20 |
		uint32(hexDigit[b[3]])<<16 |
		uint32(hexDigit[b[4]])<<12 |
		uint32(hexDigit[b[5]])<<8 |
		uint32(hexDigit[b[6]])<<4 |
		uint32(hexDigit[b[7]])
	return v
}

// Pre-computed constants for better performance
var (
	cpioMagic     = []byte("070701")
	trailerPrefix = []byte("TRAILER!!!")
)

// Buffer pools for reusing memory allocations
var (
	hdrBufPool = sync.Pool{
		New: func() interface{} {
			return make([]byte, 110)
		},
	}

	nameBufPool = sync.Pool{
		New: func() interface{} {
			return make([]byte, 256)
		},
	}

	copyBufPool = sync.Pool{
		New: func() interface{} {
			return make([]byte, 64<<10) // 64KB
		},
	}

	largeCopyBufPool = sync.Pool{
		New: func() interface{} {
			return make([]byte, 256<<10) // 256KB for larger operations
		},
	}
)

// Pre-allocated padding buffers to avoid repeated allocations
var (
	paddingBuf1 = []byte{0, 0, 0} // Max 3 bytes padding
	paddingBuf2 = []byte{0, 0, 0} // Max 3 bytes padding
)

// High-performance constants and pre-compiled patterns
var (
	// Pre-compiled byte patterns for fast comparison
	cpioMagicBytes = [6]byte{'0', '7', '0', '7', '0', '1'}
	trailerBytes   = [10]byte{'T', 'R', 'A', 'I', 'L', 'E', 'R', '!', '!', '!'}

	// Chunk size for high-performance streaming (1MB chunks)
	chunkSize = 1024 * 1024

	// Pre-allocated chunk buffer pool
	chunkBufPool = sync.Pool{
		New: func() interface{} {
			return make([]byte, chunkSize)
		},
	}

	// Large buffer pool for processing
	processBufPool = sync.Pool{
		New: func() interface{} {
			return make([]byte, 2*chunkSize) // Double buffering
		},
	}
)

// StreamInjectUltra - MAXIMUM INSANITY: Minimal validation, maximum speed
func StreamInjectUltra(ctx context.Context, src io.Reader, hdr initramfs.Header, data []byte) io.ReadCloser {
	pr, pw := io.Pipe()

	go func() {
		defer pw.Close()

		// Stream directly with minimal buffering
		br := bufio.NewReaderSize(src, 32<<10) // Smaller buffer
		bw := bufio.NewWriterSize(pw, 32<<10)  // Smaller buffer
		defer bw.Flush()

		// Pre-compile injection data once
		injectionBuf := &bytes.Buffer{}
		writeNewRecord(injectionBuf, hdr, data)
		injectionData := injectionBuf.Bytes()

		// Simple state machine with minimal allocations
		state := &scanState{
			targetPattern:  []byte(hdr.Filename + "\x00"),
			trailerPattern: []byte("TRAILER!!!"),
			injectionData:  injectionData,
		}

		// Simple scanning with reused buffers
		scanBuf := make([]byte, 8192) // Fixed 8KB scan buffer

		for {
			n, err := br.Read(scanBuf)
			if n > 0 {
				processed := processChunk(scanBuf[:n], state, bw)
				if !processed && err == io.EOF {
					// Write remaining data
					bw.Write(scanBuf[:n])
				}
			}
			if err == io.EOF {
				break
			}
			if err != nil {
				pw.CloseWithError(err)
				return
			}
		}
	}()

	return pr
}

type scanState struct {
	targetPattern  []byte
	trailerPattern []byte
	injectionData  []byte
	injected       bool
	partialHeader  []byte
}

// processChunk does minimal processing on each chunk
func processChunk(chunk []byte, state *scanState, w *bufio.Writer) bool {
	// Super simple approach: just look for patterns and inject

	// Look for target filename and replace
	if pos := bytes.Index(chunk, state.targetPattern); pos != -1 {
		// Replace the 't' with 'z' in "init\x00"
		if pos+len(state.targetPattern)-2 < len(chunk) {
			chunk[pos+len(state.targetPattern)-2] = 'z'
		}
	}

	// Look for trailer and inject before it
	if !state.injected {
		if pos := bytes.Index(chunk, state.trailerPattern); pos != -1 {
			// Write up to the trailer
			w.Write(chunk[:pos])
			// Inject our data
			w.Write(state.injectionData)
			// Write from trailer onwards
			w.Write(chunk[pos:])
			state.injected = true
			return true
		}
	}

	// Just write the chunk
	w.Write(chunk)
	return false
}

// StreamInjectHyper - HYPER OPTIMIZED: No pools, stack allocation, inlined everything
func StreamInjectHyper(ctx context.Context, src io.Reader, hdr initramfs.Header, data []byte) io.ReadCloser {
	pr, pw := io.Pipe()

	go func() {
		defer pw.Close()

		// Stack-allocated buffers for maximum speed
		var hdrBuf [110]byte
		var nameBuf [512]byte                  // Fixed size, no growth
		br := bufio.NewReaderSize(src, 64<<10) // Smaller, cache-friendly
		bw := bufio.NewWriterSize(pw, 64<<10)  // Smaller, cache-friendly
		defer bw.Flush()

		// Pre-compile patterns as byte arrays for faster comparison
		wantLen := len(hdr.Filename) + 1
		trailerPattern := [10]byte{'T', 'R', 'A', 'I', 'L', 'E', 'R', '!', '!', '!'}

		for {
			// Read header - no error checking for max speed
			if _, err := io.ReadFull(br, hdrBuf[:]); err != nil {
				return
			}

			// Inline magic check - fastest possible
			if hdrBuf[0] != '0' || hdrBuf[1] != '7' || hdrBuf[2] != '0' ||
				hdrBuf[3] != '7' || hdrBuf[4] != '0' || hdrBuf[5] != '1' {
				pw.CloseWithError(fmt.Errorf("bad magic"))
				return
			}

			// Ultra-fast inline hex parsing - no function calls
			namesize := uint32(hexDigit[hdrBuf[94]])<<28 | uint32(hexDigit[hdrBuf[95]])<<24 |
				uint32(hexDigit[hdrBuf[96]])<<20 | uint32(hexDigit[hdrBuf[97]])<<16 |
				uint32(hexDigit[hdrBuf[98]])<<12 | uint32(hexDigit[hdrBuf[99]])<<8 |
				uint32(hexDigit[hdrBuf[100]])<<4 | uint32(hexDigit[hdrBuf[101]])

			filesize := uint32(hexDigit[hdrBuf[54]])<<28 | uint32(hexDigit[hdrBuf[55]])<<24 |
				uint32(hexDigit[hdrBuf[56]])<<20 | uint32(hexDigit[hdrBuf[57]])<<16 |
				uint32(hexDigit[hdrBuf[58]])<<12 | uint32(hexDigit[hdrBuf[59]])<<8 |
				uint32(hexDigit[hdrBuf[60]])<<4 | uint32(hexDigit[hdrBuf[61]])

			// Bounds check
			if namesize > 512 {
				return
			}

			// Read name into stack buffer
			name := nameBuf[:namesize]
			if _, err := io.ReadFull(br, name); err != nil {
				return
			}

			// Ultra-fast filename check - inline comparison
			if int(namesize) == wantLen && name[0] == hdr.Filename[0] {
				match := true
				for i := 1; i < len(hdr.Filename); i++ {
					if name[i] != hdr.Filename[i] {
						match = false
						break
					}
				}
				if match && name[namesize-1] == 0 {
					name[namesize-2] = 'z' // Change 't' to 'z'
				}
			}

			// Ultra-fast trailer check - inline comparison
			isTrailer := false
			if namesize > 10 {
				isTrailer = true
				for i := 0; i < 10; i++ {
					if name[i] != trailerPattern[i] {
						isTrailer = false
						break
					}
				}
			}

			if isTrailer {
				// Inject before trailer
				writeNewRecord(bw, hdr, data)
			}

			// Write header and name in one call
			bw.Write(hdrBuf[:])
			bw.Write(name)

			// Fast padding calculation and copy
			pad1 := (4 - ((110 + int(namesize)) & 3)) & 3
			pad2 := (4 - (int(filesize) & 3)) & 3

			// Copy all data in minimal calls
			totalCopy := int64(pad1) + int64(filesize) + int64(pad2)
			if totalCopy > 0 {
				io.CopyN(bw, br, totalCopy)
			}

			if isTrailer {
				return
			}
		}
	}()

	return pr
}

// FastInjectFileToCpioHyper - Hyper-optimized version with minimal allocations
func FastInjectFileToCpioHyper(ctx context.Context, pipe io.Reader, header initramfs.Header, data []byte) (io.ReadCloser, error) {
	// Read the entire CPIO archive into memory
	cpioData, err := io.ReadAll(pipe)
	if err != nil {
		return nil, errors.Errorf("reading CPIO data: %w", err)
	}

	// Use unsafe operations for maximum speed
	cpioLen := len(cpioData)
	targetBytes := []byte(header.Filename)

	// Fast search for the target filename using Boyer-Moore-like approach
	initPos := fastSearchFilename(cpioData, targetBytes)
	if initPos != -1 {
		// Replace 't' with 'z' directly
		cpioData[initPos+len(header.Filename)-1] = 'z'
	}

	// Fast trailer search using reverse scan
	trailerPos := fastSearchTrailer(cpioData)
	if trailerPos == -1 {
		return nil, errors.Errorf("could not find CPIO trailer")
	}

	// Pre-calculate new record size to avoid buffer resizing
	header.DataSize = uint32(len(data))
	header.Inode = math.MaxUint32

	// Calculate exact sizes needed
	headerSize := 110 + int(header.FilenameSize)
	pad1 := (4 - (headerSize % 4)) % 4
	pad2 := (4 - (len(data) % 4)) % 4
	newRecordSize := headerSize + pad1 + len(data) + pad2

	// Count trailing zeros efficiently
	trailingZeros := countTrailingZeros(cpioData)

	// Allocate output exactly once
	totalLen := cpioLen + newRecordSize - trailingZeros
	out := make([]byte, totalLen)

	trailerHeaderPos := trailerPos - 110

	// Triple copy optimization - minimize memory operations
	copy(out, cpioData[:trailerHeaderPos])

	// Write new record directly into output buffer
	pos := trailerHeaderPos
	pos += writeHeaderDirect(out[pos:], header)
	pos += writePaddingDirect(out[pos:], pad1)
	pos += copy(out[pos:], data)
	pos += writePaddingDirect(out[pos:], pad2)

	// Copy remainder
	copy(out[pos:], cpioData[trailerHeaderPos:cpioLen-trailingZeros])

	return io.NopCloser(bytes.NewReader(out)), nil
}

// fastSearchFilename uses optimized pattern search
func fastSearchFilename(data []byte, pattern []byte) int {
	dataLen := len(data)
	patternLen := len(pattern)

	if patternLen == 0 || dataLen < patternLen {
		return -1
	}

	// Simple but fast search
	for i := 0; i <= dataLen-patternLen; i++ {
		// Quick first-byte check
		if data[i] == pattern[0] {
			match := true
			for j := 1; j < patternLen; j++ {
				if data[i+j] != pattern[j] {
					match = false
					break
				}
			}
			if match {
				// Verify it's followed by null terminator and not part of a larger name
				if i+patternLen < dataLen && data[i+patternLen] == 0 {
					return i
				}
			}
		}
	}

	return -1
}

// fastSearchTrailer searches for trailer from the end
func fastSearchTrailer(data []byte) int {
	trailer := []byte("TRAILER!!!")
	dataLen := len(data)
	trailerLen := len(trailer)

	// Search backwards for better cache locality
	for i := dataLen - trailerLen; i >= 0; i-- {
		if data[i] == 'T' {
			match := true
			for j := 1; j < trailerLen; j++ {
				if data[i+j] != trailer[j] {
					match = false
					break
				}
			}
			if match {
				return i
			}
		}
	}

	return -1
}

// countTrailingZeros counts trailing zero bytes efficiently
func countTrailingZeros(data []byte) int {
	count := 0
	for i := len(data) - 1; i >= 0 && data[i] == 0; i-- {
		count++
	}
	return count
}

// writeHeaderDirect writes header directly to buffer, returns bytes written
func writeHeaderDirect(buf []byte, hdr initramfs.Header) int {
	var tmpBuf bytes.Buffer
	hdr.WriteTo(&tmpBuf)
	return copy(buf, tmpBuf.Bytes())
}

// writePaddingDirect writes padding bytes directly to buffer
func writePaddingDirect(buf []byte, count int) int {
	for i := 0; i < count; i++ {
		buf[i] = 0
	}
	return count
}
