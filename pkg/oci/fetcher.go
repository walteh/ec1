package oci

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/mholt/archives"
	"gitlab.com/tozd/go/errors"

	"github.com/walteh/ec1/pkg/units"
)

var (
	_ ImageFetcher = &SkopeoImageFetcher{}
	_ ImageFetcher = &MemoryMapFetcher{}
	_ ImageFetcher = &CachedFetcher{}
)

// SkopeoImageFetcher implements ImageFetcher using skopeo for production use
type SkopeoImageFetcher struct {
	TempDir string // Optional: custom temp directory for fetches
}

// FetchImage fetches an image using skopeo and returns the OCI layout path
func (f *SkopeoImageFetcher) FetchImageToOCILayout(ctx context.Context, imageRef string) (string, error) {
	slog.InfoContext(ctx, "fetching image with skopeo", "image", imageRef)

	// Create temp directory for this fetch
	tempDir := f.TempDir
	if tempDir == "" {
		tempDir = os.TempDir()
	}

	ociLayoutPath := filepath.Join(tempDir, ociLayoutDirFromImageRef(imageRef))

	// Use skopeo to copy the image to OCI layout format
	cmd := exec.CommandContext(ctx, "skopeo", "copy",
		"docker://"+imageRef,
		"oci:"+ociLayoutPath)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		// Clean up on failure
		os.RemoveAll(ociLayoutPath)
		return "", errors.Errorf("skopeo copy failed: %w", err)
	}

	slog.InfoContext(ctx, "successfully fetched image", "image", imageRef, "oci_layout", ociLayoutPath)

	return ociLayoutPath, nil
}

// NewSkopeoImageFetcher creates a new skopeo-based image fetcher
func NewSkopeoImageFetcher() *SkopeoImageFetcher {
	return &SkopeoImageFetcher{}
}

// NewSkopeoImageFetcherWithTempDir creates a new skopeo-based image fetcher with custom temp directory
func NewSkopeoImageFetcherWithTempDir(tempDir string) *SkopeoImageFetcher {
	return &SkopeoImageFetcher{
		TempDir: tempDir,
	}
}

type MemoryMapFetcher struct {
	cacheDir        string
	compressedInput map[string][]byte
	mutex           sync.Mutex
}

func NewMemoryMapFetcher[T ~string](cacheDir string, compressedInput map[T][]byte) *MemoryMapFetcher {
	compressedInputMap := make(map[string][]byte)
	for k, v := range compressedInput {
		compressedInputMap[string(k)] = v
	}

	return &MemoryMapFetcher{
		cacheDir:        cacheDir,
		compressedInput: compressedInputMap,
	}
}

func (fch *MemoryMapFetcher) FetchImageToOCILayout(ctx context.Context, imageRef string) (string, error) {
	fch.mutex.Lock()
	input, ok := fch.compressedInput[imageRef]
	fch.mutex.Unlock()

	if !ok {
		return "", errors.Errorf("image not found in cache: %s", imageRef)
	}

	destDir := filepath.Join(fch.cacheDir, ociLayoutDirFromImageRef(imageRef))

	err := os.MkdirAll(destDir, 0755)
	if err != nil {
		return "", errors.Errorf("creating temp directory: %w", err)
	}

	err = extractCompressedOCI(ctx, input, destDir)
	if err != nil {
		return "", errors.Errorf("extracting compressed OCI layout to filesystem: %w", err)
	}

	return destDir, nil
}

type CachedFetcher struct {
	cacheDir    string
	resultCache map[string]string
	realFetcher ImageFetcher
}

func NewCachedFetcher(cacheDir string, realFetcher ImageFetcher) *CachedFetcher {

	fch := &CachedFetcher{
		cacheDir:    cacheDir,
		resultCache: make(map[string]string),
		realFetcher: realFetcher,
	}

	// errs := make(chan error, len(mapper))

	// for key, value := range mapper {
	// 	go func() {

	// 		tmpDir, err := os.MkdirTemp(os.TempDir(), "oci-layout-*")
	// 		if err != nil {
	// 			errs <- errors.Errorf("creating temp directory: %w", err)
	// 			return
	// 		}

	// 		fch.mutex.Lock()
	// 		fch.rootfsCache[string(key)] = tmpDir
	// 		fch.compressedInput[string(key)] = value
	// 		fch.mutex.Unlock()

	// 		errs <- nil
	// 	}()
	// }

	// grp := sync.WaitGroup{}
	// grp.Add(len(mapper))
	// go func() {
	// 	grp.Wait()
	// 	close(errs)
	// }()

	// for err := range errs {
	// 	grp.Done()
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// }

	return fch
}

func (fch *CachedFetcher) FetchImageToOCILayout(ctx context.Context, imageRef string) (string, error) {
	if _, ok := fch.resultCache[imageRef]; ok {
		return fch.resultCache[imageRef], nil
	}

	// check if the image is already cached
	cachePath := filepath.Join(fch.cacheDir, ociLayoutDirFromImageRef(imageRef))
	if _, err := os.Stat(cachePath); err == nil {
		return cachePath, nil
	}

	tempDir, err := fch.realFetcher.FetchImageToOCILayout(ctx, imageRef)
	if err != nil {
		return "", errors.Errorf("fetching image to OCI layout: %w", err)
	}

	fch.resultCache[imageRef] = tempDir

	return tempDir, nil
}

func extractCompressedOCI(ctx context.Context, data []byte, destDir string) error {
	// Create a reader from the embedded data
	reader := bytes.NewReader(data)

	// Use mholt's archives to identify and extract the archive format dynamically
	format, input, err := archives.Identify(ctx, "", reader)
	if err != nil {
		return errors.Errorf("identifying archive format: %w", err)
	}

	slog.DebugContext(ctx, "identified archive format", "format", format.Extension())

	// Check if the format supports extraction
	extractor, ok := format.(archives.Extractor)
	if !ok {
		return errors.Errorf("archive format %s does not support extraction", format.Extension())
	}

	// Extract the archive with path stripping
	err = extractor.Extract(ctx, input, func(ctx context.Context, f archives.FileInfo) error {
		// Strip the first directory component from the path
		// e.g., "docker_io_library_alpine_3_21/index.json" -> "index.json"
		pathParts := strings.Split(f.NameInArchive, "/")
		if len(pathParts) <= 1 {
			return nil // Skip the root directory entry
		}
		relativePath := strings.Join(pathParts[1:], "/")
		if relativePath == "" {
			return nil // Skip empty paths
		}

		targetPath := filepath.Join(destDir, relativePath)

		if f.IsDir() {
			// Create directory
			return os.MkdirAll(targetPath, f.Mode())
		}

		// Create parent directories
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return errors.Errorf("creating parent directory: %w", err)
		}

		// Extract file
		file, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY, f.Mode())
		if err != nil {
			return errors.Errorf("creating file %s: %w", targetPath, err)
		}
		defer file.Close()

		// Open the file from the archive
		rc, err := f.Open()
		if err != nil {
			return errors.Errorf("opening file from archive: %w", err)
		}
		defer rc.Close()

		// Copy the content
		if _, err := io.Copy(file, rc); err != nil {
			return errors.Errorf("copying file content: %w", err)
		}

		return nil
	})

	if err != nil {
		return errors.Errorf("extracting archive: %w", err)
	}

	return nil
}

func ociLayoutDirFromImageRef(imageRef string) string {
	ref := strings.ReplaceAll(imageRef, "/", "_")
	ref = strings.ReplaceAll(ref, ":", "_")
	return fmt.Sprintf("oci-layout-%s", ref)
}

func rootfsPathFromOCILayoutDirAndPlatform(ociLayoutPath string, platform units.Platform) string {
	return filepath.Join(filepath.Base(ociLayoutPath), platform.OS()+"_"+platform.Arch(), "rootfs")
}

func ext4PathFromOCILayoutDirAndPlatform(ociLayoutPath string, platform units.Platform) string {
	return filepath.Join(filepath.Base(ociLayoutPath), platform.OS()+"_"+platform.Arch(), "ext4")
}

func metadataPathFromOCILayoutDirAndPlatform(ociLayoutPath string, platform units.Platform) string {
	return filepath.Join(filepath.Base(ociLayoutPath), platform.OS()+"_"+platform.Arch(), "metadata.json")
}
