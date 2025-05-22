package initramfs_test

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/u-root/u-root/pkg/cpio"

	"github.com/walteh/ec1/pkg/initramfs"
	"github.com/walteh/ec1/pkg/testing/tlog"
)

// mockInitBinary is a simple mock binary for testing

// generateTestCpio creates a simple CPIO file for testing
func generateTestCpio(t *testing.T, files map[string]string) io.ReadSeekCloser {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "test.cpio")

	f, err := os.Create(outputPath)
	require.NoError(t, err)

	img := cpio.Newc.Writer(f)

	records := make([]cpio.Record, 0)

	for k, v := range files {
		records = append(records, cpio.StaticFile(k, v, 0644))
	}

	// Write all records
	err = cpio.WriteRecords(img, records)
	require.NoError(t, err)

	// Write trailer
	err = cpio.WriteTrailer(img)
	require.NoError(t, err)

	f.Seek(0, io.SeekStart)

	return f
}

// TestInjectInitBinaryToInitramfsCpio focuses on testing the CPIO generation logic
// We will analyze the issues with the real function by checking its output
func TestInjectFileToCpio(t *testing.T) {
	ctx := tlog.SetupSlogForTestWithContext(t, t.Context())
	originalInitContent := "#!/bin/sh\necho 'Original init'\n"
	// Create a test CPIO file
	cpioPath := generateTestCpio(t, map[string]string{
		"init": originalInitContent,
		"sh":   "#!/bin/sh\necho 'shell'\n",     // dummy file
		"pwd":  "root:x:0:0:root:/root:/bin/sh", // dummy file
	})
	defer cpioPath.Close()

	fileHeadersBefore, fileDataBefore, err := initramfs.ExtractFilesFromCpio(ctx, cpioPath)
	require.NoError(t, err)

	cpioPath.Seek(0, io.SeekStart)

	var mockInitBinary = "#!/bin/sh\necho 'This is a mock init binary'\n"

	rdr, err := initramfs.InjectFileToCpio(ctx, cpioPath, initramfs.NewExecHeader("init"), []byte(mockInitBinary))
	require.NoError(t, err)

	cpioPath.Seek(0, io.SeekStart)

	fileHeadersAfter, fileDataAfter, err := initramfs.ExtractFilesFromCpio(ctx, rdr)
	require.NoError(t, err)

	cpioPath.Seek(0, io.SeekStart)

	assert.Equal(t, "init", fileHeadersBefore["init"].Filename)
	assert.Equal(t, originalInitContent, string(fileDataBefore["init"]))

	assert.Equal(t, "init", fileHeadersAfter["init"].Filename)
	assert.Equal(t, string(mockInitBinary), string(fileDataAfter["init"]))

	assert.Equal(t, "iniz", fileHeadersAfter["iniz"].Filename)
	assert.Equal(t, originalInitContent, string(fileDataAfter["iniz"]))
}
