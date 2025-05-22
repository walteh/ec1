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

// mockInitBinary is a simple mock binary for testing

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

func TestFastInjectFromMock(t *testing.T) {
	cpioPath := generateTestCpio(t, map[string][]byte{
		"init": []byte("#!/bin/sh\necho 'original_init'\n"),
	})
	defer cpioPath.Close()

	cpioPath.Seek(0, io.SeekStart)

	for i := 0; i < 4; i++ {
		t.Run(fmt.Sprintf("padding_%d", i), func(t *testing.T) {
			thirtyTwoCharacterString := "#!/bin/sh\n\necho 'mock_init' %s\n"
			mockInitBinary := fmt.Sprintf(thirtyTwoCharacterString, strings.Repeat("a", i))
			testFastInjectFileToCpio(t, mockInitBinary, cpioPath)
		})
	}
}

func TestFastInjectFileToCpioFromEmbed(t *testing.T) {
	cpioPath := testdataembed.MustCreateTmpFileFor(t, testdata.Testdata(), "start.cpio")
	defer cpioPath.Close()

	cpioPath.Seek(0, io.SeekStart)

	mockInitBinary := "#!/bin/sh\necho 'mock_init' with padding for some extra struff\n"

	testFastInjectFileToCpio(t, mockInitBinary, cpioPath)
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

func TestFastInjectFileToCpioFromEmbedLarge(t *testing.T) {
	cpioPath := openLargeCpio(t)
	defer cpioPath.Close()

	mockInitBinary := "#!/bin/sh\necho 'mock_init' with padding for some extra struff\n"

	testFastInjectFileToCpio(t, mockInitBinary, cpioPath)
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

func testFastInjectFileToCpio(t *testing.T, mockInitBinary string, cpioPath io.ReadSeekCloser) {
	ctx := tlog.SetupSlogForTestWithContext(t, t.Context())

	fileHeadersBefore, fileDataBefore, err := initramfs.ExtractFilesFromCpio(ctx, cpioPath)
	require.NoError(t, err)

	cpioPath.Seek(0, io.SeekStart)

	dat, err := io.ReadAll(cpioPath)
	require.NoError(t, err)
	// fmt.Println("dat", dat)
	fmt.Println("countTrailingZeroBytes", countTrailingZeroBytes(dat))

	cpioPath.Seek(0, io.SeekStart)

	// pre, closer := tlog.TeeToDownloadsFolder(cpioPath, "BEFORE.initramfs.cpio")
	// defer closer.Close()

	// var mockInitBinary = "#!/bin/sh\necho 'mock_init' with padding for some extra struff\n"

	fastReader := initramfs.StreamInjectHyper(ctx, cpioPath, initramfs.NewExecHeader("init"), []byte(mockInitBinary))
	require.NoError(t, err)

	cpioPath.Seek(0, io.SeekStart)

	fastData, err := io.ReadAll(fastReader)
	require.NoError(t, err)

	fmt.Println("fast countTrailingZeroBytes", countTrailingZeroBytes(fastData))

	cpioPath.Seek(0, io.SeekStart)

	slowReader, err := initramfs.InjectFileToCpio(ctx, cpioPath, initramfs.NewExecHeader("init"), []byte(mockInitBinary))
	require.NoError(t, err)

	// srd, sfle := tlog.TeeToDownloadsFolder(slrdr, "SLOW.initramfs.cpio")
	// defer sfle.Close()

	// rdrz, rfle := tlog.TeeToDownloadsFolder(rdr, "FAST.initramfs.cpio")
	// defer rfle.Close()

	slowData, err := io.ReadAll(slowReader)
	require.NoError(t, err)

	fileHeadersAfterFast, fileDataAfterFast, err := initramfs.ExtractFilesFromCpio(ctx, bytes.NewReader(fastData))
	require.NoError(t, err)

	fileHeadersAfterSLow, fileDataAfterSLow, err := initramfs.ExtractFilesFromCpio(ctx, bytes.NewReader(slowData))
	require.NoError(t, err)

	cpioPath.Seek(0, io.SeekStart)

	for _, h := range initramfs.OrderedByInode(fileHeadersAfterFast) {
		t.Logf("fileHeadersAfterFast: %s (inode: %d)", h.Filename, h.Inode)
	}

	t.Logf("--------------------------------")

	for _, h := range initramfs.OrderedByInode(fileHeadersAfterSLow) {
		t.Logf("fileHeadersAfterSLow: %s (inode: %d)", h.Filename, h.Inode)
	}

	require.NotNil(t, fileHeadersBefore["init"])
	require.NotNil(t, fileDataBefore["init"])
	require.NotNil(t, fileHeadersAfterSLow["init"])
	require.NotNil(t, fileDataAfterSLow["init"])
	require.NotNil(t, fileHeadersAfterSLow["iniz"])
	require.NotNil(t, fileDataAfterSLow["iniz"])
	require.NotNil(t, fileHeadersAfterFast["init"])
	require.NotNil(t, fileDataAfterFast["init"])
	require.NotNil(t, fileHeadersAfterFast["iniz"])
	require.NotNil(t, fileDataAfterFast["iniz"])

	assert.Equal(t, "init", fileHeadersBefore["init"].Filename)
	assert.NotEqual(t, mockInitBinary, string(fileDataBefore["init"]))

	assert.Equal(t, "init", fileHeadersAfterSLow["init"].Filename)
	assert.Equal(t, string(mockInitBinary), string(fileDataAfterSLow["init"]))

	require.NotNil(t, fileHeadersAfterFast["init"])
	assert.Equal(t, "init", fileHeadersAfterFast["init"].Filename)
	assert.Equal(t, string(mockInitBinary), string(fileDataAfterFast["init"]))

	assert.Equal(t, "iniz", fileHeadersAfterSLow["iniz"].Filename)
	assert.NotEqual(t, mockInitBinary, string(fileDataAfterSLow["iniz"]))

	require.NotNil(t, fileHeadersAfterFast["iniz"])
	assert.Equal(t, "iniz", fileHeadersAfterFast["iniz"].Filename)
	assert.NotEqual(t, mockInitBinary, string(fileDataAfterFast["iniz"]))

	// make sure the final bytes match
	assert.Equal(t, fileDataAfterSLow["init"], fileDataAfterFast["init"])

	// make sure the final bytes match
	assert.Equal(t, fileDataAfterSLow["iniz"], fileDataAfterFast["iniz"])

	assert.Equal(t, slowData, fastData)
}

func generateLargeMockInitBinary(size int) []byte {
	mockInitBinary := make([]byte, size)
	for i := range mockInitBinary {
		mockInitBinary[i] = byte(i % 256)
	}
	return mockInitBinary
}

// BenchmarkFastInjectFileToCpio benchmarks the fast CPIO injection function
func BenchmarkFastInjectFileToCpio(b *testing.B) {
	ctx := context.Background()

	slog.SetDefault(slog.New(slog.DiscardHandler))

	// Create a large mock init binary for benchmarking
	mockInitBinary := make([]byte, 1024*1024) // 1MB
	for i := range mockInitBinary {
		mockInitBinary[i] = byte(i % 256)
	}

	// dats := map[string][]byte{
	// 	"init": openLargeCpio(b),
	// }
	// for i := 0; i < 1024; i++ {
	// 	dats[fmt.Sprintf("file-%d", i)] =
	// }

	f := openLargeCpio(b)
	defer f.Close()

	b.Run("stream", func(b *testing.B) {
		for b.Loop() {
			f.Seek(0, io.SeekStart)
			reader := initramfs.StreamInject(ctx, f, initramfs.NewExecHeader("init"), mockInitBinary)
			_, err := io.Copy(io.Discard, reader)
			require.NoError(b, err)
		}
	})

	b.Run("stream-optimized", func(b *testing.B) {
		for b.Loop() {
			f.Seek(0, io.SeekStart)
			reader := initramfs.StreamInjectOptimized(ctx, f, initramfs.NewExecHeader("init"), mockInitBinary)
			_, err := io.Copy(io.Discard, reader)
			require.NoError(b, err)
		}
	})

	b.Run("stream-ultra", func(b *testing.B) {
		for b.Loop() {
			f.Seek(0, io.SeekStart)
			reader := initramfs.StreamInjectUltra(ctx, f, initramfs.NewExecHeader("init"), mockInitBinary)
			_, err := io.Copy(io.Discard, reader)
			require.NoError(b, err)
		}
	})

	b.Run("stream-hyper", func(b *testing.B) {
		for b.Loop() {
			f.Seek(0, io.SeekStart)
			reader := initramfs.StreamInjectHyper(ctx, f, initramfs.NewExecHeader("init"), mockInitBinary)
			_, err := io.Copy(io.Discard, reader)
			require.NoError(b, err)
		}
	})

	b.Run("stream-insane", func(b *testing.B) {
		for b.Loop() {
			f.Seek(0, io.SeekStart)
			reader := initramfs.StreamInjectInsane(ctx, f, initramfs.NewExecHeader("init"), mockInitBinary)
			_, err := io.Copy(io.Discard, reader)
			require.NoError(b, err)
		}
	})

	b.Run("stream-blazing", func(b *testing.B) {
		for b.Loop() {
			f.Seek(0, io.SeekStart)
			reader := initramfs.StreamInjectBlazingFast(ctx, f, initramfs.NewExecHeader("init"), mockInitBinary)
			_, err := io.Copy(io.Discard, reader)
			require.NoError(b, err)
		}
	})

	b.Run("fast", func(b *testing.B) {
		for b.Loop() {
			f.Seek(0, io.SeekStart)
			reader, err := initramfs.FastInjectFileToCpio(ctx, f, initramfs.NewExecHeader("init"), mockInitBinary)
			require.NoError(b, err)
			_, err = io.Copy(io.Discard, reader)
			require.NoError(b, err)
		}
	})

	b.Run("fast-hyper", func(b *testing.B) {
		for b.Loop() {
			f.Seek(0, io.SeekStart)
			reader, err := initramfs.FastInjectFileToCpioHyper(ctx, f, initramfs.NewExecHeader("init"), mockInitBinary)
			require.NoError(b, err)
			_, err = io.Copy(io.Discard, reader)
			require.NoError(b, err)
		}
	})

	b.Run("fast-insane", func(b *testing.B) {
		for b.Loop() {
			f.Seek(0, io.SeekStart)
			reader, err := initramfs.FastInjectFileToCpioInsane(ctx, f, initramfs.NewExecHeader("init"), mockInitBinary)
			require.NoError(b, err)
			_, err = io.Copy(io.Discard, reader)
			require.NoError(b, err)
		}
	})

	b.Run("fast-blazing", func(b *testing.B) {
		for b.Loop() {
			f.Seek(0, io.SeekStart)
			reader, err := initramfs.FastInjectFileToCpioBlazingFast(ctx, f, initramfs.NewExecHeader("init"), mockInitBinary)
			require.NoError(b, err)
			_, err = io.Copy(io.Discard, reader)
			require.NoError(b, err)
		}
	})

	b.Run("slow", func(b *testing.B) {
		for b.Loop() {
			f.Seek(0, io.SeekStart)
			reader, err := initramfs.InjectFileToCpio(ctx, f, initramfs.NewExecHeader("init"), mockInitBinary)
			_, err = io.Copy(io.Discard, reader)
			require.NoError(b, err)
		}
	})

}
