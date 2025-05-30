package toci

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/mholt/archives"
	"github.com/stretchr/testify/require"
	"gitlab.com/tozd/go/errors"

	"github.com/walteh/ec1/pkg/oci"
	"github.com/walteh/ec1/pkg/units"

	oci_image_cache "github.com/walteh/ec1/gen/oci-image-cache"
)

// TestSimpleCache creates a cache with test implementations
// All images are extracted upfront during this call, not lazily
func TestSimpleCache(t testing.TB) *oci.SimpleCache {
	ctx := context.Background()
	downloader := NewTestImageDownloader(t)

	// Eagerly initialize the cache - extract all images now
	if err := downloader.InitializeCache(ctx); err != nil {
		t.Fatalf("Failed to initialize test image cache: %v", err)
	}

	return oci.NewSimpleCache(filepath.Join(t.TempDir(), "ec1-oci-test-cache"), downloader, &oci.OCIImageExtractor{})
}

// TestSimpleCacheWithPreload creates a cache and pre-loads specific images with their platforms
// This does both OCI layout extraction AND the expensive rootfs/ext4 extraction upfront
func TestSimpleCacheWithPreload(t testing.TB, imagesToPreload []ImagePlatformPair) *oci.SimpleCache {
	ctx := context.Background()
	downloader := NewTestImageDownloader(t)

	// Eagerly initialize the cache - extract all images now
	if err := downloader.InitializeCache(ctx); err != nil {
		t.Fatalf("Failed to initialize test image cache: %v", err)
	}

	cache := oci.NewSimpleCache(filepath.Join(t.TempDir(), "ec1-oci-test-cache"), downloader, &oci.OCIImageExtractor{})

	// Pre-load specific images with their platforms (expensive operations)
	for _, img := range imagesToPreload {
		slog.InfoContext(ctx, "pre-loading image extraction for tests", "image", img.ImageRef, "platform", img.Platform)
		_, err := cache.LoadImage(ctx, img.ImageRef, img.Platform)
		require.NoError(t, err, "failed to pre-load image %s for platform %s", img.ImageRef, img.Platform)
	}

	slog.InfoContext(ctx, "all test images pre-loaded, tests should be fast now")
	return cache
}

// TestImageDownloaderInitialized creates a standalone initialized TestImageDownloader for tests
// that need to use the downloader directly (e.g., for containerd integration scenarios)
func TestImageDownloaderInitialized(t testing.TB) *TestImageDownloader {
	ctx := context.Background()
	downloader := NewTestImageDownloader(t)

	// Eagerly initialize the cache - extract all images now
	if err := downloader.InitializeCache(ctx); err != nil {
		t.Fatalf("Failed to initialize test image downloader: %v", err)
	}

	return downloader
}

// TestImageDownloader implements ImageDownloader using embedded test data
// Each instance manages its own cache for the lifetime of a test
type TestImageDownloader struct {
	t           testing.TB
	cacheDir    string
	imageCache  map[string]string // imageRef -> ociLayoutPath
	mutex       sync.RWMutex
	initialized bool
}

// NewTestImageDownloader creates a new test downloader with per-test cache
func NewTestImageDownloader(t testing.TB) *TestImageDownloader {
	downloader := &TestImageDownloader{
		t:          t,
		imageCache: make(map[string]string),
	}

	// Setup cleanup when test ends
	t.Cleanup(func() {
		downloader.cleanup()
	})

	return downloader
}

// InitializeCache extracts all embedded images to the test's cache directory
// This is called eagerly from TestSimpleCache, not lazily
func (d *TestImageDownloader) InitializeCache(ctx context.Context) error {
	// Check if already initialized
	d.mutex.Lock()
	if d.initialized {
		d.mutex.Unlock()
		return nil
	}
	d.mutex.Unlock()

	slog.InfoContext(ctx, "initializing test image cache")

	// Create cache directory for this test
	var err error
	d.cacheDir, err = os.MkdirTemp("", "ec1-test-oci-cache-*")
	if err != nil {
		return errors.Errorf("creating cache directory: %w", err)
	}

	slog.InfoContext(ctx, "created test cache directory", "path", d.cacheDir)

	// Extract all images from the registry
	for imageRef, data := range oci_image_cache.Registry {
		imageRefStr := string(imageRef)
		slog.InfoContext(ctx, "extracting image to cache", "image", imageRefStr)

		// Create directory for this image
		imageCacheDir := filepath.Join(d.cacheDir, sanitizeImageRef(imageRefStr))
		if err := os.MkdirAll(imageCacheDir, 0755); err != nil {
			return errors.Errorf("creating image cache directory for %s: %w", imageRefStr, err)
		}

		// Extract the image
		if err := extractData(ctx, data, imageCacheDir); err != nil {
			return errors.Errorf("extracting image %s: %w", imageRefStr, err)
		}

		// Store in cache map (with mutex protection)
		d.mutex.Lock()
		d.imageCache[imageRefStr] = imageCacheDir
		d.mutex.Unlock()
		slog.InfoContext(ctx, "cached image", "image", imageRefStr, "path", imageCacheDir)
	}

	// Mark as initialized
	d.mutex.Lock()
	d.initialized = true
	d.mutex.Unlock()

	slog.InfoContext(ctx, "test image cache initialized", "images", len(d.imageCache))
	return nil
}

// DownloadImage "downloads" an image by returning the path from the cache (very fast - cache is pre-initialized)
func (d *TestImageDownloader) DownloadImage(ctx context.Context, imageRef string) (string, error) {
	// Check if cache is initialized (with proper mutex protection)
	d.mutex.RLock()
	isInitialized := d.initialized
	d.mutex.RUnlock()

	if !isInitialized {
		return "", errors.Errorf("cache not initialized - call InitializeCache first")
	}

	// Look up in cache
	d.mutex.RLock()
	cachedPath, exists := d.imageCache[imageRef]
	d.mutex.RUnlock()

	if !exists {
		return "", errors.Errorf("test image not found in cache: %s", imageRef)
	}

	slog.DebugContext(ctx, "serving image from cache", "image", imageRef, "path", cachedPath)
	return cachedPath, nil
}

// cleanup removes the cache directory for this test
func (d *TestImageDownloader) cleanup() {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if d.cacheDir != "" {
		slog.Info("cleaning up test image cache", "path", d.cacheDir)
		if err := os.RemoveAll(d.cacheDir); err != nil {
			slog.Warn("failed to clean up cache", "path", d.cacheDir, "error", err)
		}
		d.cacheDir = ""
		d.imageCache = make(map[string]string)
	}
}

// sanitizeImageRef converts an image reference to a safe directory name
func sanitizeImageRef(imageRef string) string {
	// Replace problematic characters with underscores
	sanitized := strings.ReplaceAll(imageRef, "/", "_")
	sanitized = strings.ReplaceAll(sanitized, ":", "_")
	sanitized = strings.ReplaceAll(sanitized, ".", "_")
	sanitized = strings.ReplaceAll(sanitized, "-", "_")
	return sanitized
}

func extractData(ctx context.Context, data []byte, destDir string) error {
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
		file, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode())
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

// ImagePlatformPair represents an image reference and its platform for pre-loading
type ImagePlatformPair struct {
	ImageRef string
	Platform units.Platform
}
