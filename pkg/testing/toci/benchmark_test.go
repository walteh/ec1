package toci

import (
	"context"
	"testing"

	"github.com/walteh/ec1/pkg/units"
)

// BenchmarkDownloadImage benchmarks the DownloadImage performance
func BenchmarkDownloadImage(b *testing.B) {
	ctx := context.Background()
	cache := TestSimpleCache(b)
	imageRef := "docker.io/library/alpine:latest"
	platform := units.PlatformLinuxAMD64

	// First call to initialize the global cache
	_, err := cache.LoadImage(ctx, imageRef, platform)
	if err != nil {
		b.Fatalf("Failed to load image: %v", err)
	}

	// Reset timer to exclude initialization time
	b.ResetTimer()

	// Benchmark subsequent calls (should be very fast)
	for i := 0; i < b.N; i++ {
		_, err := cache.LoadImage(ctx, imageRef, platform)
		if err != nil {
			b.Fatalf("Failed to load image: %v", err)
		}
	}
}

// BenchmarkMultipleImages benchmarks loading different images
func BenchmarkMultipleImages(b *testing.B) {
	ctx := context.Background()
	cache := TestSimpleCache(b)
	platform := units.PlatformLinuxAMD64

	images := []string{
		"docker.io/library/alpine:latest",
		"docker.io/alpine/socat:latest",
		"docker.io/library/busybox:glibc",
	}

	// First call to initialize the global cache
	for _, imageRef := range images {
		_, err := cache.LoadImage(ctx, imageRef, platform)
		if err != nil {
			b.Fatalf("Failed to load image %s: %v", imageRef, err)
		}
	}

	// Reset timer to exclude initialization time
	b.ResetTimer()

	// Benchmark subsequent calls (should be very fast)
	for i := 0; i < b.N; i++ {
		imageRef := images[i%len(images)]
		_, err := cache.LoadImage(ctx, imageRef, platform)
		if err != nil {
			b.Fatalf("Failed to load image %s: %v", imageRef, err)
		}
	}
} 