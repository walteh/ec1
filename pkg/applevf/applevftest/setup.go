package applevftest

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cavaliergopher/grab/v3"
	"github.com/mholt/archives"
	"gitlab.com/tozd/go/errors"
)

func ShortTestTempDir(t *testing.T) string {
	// hash the test name, take the first 8 characters
	hash := sha256.Sum256([]byte(t.Name()))
	tmpdir := os.TempDir()
	testTmpDir := t.TempDir()
	dir := filepath.Join(tmpdir, fmt.Sprintf("t%x", hash[:4]), filepath.Base(testTmpDir))
	slog.InfoContext(t.Context(), "creating short test temp dir", "dir", dir)
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		t.Fatalf("creating temp dir: %s", err)
	}
	t.Cleanup(func() {
		os.RemoveAll(dir)
	})
	RegisterRedactedLogValue(t, dir, "[short-tmp-dir]")
	return dir
}

const MagicDecompressedFileName = "base"

func FullSetupOS(t *testing.T, prov OsProvider) *testVM {
	tmpDir := ShortTestTempDir(t)
	err := SetupOS(t, prov, tmpDir)
	if err != nil {
		t.Fatalf("setting up os: %s", err)
	}
	return NewTestVM(t, prov, tmpDir)
}

func SetupOS(t *testing.T, prov OsProvider, tmpDir string) error {
	ctx := t.Context()
	// create a new temp dir for the os image

	url := prov.URL()

	if true {
	}

	extractedTmpDir := filepath.Join(tmpDir, "extracted")

	cacheDir, err := cacheDir(url)
	if err != nil {
		return errors.Errorf("getting cache dir: %w", err)
	}

	extractedCachedZip := filepath.Join(cacheDir, "extracted.zip")
	cacheFile := filepath.Join(cacheDir, filepath.Base(url))

	if _, err := os.Stat(cacheFile); err != nil {
		err = downloadURLToFile(ctx, url, cacheFile)
		if err != nil {
			return errors.Errorf("downloading url to file: %w", err)
		}
	}

	if _, err := os.Stat(extractedCachedZip); err != nil {

		err = extractIntoDir(ctx, cacheFile, extractedTmpDir)
		if err != nil {
			return errors.Errorf("extracting into dir: %w", err)
		}

		err = compressDirToZip(ctx, extractedTmpDir, extractedCachedZip)
		if err != nil {
			return errors.Errorf("compressing dir to zip: %w", err)
		}
	}

	if _, err := os.Stat(extractedTmpDir); err != nil {
		err = extractIntoDir(ctx, extractedCachedZip, extractedTmpDir)
		if err != nil {
			return errors.Errorf("extracting into dir: %w", err)
		}
	}

	err = prov.Initialize(ctx, extractedTmpDir)
	if err != nil {
		return errors.Errorf("initializing os provider: %w", err)
	}

	slog.InfoContext(ctx, "OS provider initialized", "url", url, "cacheFile", cacheFile, "extractedTmpDir", extractedTmpDir)

	return nil
}

func downloadURLToFile(ctx context.Context, url string, file string) error {
	slog.InfoContext(ctx, "downloading url to file", "url", url, "file", file)

	// move the file to the cache
	err := os.MkdirAll(filepath.Dir(file), 0755)
	if err != nil {
		return err
	}

	grab.DefaultClient.UserAgent = "ec1"
	resp, err := grab.Get(filepath.Dir(file), url)
	if err != nil {
		return errors.Errorf("downloading url: %w", err)
	}

	err = os.Rename(resp.Filename, file)
	if err != nil {
		return errors.Errorf("renaming file: %w", err)
	}

	return nil
}

func getDirSize(dir string) (int64, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return 0, errors.Errorf("reading dir: %w", err)
	}

	var size int64
	for _, file := range files {
		info, err := file.Info()
		if err != nil {
			return 0, errors.Errorf("getting file info: %w", err)
		}
		size += info.Size()
	}

	return size, nil
}

func compressDirToZip(ctx context.Context, dir string, zipFile string) error {

	preCompressionSize, err := getDirSize(dir)
	if err != nil {
		return err
	}

	slog.InfoContext(ctx, "compressing dir to zip", "dir", dir, "zipFile", zipFile, "preCompressionSize", preCompressionSize)

	zipper := archives.Zip{
		Compression: archives.ZipMethodZstd,
	}
	fsys, err := archives.FilesFromDisk(ctx, &archives.FromDiskOptions{}, map[string]string{
		dir: ".",
	})
	if err != nil {
		return errors.Errorf("getting files from disk: %w", err)
	}

	file, err := os.Create(zipFile)
	if err != nil {
		return errors.Errorf("creating zip file: %w", err)
	}
	defer file.Close()

	err = zipper.Archive(ctx, file, fsys)
	if err != nil {
		return errors.Errorf("archiving files: %w", err)
	}

	postCompressionSize, err := os.Stat(zipFile)
	if err != nil {
		return errors.Errorf("getting zip file size: %w", err)
	}

	slog.InfoContext(ctx, "compressed dir to zip", "dir", dir, "zipFile", zipFile, "preCompressionSize", preCompressionSize, "postCompressionSize", postCompressionSize.Size())

	return nil

}

func saveArchivesFileToDir(ctx context.Context, info archives.FileInfo, dir string) error {
	dest := filepath.Join(dir, info.Name())

	if info.IsDir() {
		return os.MkdirAll(dest, 0755)
	}

	file, err := info.Open()
	if err != nil {
		return errors.Errorf("opening file: %w", err)
	}
	defer file.Close()

	err = os.MkdirAll(filepath.Dir(dest), 0755)
	if err != nil {
		return errors.Errorf("creating dir: %w", err)
	}

	// create a new file in the cache dir
	outfile, err := os.Create(dest)
	if err != nil {
		return errors.Errorf("creating file: %w", err)
	}
	defer outfile.Close()

	_, err = io.Copy(outfile, file)
	if err != nil {
		return errors.Errorf("copying file: %w", err)
	}

	return nil
}

func extractIntoDir(ctx context.Context, file string, dir string) error {
	slog.InfoContext(ctx, "reformatting unknown file into dir", "file", file, "dir", dir)
	inputFile, err := os.Open(file)
	if err != nil {
		return err
	}
	defer inputFile.Close()

	// make sure the dir exists
	err = os.MkdirAll(dir, 0755)
	if err != nil {
		return errors.Errorf("creating dir: %w", err)
	}

	fmt, rdr, err := archives.Identify(ctx, file, inputFile)
	if err != nil && err != archives.NoMatch { // on no match we just copy the file
		return errors.Errorf("identifying file: %w", err)
	}

	if archival, ok := fmt.(archives.Archival); ok {
		err = archival.Extract(ctx, rdr, func(ctx context.Context, info archives.FileInfo) error {
			return saveArchivesFileToDir(ctx, info, dir)
		})
		if err != nil {
			return errors.Errorf("extracting archive: %w", err)
		}
		slog.InfoContext(ctx, "reformatted archive", "file", file, "dir", dir)
		return nil
	}

	if compression, ok := fmt.(archives.Compression); ok {
		rdrz, err := compression.OpenReader(rdr)
		if err != nil {
			return errors.Errorf("opening compression: %w", err)
		}
		defer rdrz.Close()
		rdr = rdrz
	}

	out := renameExtensionOfExtractedFile(ctx, fmt, file)

	out = filepath.Join(dir, out)

	outputFile, err := os.Create(out)
	if err != nil {
		return errors.Errorf("creating file: %w", err)
	}
	defer outputFile.Close()

	_, err = io.Copy(outputFile, rdr)
	if err != nil {
		return errors.Errorf("copying file: %w", err)
	}

	slog.InfoContext(ctx, "reformatted file", "in", file, "out", out)

	return nil

}

func renameExtensionOfExtractedFile(ctx context.Context, afmt archives.Format, file string) string {
	// try to remove the extension, otherwise return the .reformated file

	out := filepath.Base(file)

	if afmt == nil {
		// no extension, return the original file name
		return out
	}

	out = strings.TrimSuffix(out, afmt.Extension())
	if out == filepath.Base(file) {

		// no change, add a .reformatted extension
		slog.WarnContext(ctx, "no change, adding .reformatted extension", "file", file, "out", out)
		return out + ".reformatted"
	}

	return out
}
