package toci

import (
	"path/filepath"
	"sync"
	"testing"

	"github.com/walteh/ec1/pkg/oci"
	"github.com/walteh/ec1/pkg/units"
)

// TestImageCache creates a cache with test implementations
// All images are extracted upfront during this call, not lazily
func PreloadedImageCache(t testing.TB, platform units.Platform, imagesToPreload []string) oci.ImageCache {

	cacheDir := filepath.Join(t.TempDir(), "ec1-oci-test-cache")
	memMapFetcher := oci.NewMemoryMapFetcher(cacheDir, Registry())
	fetcher := oci.NewImageCache(cacheDir, memMapFetcher, oci.NewOCIFilesystemConverter())
	errs := make(chan error)
	for _, img := range imagesToPreload {
		go func() {
			_, err := oci.FetchAndConvertImage(t.Context(), fetcher, string(img), platform)
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
