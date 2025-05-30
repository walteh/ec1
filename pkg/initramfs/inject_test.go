package initramfs_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mholt/archives"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	oinitramfs "go.pdmccormick.com/initramfs"

	"github.com/walteh/ec1/pkg/initramfs"
	"github.com/walteh/ec1/pkg/initramfs/testdata"
	"github.com/walteh/ec1/pkg/testing/testdataembed"
	"github.com/walteh/ec1/pkg/testing/tlog"
)

var injectors = map[string]initramfs.InitramfsFileInjectorFunc{
	"stream_inject_original": initramfs.StreamInjectOriginal,
	// "stream_inject_pooled":   initramfs.StreamInjectPooled,
	"stream_inject_hyper":    initramfs.StreamInjectHyper,
	"stream_inject_library":  initramfs.StreamInjectLibrary,
	"stream_inject_read_all": initramfs.StreamInjectReadAll,
	"stream_inject_simple":   initramfs.StreamInjectSimple,
}

func logicalInjectorTest(t *testing.T, newFileName string, mockInitBinary string, cpioPath io.ReadSeekCloser, truthReaderName string, readers map[string]initramfs.InitramfsFileInjectorFunc) {
	ctx := tlog.SetupSlogForTestWithContext(t, t.Context())

	var err error

	init := newFileName
	iniz := replaceLastLetterWithZ(newFileName)
	initHeader := initramfs.NewExecHeader(init)

	fileHeadersBefore, fileDataBefore, err := initramfs.ExtractFilesFromCpio(ctx, cpioPath)
	require.NoError(t, err)

	require.NotNil(t, fileHeadersBefore[init], "the input cpio headers should have an init file")
	require.NotNil(t, fileDataBefore[init], "the input cpio data should have an init file")
	require.Equal(t, init, fileHeadersBefore[init].Filename, "the input cpio init file should be named init")
	require.NotEqual(t, mockInitBinary, string(fileDataBefore[init]), "the input cpio init file should not be the mock init binary")

	cpioPath.Seek(0, io.SeekStart)

	type resultd struct {
		name    string
		reader  initramfs.InitramfsFileInjectorFunc
		headers map[string]*oinitramfs.Header
		data    map[string][]byte
		rawData []byte
	}

	var truth *resultd

	args := make([]*resultd, 0, len(readers))

	for name, reader := range readers {
		argd := resultd{
			name:   name,
			reader: reader,
		}

		if name == truthReaderName {
			truth = &argd
		}

		args = append(args, &argd)
	}

	runner := func(t *testing.T, arg *resultd) {

		cpioPath.Seek(0, io.SeekStart)

		fastReader := arg.reader(ctx, cpioPath, initHeader, []byte(mockInitBinary))
		require.NoError(t, err)

		arg.rawData, err = io.ReadAll(fastReader)
		require.NoError(t, err)

		arg.headers, arg.data, err = initramfs.ExtractFilesFromCpio(ctx, bytes.NewReader(arg.rawData))
		require.NoError(t, err)

		cpioPath.Seek(0, io.SeekStart)

		require.NotNil(t, arg.headers[init], "the output should have an init file")
		assert.Equal(t, init, arg.headers[init].Filename, "the output should have an init file")
		assert.Equal(t, string(mockInitBinary), string(arg.data[init]), "the output should have the mock init binary")

		require.NotNil(t, arg.headers[iniz], "the output should have an iniz file")
		assert.Equal(t, iniz, arg.headers[iniz].Filename, "the output should have an iniz file")
		assert.NotEqual(t, mockInitBinary, string(arg.data[iniz]), "the output should not have the mock init binary")

		if arg.name == truthReaderName {
			return
		}

		require.NotNil(t, truth.rawData, "the truth reader should be in the results", truthReaderName)

		assert.Equal(t, truth.headers[init], arg.headers[init], "the output should have the same headers", arg.name)
		assert.Equal(t, truth.data[init], arg.data[init], "the output should have the same data", arg.name)
		assert.Equal(t, truth.rawData, arg.rawData, "the output should have the same raw data", arg.name)
		assert.Equal(t, truth.headers[iniz], arg.headers[iniz], "the output should have the same headers", arg.name)
		assert.Equal(t, truth.data[iniz], arg.data[iniz], "the output should have the same data", arg.name)
		assert.Equal(t, truth.rawData, arg.rawData, "the output should have the same raw data", arg.name)
	}

	runner(t, truth)

	for _, arg := range args {
		t.Run(arg.name, func(t *testing.T) {
			runner(t, arg)
		})
	}

}

func TestInjectorLogicMockFileDifferentPadding(t *testing.T) {

	thirtyTwoCharacterString := "#!/bin/sh\n\necho 'mock_init' %s\n"
	fourCharacterName := "init"

	for i := 0; i < 4; i++ {
		t.Run(fmt.Sprintf("binary_padding_%d", i), func(t *testing.T) {
			mockInitBinary := fmt.Sprintf(thirtyTwoCharacterString, strings.Repeat("a", i))
			mockInitName := fourCharacterName
			cpioPath := generateTestCpio(t, map[string][]byte{mockInitName: []byte("#!/bin/sh\necho 'original_init'\n")})
			defer cpioPath.Close()
			cpioPath.Seek(0, io.SeekStart)
			logicalInjectorTest(t, mockInitName, mockInitBinary, cpioPath, "stream_inject_library", injectors)
		})
		t.Run(fmt.Sprintf("name_padding_%d", i), func(t *testing.T) {
			mockInitBinary := thirtyTwoCharacterString
			mockInitName := fmt.Sprintf("%s_%s", fourCharacterName, strings.Repeat("a", i))
			cpioPath := generateTestCpio(t, map[string][]byte{mockInitName: []byte("#!/bin/sh\necho 'original_init'\n")})
			defer cpioPath.Close()
			cpioPath.Seek(0, io.SeekStart)
			logicalInjectorTest(t, mockInitName, mockInitBinary, cpioPath, "stream_inject_library", injectors)
		})
		t.Run(fmt.Sprintf("both_padding_%d", i), func(t *testing.T) {
			mockInitBinary := fmt.Sprintf(thirtyTwoCharacterString, strings.Repeat("a", i))
			mockInitName := fmt.Sprintf("%s_%s", fourCharacterName, strings.Repeat("a", i))
			cpioPath := generateTestCpio(t, map[string][]byte{mockInitName: []byte("#!/bin/sh\necho 'original_init'\n")})
			defer cpioPath.Close()
			cpioPath.Seek(0, io.SeekStart)
			logicalInjectorTest(t, mockInitName, mockInitBinary, cpioPath, "stream_inject_library", injectors)
		})
	}
}

func TestInjectorLogicSmallFile(t *testing.T) {
	cpioPath := testdataembed.MustCreateTmpFileFor(t, testdata.Testdata(), "start.cpio")
	defer cpioPath.Close()

	cpioPath.Seek(0, io.SeekStart)

	mockInitBinary := "#!/bin/sh\necho 'mock_init' with padding for some extra struff\n"

	logicalInjectorTest(t, "init", mockInitBinary, cpioPath, "stream_inject_library", injectors)
}

func TestInjectorLogicLargeFile(t *testing.T) {
	cpioPath := openLargeCpio(t)
	defer cpioPath.Close()

	mockInitBinary := "#!/bin/sh\necho 'mock_init' with padding for some extra struff\n"

	logicalInjectorTest(t, "init", mockInitBinary, cpioPath, "stream_inject_library", injectors)
}

// BenchmarkInjectors benchmarks the injectors
func BenchmarkInjectors(b *testing.B) {
	ctx := context.Background()

	slog.SetDefault(slog.New(slog.DiscardHandler))

	// Create a large mock init binary for benchmarking
	mockInitBinary := make([]byte, 1024*1024) // 1MB
	for i := range mockInitBinary {
		mockInitBinary[i] = byte(i % 256)
	}

	f := openLargeCpio(b)
	defer f.Close()

	for name, reader := range injectors {
		b.Run(name, func(b *testing.B) {
			for b.Loop() {
				f.Seek(0, io.SeekStart)
				reader := reader(ctx, f, initramfs.NewExecHeader("init"), mockInitBinary)
				_, err := io.Copy(io.Discard, reader)
				require.NoError(b, err)
			}
		})
	}

}

// generateTestCpio creates a simple CPIO file for testing
func generateTestCpio(t testing.TB, files map[string][]byte) io.ReadSeekCloser {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "test.cpio")

	f, err := os.Create(outputPath)
	require.NoError(t, err)

	img := oinitramfs.NewWriter(f)
	offset := uint32(0)

	for k, v := range files {
		offset++
		err := img.WriteHeader(&oinitramfs.Header{
			Filename:     k,
			Mode:         oinitramfs.Mode_FileTypeMask | oinitramfs.GroupExecute | oinitramfs.UserExecute | oinitramfs.OtherExecute,
			DataSize:     uint32(len(v)),
			Inode:        offset,
			FilenameSize: uint32(len(k)),
			Magic:        oinitramfs.Magic_070701,
		})
		require.NoError(t, err)

		_, err = img.Write(v)
		require.NoError(t, err)
	}

	err = img.WriteTrailer()
	require.NoError(t, err)

	err = img.Close()
	require.NoError(t, err)

	f, err = os.Open(outputPath)
	require.NoError(t, err)

	return f
}

func openLargeCpio(t testing.TB) io.ReadSeekCloser {
	cpioPath := testdataembed.MustCreateTmpFileFor(t, testdata.Testdata(), "large.cpio.xz")
	defer cpioPath.Close()

	reader, err := (archives.Xz{}).OpenReader(cpioPath)
	require.NoError(t, err)

	// save to a temp file
	tmpFile, err := os.CreateTemp("", "large.cpio")
	require.NoError(t, err)

	_, err = io.Copy(tmpFile, reader)
	require.NoError(t, err)

	tmpFile.Seek(0, io.SeekStart)

	return tmpFile
}

func countTrailingZeroBytes(data []byte) int {
	trailingZeros := 0
	for i := len(data) - 1; i >= 0; i-- {
		if data[i] != 0 {
			break
		}
		trailingZeros++
	}
	return trailingZeros
}

func replaceLastLetterWithZ(data string) string {
	if len(data) == 0 {
		return data
	}
	return data[:len(data)-1] + string(initramfs.Z)
}

// TestMultipleFileInjection tests that chaining multiple StreamInjectHyper calls works correctly
func TestMultipleFileInjection(t *testing.T) {
	ctx := context.Background()

	// Create a simple initial CPIO archive
	buf := bytes.NewBuffer(nil)
	w := oinitramfs.NewWriter(buf)

	// Add a simple existing file
	existingHeader := oinitramfs.Header{
		Filename:     "existing.txt",
		FilenameSize: uint32(len("existing.txt") + 1),
		Mode:         oinitramfs.Mode_File,
		NumLinks:     1,
		Magic:        oinitramfs.Magic_070701,
		DataSize:     uint32(len("existing content")),
	}
	require.NoError(t, w.WriteHeader(&existingHeader))
	_, err := w.Write([]byte("existing content"))
	require.NoError(t, err)

	// Write trailer
	require.NoError(t, w.WriteTrailer())
	require.NoError(t, w.Close())

	initialCPIO := buf.Bytes()

	// Test injecting multiple files using chained StreamInjectHyper calls
	
	// File 1: blacklist.conf
	blacklistHeader := oinitramfs.Header{
		Filename:     "etc/modprobe.d/blacklist.conf",
		FilenameSize: uint32(len("etc/modprobe.d/blacklist.conf") + 1),
		Mode:         oinitramfs.Mode_File,
		NumLinks:     1,
		Magic:        oinitramfs.Magic_070701,
	}
	blacklistContent := []byte("blacklist floppy\nblacklist pcspkr\n")

	// File 2: early.conf
	earlyHeader := oinitramfs.Header{
		Filename:     "etc/modprobe.d/early.conf",
		FilenameSize: uint32(len("etc/modprobe.d/early.conf") + 1),
		Mode:         oinitramfs.Mode_File,
		NumLinks:     1,
		Magic:        oinitramfs.Magic_070701,
	}
	earlyContent := []byte("options scsi_mod scan=manual\n")

	// File 3: vsock script
	vsockHeader := oinitramfs.Header{
		Filename:     "usr/bin/vsock-early",
		FilenameSize: uint32(len("usr/bin/vsock-early") + 1),
		Mode:         oinitramfs.Mode_File | 0755,
		NumLinks:     1,
		Magic:        oinitramfs.Magic_070701,
	}
	vsockContent := []byte("#!/bin/sh\nmodprobe vsock\n")

	// Chain the injections
	stage1Reader := initramfs.StreamInjectHyper(ctx, bytes.NewReader(initialCPIO), blacklistHeader, blacklistContent)
	stage2Reader := initramfs.StreamInjectHyper(ctx, stage1Reader, earlyHeader, earlyContent)
	stage3Reader := initramfs.StreamInjectHyper(ctx, stage2Reader, vsockHeader, vsockContent)

	// Read the final result
	finalCPIO, err := io.ReadAll(stage3Reader)
	require.NoError(t, err)

	// Parse the final CPIO to verify all files are present
	r := oinitramfs.NewReader(bytes.NewReader(finalCPIO))
	foundFiles := make(map[string][]byte)

	for {
		rec, err := r.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		if rec.Trailer() {
			break
		}

		data := make([]byte, rec.DataSize)
		if rec.DataSize > 0 {
			_, err = io.ReadFull(r, data)
			require.NoError(t, err)
		}

		foundFiles[rec.Filename] = data
	}

	// Verify all files are present and correct
	assert.Contains(t, foundFiles, "existing.txt")
	assert.Equal(t, []byte("existing content"), foundFiles["existing.txt"])

	assert.Contains(t, foundFiles, "etc/modprobe.d/blacklist.conf")
	assert.Equal(t, blacklistContent, foundFiles["etc/modprobe.d/blacklist.conf"])

	assert.Contains(t, foundFiles, "etc/modprobe.d/early.conf") 
	assert.Equal(t, earlyContent, foundFiles["etc/modprobe.d/early.conf"])

	assert.Contains(t, foundFiles, "usr/bin/vsock-early")
	assert.Equal(t, vsockContent, foundFiles["usr/bin/vsock-early"])

	t.Logf("Successfully injected %d files", len(foundFiles))
	for filename := range foundFiles {
		t.Logf("Found file: %s", filename)
	}
}

// TestSingleFileInjection tests that single file injection works (baseline)
func TestSingleFileInjection(t *testing.T) {
	ctx := context.Background()

	// Create a simple initial CPIO archive
	buf := bytes.NewBuffer(nil)
	w := oinitramfs.NewWriter(buf)

	// Add a simple existing file
	existingHeader := oinitramfs.Header{
		Filename:     "existing.txt",
		FilenameSize: uint32(len("existing.txt") + 1),
		Mode:         oinitramfs.Mode_File,
		NumLinks:     1,
		Magic:        oinitramfs.Magic_070701,
		DataSize:     uint32(len("existing content")),
	}
	require.NoError(t, w.WriteHeader(&existingHeader))
	_, err := w.Write([]byte("existing content"))
	require.NoError(t, err)

	// Write trailer
	require.NoError(t, w.WriteTrailer())
	require.NoError(t, w.Close())

	initialCPIO := buf.Bytes()

	// Test injecting a single file
	header := oinitramfs.Header{
		Filename:     "test.conf",
		FilenameSize: uint32(len("test.conf") + 1),
		Mode:         oinitramfs.Mode_File,
		NumLinks:     1,
		Magic:        oinitramfs.Magic_070701,
	}
	content := []byte("test content")

	// Inject the file
	result := initramfs.StreamInjectHyper(ctx, bytes.NewReader(initialCPIO), header, content)

	// Read the result
	finalCPIO, err := io.ReadAll(result)
	require.NoError(t, err)

	// Parse the final CPIO to verify files are present
	r := oinitramfs.NewReader(bytes.NewReader(finalCPIO))
	foundFiles := make(map[string][]byte)

	for {
		rec, err := r.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		if rec.Trailer() {
			break
		}

		data := make([]byte, rec.DataSize)
		if rec.DataSize > 0 {
			_, err = io.ReadFull(r, data)
			require.NoError(t, err)
		}

		foundFiles[rec.Filename] = data
	}

	// Verify both files are present
	assert.Contains(t, foundFiles, "existing.txt")
	assert.Equal(t, []byte("existing content"), foundFiles["existing.txt"])

	assert.Contains(t, foundFiles, "test.conf")
	assert.Equal(t, content, foundFiles["test.conf"])

	t.Logf("Successfully injected single file into CPIO with %d total files", len(foundFiles))
}
