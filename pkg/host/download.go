package host

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/cavaliergopher/grab/v3"
	"github.com/lima-vm/go-qcow2reader"
	"github.com/lima-vm/go-qcow2reader/convert"
	"github.com/mholt/archives"
	"gitlab.com/tozd/go/errors"
)

func DownloadAndExtractVMI(ctx context.Context, downloads map[string]string) (map[string]io.Reader, error) {

	files := make(map[string]io.Reader)

	group := sync.WaitGroup{}
	errs := make(chan error)
	mutex := sync.Mutex{}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	for name, url := range downloads {
		group.Add(1)
		go func() {
			defer group.Done()
			slog.InfoContext(ctx, "downloading and extracting file", "name", name, "url", url)
			rdr, err := downloadAndExtractFileFromURL(ctx, name, url)
			if err != nil {
				errs <- errors.Errorf("downloading and extracting file %s: %w", name, err)
				return
			}
			slog.InfoContext(ctx, "downloaded and extracted file", "name", name, "url", url)
			mutex.Lock()
			// save as read seeker
			files[name] = rdr
			mutex.Unlock()
		}()
	}

	go func() {
		fmt.Printf("waiting for group\n")
		group.Wait()
		fmt.Printf("group done\n")
		close(errs)
	}()

	errout := []error{}
	fmt.Printf("waiting for errs\n")
	for err := range errs {
		errout = append(errout, err)
	}

	if len(errout) > 0 {
		return nil, errors.Errorf("errors: %w", errors.Join(errout...))
	}

	return files, nil
}

func downloadAndExtractFileFromURL(ctx context.Context, name string, url string) (io.Reader, error) {

	cacheDir, err := CacheDirForURL(url)
	if err != nil {
		return nil, errors.Errorf("getting cache dir for url: %w", err)
	}

	err = os.MkdirAll(cacheDir, 0755)
	if err != nil {
		return nil, errors.Errorf("creating cache dir: %w", err)
	}

	extractedCachedZip := filepath.Join(cacheDir, name+".cached")

	if _, err := os.Stat(extractedCachedZip); err != nil {

		// tmpfile, err := os.Create(extractedCachedZip)
		// if err != nil {
		// 	return nil, errors.Errorf("creating temp file: %w", err)
		// }

		// tmpfile.Close()

		err = downloadURLToFile(ctx, url, extractedCachedZip)
		if err != nil {
			// os.Remove(tmpfile.Name())
			return nil, errors.Errorf("downloading url to file: %w", err)
		}

		slog.InfoContext(ctx, "downloaded file", "url", url, "file", extractedCachedZip)

	}

	rdrf, err := os.Open(extractedCachedZip)
	if err != nil {
		return nil, errors.Errorf("opening file: %w", err)
	}

	slog.InfoContext(ctx, "opening file", "file", extractedCachedZip)

	// z, err := (archives.Gz{}).OpenReader(rdrf)
	// if err != nil {
	// 	return nil, errors.Errorf("extracting archive: %w", err)
	// }

	return rdrf, nil

	// if strings.HasSuffix(url, ".qcow2") {
	// 	slog.InfoContext(ctx, "converting qcow2 to raw", "cacheFile", cacheFile)
	// 	updatedCacheFile := strings.TrimSuffix(cacheFile, ".qcow2") + ".raw"
	// 	outFile, err := os.Create(updatedCacheFile)
	// 	if err != nil {
	// 		return errors.Errorf("creating file: %w", err)
	// 	}
	// 	defer outFile.Close()
	// 	qcow2File, err := os.Open(cacheFile)
	// 	if err != nil {
	// 		return errors.Errorf("opening file: %w", err)
	// 	}
	// 	defer qcow2File.Close()
	// 	err = convertQcow2ToRaw(ctx, qcow2File, outFile)
	// 	if err != nil {
	// 		return errors.Errorf("converting qcow2 to raw: %w", err)
	// 	}
	// 	cacheFile = updatedCacheFile
	// }

	// if _, err := os.Stat(extractedCachedZip); err != nil {

	// }

	// if _, err := os.Stat(filepath.Join(outDir, ".extracted")); err != nil {
	// 	err = extractIntoDir(ctx, extractedCachedZip, outDir)
	// 	if err != nil {
	// 		return errors.Errorf("extracting into dir: %w", err)
	// 	}
	// }

	// err = prov.Initialize(ctx, outDir)
	// if err != nil {
	// 	return errors.Errorf("initializing os provider: %w", err)
	// }

	// slog.InfoContext(ctx, "OS provider initialized", "url", url, "cacheFile", cacheFile, "outDir", outDir)

	// return nil
}

// func SaveDownloadedFilesToCache(ctx context.Context, tmpCacheDir string) error {

// 	files, err := os.ReadDir(tmpCacheDir)
// 	if err != nil {
// 		return errors.Errorf("reading dir: %w", err)
// 	}

// 	for _, file := range files {
// 		cacheDir, err := CacheDirForURL(url)
// 		if err != nil {
// 			return errors.Errorf("getting cache dir for url: %w", err)
// 		}

// 		rdr, ok := readers[name]
// 		if !ok {
// 			return errors.Errorf("reader not found for url: %s: %s", name, url)
// 		}

// 		// readers[name] = bufrd

// 		err = os.MkdirAll(cacheDir, 0755)
// 		if err != nil {
// 			return errors.Errorf("creating cache dir: %w", err)
// 		}

// 		gzipFile, err := os.Create(filepath.Join(cacheDir, name+".cached.gz"))
// 		if err != nil {
// 			return errors.Errorf("creating gzip file: %w", err)
// 		}
// 		defer gzipFile.Close()

// 		wrt, err := (archives.Gz{}).OpenWriter(gzipFile)
// 		if err != nil {
// 			return errors.Errorf("opening gzip file: %w", err)
// 		}

// 		slog.InfoContext(ctx, "saving file to cache", "name", name, "url", url, "cacheDir", cacheDir)
// 		_, err = io.Copy(wrt, rdr)
// 		if err != nil {
// 			return errors.Errorf("copying file: %w", err)
// 		}

// 		slog.InfoContext(ctx, "saved file to cache", "name", name, "url", url, "cacheDir", cacheDir)

// 		err = wrt.Close()
// 		if err != nil {
// 			return errors.Errorf("closing gzip file: %w", err)
// 		}

// 	}

// 	return nil
// }

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

func DecompressUnknown(ctx context.Context, rdr io.Reader) (io.Reader, error) {

	compressed, err := io.ReadAll(rdr)
	if err != nil {
		return nil, errors.Errorf("reading data: %w", err)
	}

	format, rdr, err := archives.Identify(ctx, "", bytes.NewReader(compressed))
	if err != nil {
		return nil, errors.Errorf("identifying data: %w", err)
	}

	if compression, ok := format.(archives.Compression); ok {
		rdrz, err := compression.OpenReader(rdr)
		if err != nil {
			return nil, errors.Errorf("opening compression: %w", err)
		}
		defer rdrz.Close()
		rdr = rdrz
	}

	return rdr, nil
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

	// write a file to the dir called .extracted
	err = os.WriteFile(filepath.Join(dir, ".extracted"), []byte("true"), 0644)
	if err != nil {
		return errors.Errorf("writing extracted file: %w", err)
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

func ConvertQcow2ToRaw(ctx context.Context, qcow2File io.ReaderAt, rawFile io.WriterAt) error {
	img, err := qcow2reader.Open(qcow2File)
	if err != nil {
		return errors.Errorf("opening qcow2 file: %w", err)
	}
	err = convert.Convert(rawFile, img, convert.Options{})
	if err != nil {
		return errors.Errorf("converting qcow2 to raw: %w", err)
	}

	return nil
}

func convertFileToRaw(ctx context.Context, in string) (string, error) {
	if !strings.HasSuffix(in, ".qcow2") {
		return "", errors.Errorf("file is not a qcow2 file: %s", in)
	}

	updatedCacheFile := strings.TrimSuffix(in, ".qcow2") + ".raw"
	outFile, err := os.Create(updatedCacheFile)
	if err != nil {
		return "", errors.Errorf("creating file: %w", err)
	}
	defer outFile.Close()
	qcow2File, err := os.Open(in)
	if err != nil {
		return "", errors.Errorf("opening file: %w", err)
	}
	defer qcow2File.Close()

	err = ConvertQcow2ToRaw(ctx, qcow2File, outFile)
	if err != nil {
		return "", errors.Errorf("converting qcow2 to raw: %w", err)
	}

	return updatedCacheFile, nil
}
