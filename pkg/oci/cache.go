package oci

// imageCache implements ImageCache interface
type imageCache struct {
	cacheDir     string
	ImageFetcher // Optional - only used for standalone mode
	FilesystemConverter
}

// NewImageCacheWithFS creates a new cache instance with custom filesystem provider
func NewImageCache(cacheDir string, fetcher ImageFetcher, converter FilesystemConverter) *imageCache {
	return &imageCache{
		cacheDir:            cacheDir,
		ImageFetcher:        NewCachedFetcher(cacheDir, fetcher),
		FilesystemConverter: NewCachedConverter(converter),
	}
}
