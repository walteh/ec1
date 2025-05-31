package toci

import (
	"context"
	"path/filepath"
	"sync"
	"testing"

	"github.com/walteh/ec1/pkg/oci"
	"github.com/walteh/ec1/pkg/units"

	oci_image_cache "github.com/walteh/ec1/gen/oci-image-cache"
)

// TestImageCache creates a cache with test implementations
// All images are extracted upfront during this call, not lazily
func PreloadedImageCache(t testing.TB, platform units.Platform, imagesToPreload []oci_image_cache.OCICachedImage) oci.ImageCache {
	ctx := context.Background()
	cacheDir := filepath.Join(t.TempDir(), "ec1-oci-test-cache")
	memMapFetcher := oci.NewMemoryMapFetcher(cacheDir, oci_image_cache.Registry)
	fetcher := oci.NewImageCache(cacheDir, memMapFetcher, oci.NewOCIFilesystemConverter())
	errs := make(chan error)
	for _, img := range imagesToPreload {
		go func() {
			_, err := oci.FetchAndConvertImage(ctx, fetcher, string(img), platform)
			errs <- err
		}()
	}
	wait := sync.WaitGroup{}
	wait.Add(len(imagesToPreload))
	go func() {
		wait.Wait()
		close(errs)
	}()

	for err := range errs {
		wait.Done()
		if err != nil {
			t.Fatalf("Failed to preload image: %v", err)
		}
	}

	return fetcher
}

// // TestImageCacheWithPreload creates a cache and pre-loads specific images with their platforms
// // This does both OCI layout extraction AND the expensive rootfs/ext4 extraction upfront
// func TestImageCacheWithPreload(t testing.TB, imagesToPreload []ImagePlatformPair) oci.ImageCache {
// 	ctx := context.Background()
// 	fetcher := NewTestImageFetcher(t)

// 	// Eagerly initialize the cache - extract all images now
// 	if err := fetcher.InitializeCache(ctx); err != nil {
// 		t.Fatalf("Failed to initialize test image cache: %v", err)
// 	}

// 	cache := oci.NewImageCache(filepath.Join(t.TempDir(), "ec1-oci-test-cache"), fetcher, nil, nil)

// 	// Pre-load specific images with their platforms (expensive operations)
// 	for _, img := range imagesToPreload {
// 		slog.InfoContext(ctx, "pre-loading image extraction for tests", "image", img.ImageRef, "platform", img.Platform)
// 		_, err := cache.LoadImage(ctx, img.ImageRef, img.Platform)
// 		require.NoError(t, err, "failed to pre-load image %s for platform %s", img.ImageRef, img.Platform)
// 	}

// 	slog.InfoContext(ctx, "all test images pre-loaded, tests should be fast now")
// 	return cache
// }

// // TestImageFetcherInitialized creates a standalone initialized TestImageFetcher for tests
// // that need to use the fetcher directly (e.g., for containerd integration scenarios)
// func TestImageFetcherInitialized(t testing.TB) *TestImageFetcher {
// 	ctx := context.Background()
// 	fetcher := NewTestImageFetcher(t)

// 	// Eagerly initialize the cache - extract all images now
// 	if err := fetcher.InitializeCache(ctx); err != nil {
// 		t.Fatalf("Failed to initialize test image fetcher: %v", err)
// 	}

// 	return fetcher
// }

// // TestImageFetcher implements ImageFetcher using embedded test data
// // Each instance manages its own cache for the lifetime of a test
// type TestImageFetcher struct {
// 	t           testing.TB
// 	cacheDir    string
// 	imageCache  map[string]string // imageRef -> ociLayoutPath
// 	mutex       sync.RWMutex
// 	initialized bool
// }

// // NewTestImageFetcher creates a new test fetcher with per-test cache
// func NewTestImageFetcher(t testing.TB) *TestImageFetcher {
// 	fetcher := &TestImageFetcher{
// 		t:          t,
// 		imageCache: make(map[string]string),
// 	}

// 	// Setup cleanup when test ends
// 	t.Cleanup(func() {
// 		fetcher.cleanup()
// 	})

// 	return fetcher
// }

// // InitializeCache extracts all embedded images to the test's cache directory
// // This is called eagerly from TestImageCache, not lazily
// func (f *TestImageFetcher) InitializeCache(ctx context.Context) error {
// 	// Check if already initialized
// 	f.mutex.Lock()
// 	if f.initialized {
// 		f.mutex.Unlock()
// 		return nil
// 	}
// 	f.mutex.Unlock()

// 	// slog.InfoContext(ctx, "initializing test image cache")

// 	// Create cache directory for this test
// 	var err error
// 	f.cacheDir, err = os.MkdirTemp("", "ec1-test-oci-cache-*")
// 	if err != nil {
// 		return errors.Errorf("creating cache directory: %w", err)
// 	}

// 	slog.InfoContext(ctx, "created test cache directory", "path", f.cacheDir)

// 	// Extract all images from the registry
// 	for imageRef, data := range oci_image_cache.Registry {
// 		imageRefStr := string(imageRef)
// 		// slog.InfoContext(ctx, "extracting image to cache", "image", imageRefStr)

// 		// Create directory for this image
// 		imageCacheDir := filepath.Join(f.cacheDir, sanitizeImageRef(imageRefStr))
// 		if err := os.MkdirAll(imageCacheDir, 0755); err != nil {
// 			return errors.Errorf("creating image cache directory for %s: %w", imageRefStr, err)
// 		}

// 		// Extract the image
// 		if err := extractData(ctx, data, imageCacheDir); err != nil {
// 			return errors.Errorf("extracting image %s: %w", imageRefStr, err)
// 		}

// 		// Store in cache map (with mutex protection)
// 		f.mutex.Lock()
// 		f.imageCache[imageRefStr] = imageCacheDir
// 		f.mutex.Unlock()
// 		// slog.InfoContext(ctx, "cached image", "image", imageRefStr, "path", imageCacheDir)
// 	}

// 	// Mark as initialized
// 	f.mutex.Lock()
// 	f.initialized = true
// 	f.mutex.Unlock()

// 	slog.InfoContext(ctx, "test image cache initialized", "images", len(f.imageCache))
// 	return nil
// }

// // FetchImage "fetches" an image by returning the path from the cache (very fast - cache is pre-initialized)
// func (f *TestImageFetcher) FetchImage(ctx context.Context, imageRef string) (string, error) {
// 	// Check if cache is initialized (with proper mutex protection)
// 	f.mutex.RLock()
// 	isInitialized := f.initialized
// 	f.mutex.RUnlock()

// 	if !isInitialized {
// 		return "", errors.Errorf("cache not initialized - call InitializeCache first")
// 	}

// 	// Look up in cache
// 	f.mutex.RLock()
// 	cachedPath, exists := f.imageCache[imageRef]
// 	f.mutex.RUnlock()

// 	if !exists {
// 		return "", errors.Errorf("test image not found in cache: %s", imageRef)
// 	}

// 	slog.DebugContext(ctx, "serving image from cache", "image", imageRef, "path", cachedPath)
// 	return cachedPath, nil
// }

// // cleanup removes the cache directory for this test
// func (f *TestImageFetcher) cleanup() {
// 	f.mutex.Lock()
// 	defer f.mutex.Unlock()

// 	if f.cacheDir != "" {
// 		slog.Info("cleaning up test image cache", "path", f.cacheDir)
// 		if err := os.RemoveAll(f.cacheDir); err != nil {
// 			slog.Warn("failed to clean up cache", "path", f.cacheDir, "error", err)
// 		}
// 		f.cacheDir = ""
// 		f.imageCache = make(map[string]string)
// 	}
// }

// // sanitizeImageRef converts an image reference to a safe directory name
// func sanitizeImageRef(imageRef string) string {
// 	// Replace problematic characters with underscores
// 	sanitized := strings.ReplaceAll(imageRef, "/", "_")
// 	sanitized = strings.ReplaceAll(sanitized, ":", "_")
// 	sanitized = strings.ReplaceAll(sanitized, ".", "_")
// 	sanitized = strings.ReplaceAll(sanitized, "-", "_")
// 	return sanitized
// }

// func extractData(ctx context.Context, data []byte, destDir string) error {
// 	// Create a reader from the embedded data
// 	reader := bytes.NewReader(data)

// 	// Use mholt's archives to identify and extract the archive format dynamically
// 	format, input, err := archives.Identify(ctx, "", reader)
// 	if err != nil {
// 		return errors.Errorf("identifying archive format: %w", err)
// 	}

// 	slog.DebugContext(ctx, "identified archive format", "format", format.Extension())

// 	// Check if the format supports extraction
// 	extractor, ok := format.(archives.Extractor)
// 	if !ok {
// 		return errors.Errorf("archive format %s does not support extraction", format.Extension())
// 	}

// 	// Extract the archive with path stripping
// 	err = extractor.Extract(ctx, input, func(ctx context.Context, f archives.FileInfo) error {
// 		// Strip the first directory component from the path
// 		// e.g., "docker_io_library_alpine_3_21/index.json" -> "index.json"
// 		pathParts := strings.Split(f.NameInArchive, "/")
// 		if len(pathParts) <= 1 {
// 			return nil // Skip the root directory entry
// 		}
// 		relativePath := strings.Join(pathParts[1:], "/")
// 		if relativePath == "" {
// 			return nil // Skip empty paths
// 		}

// 		targetPath := filepath.Join(destDir, relativePath)

// 		if f.IsDir() {
// 			// Create directory
// 			return os.MkdirAll(targetPath, f.Mode())
// 		}

// 		// Create parent directories
// 		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
// 			return errors.Errorf("creating parent directory: %w", err)
// 		}

// 		// Extract file
// 		file, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode())
// 		if err != nil {
// 			return errors.Errorf("creating file %s: %w", targetPath, err)
// 		}
// 		defer file.Close()

// 		// Open the file from the archive
// 		rc, err := f.Open()
// 		if err != nil {
// 			return errors.Errorf("opening file from archive: %w", err)
// 		}
// 		defer rc.Close()

// 		// Copy the content
// 		if _, err := io.Copy(file, rc); err != nil {
// 			return errors.Errorf("copying file content: %w", err)
// 		}

// 		return nil
// 	})

// 	if err != nil {
// 		return errors.Errorf("extracting archive: %w", err)
// 	}

// 	return nil
// }

// // ImagePlatformPair represents an image reference and its platform for pre-loading
// type ImagePlatformPair struct {
// 	ImageRef string
// 	Platform units.Platform
// }
