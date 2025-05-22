package initramfs

import (
	_ "unsafe"

	"bytes"
	"context"
	"io"
	"log/slog"
	"math"
	"slices"
	"strconv"
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

// // find all instances of the filename
// mfsSig := append([]byte("bin/curl"), 0x00)
// initSplits := bytes.Split(cpioData, mfsSig)
// initPoses := []int{}
// for _, split := range initSplits {
// 	initPoses = append(initPoses, len(split))
// }

// fmt.Println("initPoses", initPoses)

//go:linkname writeHeader go.pdmccormick.com/initramfs.(*Writer).writeHeader
func writeHeader(iw *initramfs.Writer, hdr *initramfs.Header) error

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

// CPIOSearchResult represents the result of searching for a file in a CPIO archive
// type CPIOSearchResult struct {
// 	// Found indicates whether the file was found
// 	Found bool
// 	// HeaderPosition is the byte offset where the CPIO header starts
// 	HeaderPosition int
// 	// FilenamePosition is the byte offset where the filename starts
// 	FilenamePosition int
// 	// DataPosition is the byte offset where the file data starts
// 	DataPosition int
// 	// Size is the size of the file data
// 	Size uint32
// }

// func grabBits(cpioData []byte, offset, length int) uint64 {
// 	// we need to round length up to the nearest multiple of 4
// 	alignedLength := (length + 3) &^ 3
// 	// we need to round offset down to the nearest multiple of 4
// 	alignedOffset := offset &^ 3

// 	// we need to divide both by 4
// 	alignedOffsetDiv4 := alignedOffset / 4
// 	alignedLengthDiv4 := alignedLength / 4

// 	dat := cpioData[alignedOffsetDiv4 : alignedOffsetDiv4+alignedLengthDiv4]

// 	fmt.Println(
// 		"alignedOffsetDiv4", alignedOffsetDiv4,
// 		"alignedLengthDiv4", alignedLengthDiv4,
// 		"offset", offset,
// 		"length", length,
// 		"alignedOffset", alignedOffset,
// 		"alignedLength", alignedLength,
// 		"len(cpioData)", len(cpioData),
// 		"dat", fmt.Sprintf("%x", dat),
// 	)

// 	// we need to shift the offse	t left by the number of bits length is away from 4
// 	rawData := binary.BigEndian.Uint64(dat)

// 	// we need to shift the offset right by the number of bits length is away from 4
// 	rawData = rawData << uint64(offset-alignedOffset)
// 	rawData = rawData >> uint64(alignedLength-length)
// 	return rawData
// }

// func alignPadding(offset, align int) int {
// 	remainder := offset % align
// 	if remainder == 0 {
// 		return 0
// 	}
// 	return align - remainder
// }

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

// // FindFileInCPIO searches for a file in a CPIO archive and returns its position
// // This function properly accounts for the CPIO structure by scanning header-by-header
// func FindFileNameByteIndex2(ctx context.Context, cpioData []byte, filename string) int {
// 	// Look for the CPIO magic number "070701"
// 	// magicBytes := []byte(initramfs.Magic_070701)

// 	nextHeaderPos := func(lastHeaderPos int) (string, int) {
// 		dat := cpioData[lastHeaderPos : lastHeaderPos+6]

// 		if string(dat) != initramfs.Magic_070701 {
// 			var cp []byte = make([]byte, len(cpioData))
// 			var last20bytes []byte
// 			_ = copy(cp, cpioData)

// 			if lastHeaderPos > 20 {
// 				last20bytes = cp[lastHeaderPos-20 : lastHeaderPos]
// 				last20bytes = append(last20bytes, '@')
// 				last20bytes = append(last20bytes, cp[lastHeaderPos:lastHeaderPos+20]...)
// 			} else {
// 				last20bytes = cp[0 : lastHeaderPos+20]
// 			}
// 			slog.WarnContext(ctx, "magic bytes not found", "magic", string(dat), "lastHeaderPos", lastHeaderPos, "last20bytes", fmt.Sprintf("%s", last20bytes))
// 			return "", -2
// 		}

// 		fnsData := cpioData[lastHeaderPos+94 : lastHeaderPos+94+8]
// 		dsData := cpioData[lastHeaderPos+54 : lastHeaderPos+54+8]

// 		fileNameSize, err := strconv.ParseUint(string(fnsData), 16, 32)
// 		if err != nil {
// 			slog.WarnContext(ctx, "error parsing fileNameSize", "error", err)
// 			return "", -2
// 		}

// 		dataSize, err := strconv.ParseUint(string(dsData), 16, 32)
// 		if err != nil {
// 			slog.WarnContext(ctx, "error parsing dataSize", "error", err)
// 			return "", -2
// 		}
// 		var fileNamePadding, dataPadding int
// 		if fileNameSize > 4 {
// 			// we need to round the file name size up to the nearest multiple of 8
// 			fileNamePadding = (int(fileNameSize)+3)&^3 - int(fileNameSize)
// 		}
// 		if dataSize > 4 {
// 			// // we need to round the data size up to the nearest multiple of 8
// 			dataPadding = (int(dataSize)+3)&^3 - int(dataSize)
// 		}

// 		// fileNamePadding := alignPadding(int(fileNameSize), 4)
// 		// dataPadding := alignPadding(int(dataSize), 4)

// 		nextHeaderPos := lastHeaderPos + 110 + int(fileNameSize) + int(fileNamePadding) + int(dataSize) + int(dataPadding) - 2
// 		lastFileName := string(cpioData[lastHeaderPos+110 : lastHeaderPos+110+int(fileNameSize)])

// 		slog.InfoContext(ctx, "nextHeaderPos",
// 			// "hdr", hdr,
// 			// "bytes", wrk,
// 			// "wrk", hex.EncodeToString(wrk),

// 			"lastHeaderPos", lastHeaderPos,
// 			"fileNameSize", fileNameSize,
// 			"dataSize", dataSize,
// 			"fileNamePadding", fileNamePadding,
// 			"dataPadding", dataPadding,
// 			"nextHeaderPos", nextHeaderPos,
// 			"lastFileName", lastFileName,
// 		)

// 		return lastFileName, nextHeaderPos
// 	}

// 	pos := 0

// 	for pos < len(cpioData) {
// 		lastFileName, nextHeaderPos := nextHeaderPos(pos)

// 		if nextHeaderPos == -2 {
// 			return -1
// 		}
// 		if lastFileName == filename {
// 			return pos + 110
// 		}
// 		pos = nextHeaderPos
// 	}

// 	return -1
// }

// // FindTrailerInCPIO locates the TRAILER!!! record in a CPIO archive
// func FindTrailerInCPIO(cpioData []byte) CPIOSearchResult {
// 	trailerSignature := []byte("TRAILER!!!")
// 	trailerPos := bytes.LastIndex(cpioData, trailerSignature)
// 	if trailerPos == -1 {
// 		return CPIOSearchResult{Found: false}
// 	}

// 	// Search backward to find the header containing this trailer
// 	magicBytes := []byte(initramfs.Magic_070701)
// 	headerPos := bytes.LastIndex(cpioData[:trailerPos], magicBytes)
// 	if headerPos == -1 {
// 		return CPIOSearchResult{Found: false}
// 	}

// 	// Double-check that this header really points to the trailer
// 	filenameOffset := headerPos + 110
// 	if filenameOffset < trailerPos &&
// 		bytes.Equal(cpioData[filenameOffset:filenameOffset+9], trailerSignature) {

// 		// Extract the data size from the header (should be 0 for trailer)
// 		dataSize := extractHeaderField(cpioData, headerPos+54, 8)

// 		return CPIOSearchResult{
// 			Found:            true,
// 			HeaderPosition:   headerPos,
// 			FilenamePosition: filenameOffset,
// 			DataPosition:     0, // No data for trailer
// 			Size:             dataSize,
// 		}
// 	}

// 	return CPIOSearchResult{Found: false}
// }

// extractHeaderField extracts a numeric field from a CPIO header by parsing
// the ASCII hex representation starting at the given offset
func extractHeaderField(cpioData []byte, offset, length int) uint32 {
	if offset+length > len(cpioData) {
		return 0
	}

	hexStr := string(cpioData[offset : offset+length])
	val, err := strconv.ParseUint(hexStr, 16, 32)
	if err != nil {
		return 0
	}

	return uint32(val)
}
