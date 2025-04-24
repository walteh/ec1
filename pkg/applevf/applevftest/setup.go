package applevftest

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/cavaliergopher/grab/v3"
	"github.com/mholt/archives"
)

func SetupOS(t *testing.T, prov OsProvider) error {
	ctx := t.Context()
	tmpDir := t.TempDir()
	url := prov.URL()

	if true {
	}

	extractedTmpDir := filepath.Join(tmpDir, "extracted")

	cacheDir, err := cacheDir(url)
	if err != nil {
		return err
	}

	extractedCachedZip := filepath.Join(cacheDir, "extracted.zip")
	cacheFile := filepath.Join(cacheDir, filepath.Base(url))

	if _, err := os.Stat(cacheFile); err != nil {
		err = downloadURLToFile(ctx, url, cacheFile)
		if err != nil {
			return err
		}
	}

	if _, err := os.Stat(extractedCachedZip); err != nil {

		err = prov.Uncompress(ctx, cacheFile, extractedTmpDir)
		if err != nil {
			return err
		}

		err = compressDirToZip(ctx, extractedTmpDir, extractedCachedZip)
		if err != nil {
			return err
		}

	}

	if _, err := os.Stat(extractedTmpDir); err != nil {
		err = decompressZipToDir(ctx, extractedCachedZip, extractedTmpDir)
		if err != nil {
			return err
		}
	}

	err = prov.Initialize(ctx, extractedTmpDir)
	if err != nil {
		return err
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
		return err
	}

	err = os.Rename(resp.Filename, file)
	if err != nil {
		return err
	}

	return nil
}

func compressDirToZip(ctx context.Context, dir string, zipFile string) error {

	slog.InfoContext(ctx, "compressing dir to zip", "dir", dir, "zipFile", zipFile)

	zipper := archives.Zip{}
	fsys, err := archives.FilesFromDisk(ctx, &archives.FromDiskOptions{}, map[string]string{
		dir: ".",
	})
	if err != nil {
		return err
	}

	file, err := os.Create(zipFile)
	if err != nil {
		return err
	}
	defer file.Close()

	err = zipper.Archive(ctx, file, fsys)
	if err != nil {
		return err
	}

	return nil

}

func decompressZipToDir(ctx context.Context, zipFile string, dir string) error {
	slog.InfoContext(ctx, "decompressing zip to dir", "zipFile", zipFile, "dir", dir)

	zipper := archives.Zip{}

	file, err := os.Open(zipFile)
	if err != nil {
		return err
	}
	defer file.Close()

	if err = zipper.Extract(ctx, file, func(ctx context.Context, info archives.FileInfo) error {
		dest := filepath.Join(dir, info.Name())

		file, err := info.Open()
		if err != nil {
			return err
		}
		defer file.Close()

		err = os.MkdirAll(filepath.Dir(dest), 0755)
		if err != nil {
			return err
		}

		// create a new file in the cache dir
		outfile, err := os.Create(dest)
		if err != nil {
			return err
		}
		defer outfile.Close()

		_, err = io.Copy(outfile, file)
		if err != nil {
			return err
		}

		return nil
	}); err != nil {
		return err
	}

	return nil
}
