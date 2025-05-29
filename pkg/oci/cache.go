package oci

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Microsoft/hcsshim/ext4/tar2ext4"
	"github.com/containers/image/v5/types"
	"github.com/mholt/archives"
	"gitlab.com/tozd/go/errors"

	v1 "github.com/opencontainers/image-spec/specs-go/v1"
)

// CacheEntry represents a cached container image with metadata
type CacheEntry struct {
	ImageRef       string    `json:"image_ref"`
	Platform       string    `json:"platform"`
	CachedAt       time.Time `json:"cached_at"`
	ExpiresAt      time.Time `json:"expires_at"`
	RootfsPath     string    `json:"rootfs_path"`
	OciLayoutPath  string    `json:"oci_layout_path"`
	MetadataPath   string    `json:"metadata_path"`
	Size           int64     `json:"size"`
	RootfsDiskPath string    `json:"rootfs_disk_path"`
}

const (
	// CacheExpiration is how long cached images are valid
	CacheExpiration = 24 * time.Hour

	// CacheMetadataFile is the name of the cache metadata file
	CacheMetadataFile = "cache-metadata.json"
)

// getCacheDir returns the cache directory for OCI images
func getCacheDir() (string, error) {
	userCacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", errors.Errorf("getting user cache dir: %w", err)
	}
	return filepath.Join(userCacheDir, "ec1", "cache", "oci"), nil
}

// getCacheDirForImage returns a cache directory specific to an image and platform
func getCacheDirForImage(imageRef string, platform *types.SystemContext) (string, error) {
	// Create a unique identifier for this image+platform combination
	platformStr := "linux-amd64" // default
	if platform != nil {
		platformStr = fmt.Sprintf("%s-%s", platform.OSChoice, platform.ArchitectureChoice)
	}

	// Hash the image reference and platform to create a unique directory name
	hasher := sha256.New()
	hasher.Write([]byte(imageRef + ":" + platformStr))
	hash := hex.EncodeToString(hasher.Sum(nil))

	// Create a readable directory name
	imageName := strings.ReplaceAll(imageRef, "/", "_")
	imageName = strings.ReplaceAll(imageName, ":", "_")
	if len(imageName) > 50 {
		imageName = imageName[:50]
	}

	dirname := fmt.Sprintf("%s_%s_%s", imageName, platformStr, hash[:16])

	cacheDir, err := getCacheDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(cacheDir, dirname), nil
}

// loadCacheEntry loads cache metadata for an image
func loadCacheEntry(cacheDir string) (*CacheEntry, error) {
	metadataPath := filepath.Join(cacheDir, CacheMetadataFile)

	data, err := os.ReadFile(metadataPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No cache entry exists
		}
		return nil, errors.Errorf("reading cache metadata: %w", err)
	}

	var entry CacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, errors.Errorf("unmarshaling cache metadata: %w", err)
	}

	return &entry, nil
}

// saveCacheEntry saves cache metadata for an image
func saveCacheEntry(cacheDir string, entry *CacheEntry) error {
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return errors.Errorf("creating cache directory: %w", err)
	}

	metadataPath := filepath.Join(cacheDir, CacheMetadataFile)

	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return errors.Errorf("marshaling cache metadata: %w", err)
	}

	if err := os.WriteFile(metadataPath, data, 0644); err != nil {
		return errors.Errorf("writing cache metadata: %w", err)
	}

	return nil
}

// isCacheValid checks if a cache entry is still valid
func isCacheValid(entry *CacheEntry) bool {
	if entry == nil {
		return false
	}

	// Check if cache has expired
	if time.Now().After(entry.ExpiresAt) {
		return false
	}

	if entry.RootfsDiskPath == "" {
		return false
	}

	if _, err := os.Stat(entry.RootfsDiskPath); err != nil {
		return false
	}

	// Check if cached files still exist
	if _, err := os.Stat(entry.RootfsPath); err != nil {
		return false
	}

	if _, err := os.Stat(entry.MetadataPath); err != nil {
		return false
	}

	return true
}

// getCachedRootfsSize calculates the size of a cached rootfs directory
func getCachedRootfsSize(rootfsPath string) (int64, error) {
	var size int64

	err := filepath.Walk(rootfsPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})

	return size, err
}

type CachedContainer struct {
	ImageRef         string
	Platform         *types.SystemContext
	ReadonlyFSPath   string
	ReadonlyExt4Path string
}

// ContainerToVirtioDeviceCached is a cached version of ContainerToVirtioDevice
// It caches extracted container images for 24 hours and handles offline scenarios gracefully
func ContainerToVirtioDeviceCached(ctx context.Context, opts ContainerToVirtioOptions) (*CachedContainer, *v1.Image, error) {
	slog.InfoContext(ctx, "converting OCI container to virtio device with caching",
		"image", opts.ImageRef,
		"cache_expiration", CacheExpiration)

	// Get cache directory for this image
	cacheDir, err := getCacheDirForImage(opts.ImageRef, opts.Platform)
	if err != nil {
		slog.WarnContext(ctx, "failed to get cache directory, falling back to non-cached", "error", err)
		return nil, nil, errors.Errorf("failed to get cache directory: %w", err)
	}

	// Load existing cache entry
	cacheEntry, err := loadCacheEntry(cacheDir)
	if err != nil {
		slog.WarnContext(ctx, "failed to load cache entry, will try to create new one", "error", err)
	}

	// Check if cache is valid
	if isCacheValid(cacheEntry) {
		slog.InfoContext(ctx, "using cached container image",
			"image", opts.ImageRef,
			"cached_at", cacheEntry.CachedAt,
			"expires_at", cacheEntry.ExpiresAt,
			"rootfs_disk_path", cacheEntry.RootfsDiskPath,
			"size_mb", cacheEntry.Size/1024/1024)

		return loadFromCache(ctx, cacheEntry, opts)
	}

	// Cache is invalid or doesn't exist, try to download and cache
	slog.InfoContext(ctx, "cache miss or expired, downloading container image",
		"image", opts.ImageRef,
		"cache_exists", cacheEntry != nil,
		"cache_valid", isCacheValid(cacheEntry))

	// Try to download and extract the image
	device, metadata, err := downloadAndCache(ctx, opts, cacheDir)
	if err != nil {
		// If we have an expired cache entry and download failed (e.g., offline),
		// try to use the expired cache as a fallback
		if cacheEntry != nil {
			slog.WarnContext(ctx, "download failed but found expired cache, using as fallback",
				"error", err,
				"cached_at", cacheEntry.CachedAt,
				"expired_since", time.Since(cacheEntry.ExpiresAt))

			fallbackDevice, fallbackMetadata, fallbackErr := loadFromCache(ctx, cacheEntry, opts)
			if fallbackErr == nil {
				slog.InfoContext(ctx, "successfully loaded from expired cache as fallback")
				return fallbackDevice, fallbackMetadata, nil
			}

			slog.ErrorContext(ctx, "failed to load from expired cache fallback", "fallback_error", fallbackErr)
		}

		return nil, nil, errors.Errorf("failed to download image and no valid cache available: %w", err)
	}

	return device, metadata, nil
}

// downloadAndCache downloads an image and caches it
func downloadAndCache(ctx context.Context, opts ContainerToVirtioOptions, cacheDir string) (*CachedContainer, *v1.Image, error) {
	// Ensure cache directory exists
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, nil, errors.Errorf("creating cache directory: %w", err)
	}

	// Create a temporary directory for the download
	tempDir, err := os.MkdirTemp(cacheDir, "download-*")
	if err != nil {
		return nil, nil, errors.Errorf("creating temp directory: %w", err)
	}

	// Clean up temp directory on error
	defer func() {
		if err != nil {
			os.RemoveAll(tempDir)
		}
	}()

	metadata, err := pullAndExtractImageSkopeo(ctx, opts.ImageRef, tempDir, opts.Platform)
	if err != nil {
		return nil, nil, errors.Errorf("pulling and extracting image: %w", err)
	}

	// Move the extracted content to the cache
	cachedRootfsPath := filepath.Join(cacheDir, "fs")
	cachedOciLayoutPath := filepath.Join(cacheDir, "oci-layout")
	cachedMetadataPath := filepath.Join(cacheDir, "metadata.json")

	// Remove old cache if it exists

	os.RemoveAll(cachedRootfsPath)
	os.Remove(cachedMetadataPath)

	// Find the extracted rootfs in the temp directory
	// Move rootfs to cache
	if err := os.Rename(filepath.Join(tempDir, "rootfs"), cachedRootfsPath); err != nil {
		return nil, nil, errors.Errorf("moving rootfs to cache: %w", err)
	}

	if err := os.Rename(filepath.Join(tempDir, "oci-layout"), cachedOciLayoutPath); err != nil {
		return nil, nil, errors.Errorf("moving rootfs to cache: %w", err)
	}

	// remove the temp dir
	os.RemoveAll(tempDir)

	// Save metadata to cache
	metadataData, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return nil, nil, errors.Errorf("marshaling metadata: %w", err)
	}

	if err := os.WriteFile(cachedMetadataPath, metadataData, 0644); err != nil {
		return nil, nil, errors.Errorf("writing metadata to cache: %w", err)
	}

	// Calculate cache size
	cacheSize, err := getCachedRootfsSize(cachedRootfsPath)
	if err != nil {
		slog.WarnContext(ctx, "failed to calculate cache size", "error", err)
		cacheSize = 0
	}

	rootfsDiskPath, err := bundleToSquashfs(ctx, cachedRootfsPath, cacheDir)
	if err != nil {
		return nil, nil, errors.Errorf("bundling rootfs to squashfs: %w", err)
	}

	// mark all these things as read only
	// os.Chmod(cacheDir, 0555)

	// Create cache entry
	now := time.Now()
	cacheEntry := &CacheEntry{
		ImageRef:       opts.ImageRef,
		Platform:       getPlatformString(opts.Platform),
		CachedAt:       now,
		ExpiresAt:      now.Add(CacheExpiration),
		RootfsPath:     cachedRootfsPath,
		OciLayoutPath:  cachedOciLayoutPath,
		MetadataPath:   cachedMetadataPath,
		RootfsDiskPath: rootfsDiskPath,
		Size:           cacheSize,
	}

	// Save cache entry metadata
	if err := saveCacheEntry(cacheDir, cacheEntry); err != nil {
		slog.WarnContext(ctx, "failed to save cache metadata", "error", err)
	}

	slog.InfoContext(ctx, "successfully cached container image",
		"image", opts.ImageRef,
		"cache_dir", cacheDir,
		"rootfs_disk_path", rootfsDiskPath,
		"size_mb", cacheSize/1024/1024,
		"expires_at", cacheEntry.ExpiresAt)

	// Clean up temp directory
	os.RemoveAll(tempDir)

	// Create new device pointing to cached rootfs
	return loadFromCache(ctx, cacheEntry, opts)
}

func bundleToSquashfs(ctx context.Context, rootfsPath string, cacheDir string) (string, error) {
	rootfsDiskPath := filepath.Join(cacheDir, "fs.ext4.img")

	fles, err := archives.FilesFromDisk(ctx, &archives.FromDiskOptions{
		// Root: rootfsPath,
	}, map[string]string{
		rootfsPath: rootfsDiskPath,
	})
	if err != nil {
		return "", errors.Errorf("getting files from disk: %w", err)
	}

	os.Remove(rootfsDiskPath)

	// pack the folder into a tar stream
	pr, pw := io.Pipe()
	go func() {
		defer pw.Close()
		err := (&archives.Tar{}).Archive(ctx, pw, fles)
		if err != nil {
			slog.ErrorContext(ctx, "failed to archive files", "error", err)
		}
	}()

	out, err := os.Create(rootfsDiskPath) // will be ext4 inside a raw file
	if err != nil {
		return "", errors.Errorf("creating rootfs disk file: %w", err)
	}
	err = tar2ext4.ConvertTarToExt4(pr, out, tar2ext4.MaximumDiskSize(1<<30), tar2ext4.InlineData) // 1 GiB
	if err != nil {
		return "", errors.Errorf("converting tar to ext4: %w", err)
	}

	// backend, err := file.CreateFromPath(rootfsDiskPath, 2048)
	// if err != nil {
	// 	return "", errors.Errorf("creating local backend: %w", err)
	// }

	// fs, err := squashfs.Create(backend, 0, 0, 0)
	// if err != nil {
	// 	return "", errors.Errorf("creating squashfs filesystem: %w", err)
	// }

	// just use mksquashfs to create the squashfs file
	// cmd := exec.Command("mksquashfs", rootfsPath, rootfsDiskPath)
	// cmd.Stdout = os.Stdout
	// cmd.Stderr = os.Stderr
	// if err := cmd.Run(); err != nil {
	// 	return "", errors.Errorf("creating squashfs file: %w", err)
	// }

	// errgrp, _ := errgroup.WithContext(ctx)

	// err = filepath.Walk(rootfsPath, func(path string, info os.FileInfo, err error) error {
	// 	if err != nil {
	// 		return errors.Errorf("walking source dir: %w", err)
	// 	}
	// 	// Compute the path inside the SquashFS
	// 	rel, _ := filepath.Rel(rootfsPath, path)
	// 	dest := "/" + rel

	// 	if info.IsDir() {
	// 		err := fs.Mkdir(dest) // create directory inside FS  [oai_citation:6‡Go Packages](https://pkg.go.dev/github.com/diskfs/go-diskfs/filesystem/squashfs)
	// 		if err != nil {
	// 			return errors.Errorf("creating directory: %w", err)
	// 		}
	// 		return nil
	// 	}
	// 	if info.Mode()&os.ModeSymlink != 0 {
	// 		// Handle symlink
	// 		target, err := os.Readlink(path)
	// 		if err != nil {
	// 			return errors.Errorf("reading symlink: %w", err)
	// 		}
	// 		err = fs.Symlink(target, dest) // create symlink inside FS  [oai_citation:7‡Go Packages](https://pkg.go.dev/github.com/diskfs/go-diskfs/filesystem/squashfs)
	// 		if err != nil {
	// 			return errors.Errorf("creating symlink: %w", err)
	// 		}
	// 		return nil
	// 	}
	// 	// Regular file: copy contents
	// 	srcFile, err := os.Open(path)
	// 	if err != nil {
	// 		return errors.Errorf("opening source file: %w", err)
	// 	}

	// 	outFile, err := fs.OpenFile(dest, os.O_CREATE|os.O_RDWR)
	// 	if err != nil {
	// 		return errors.Errorf("opening destination file: %w", err)
	// 	}

	// 	errgrp.Go(func() error {
	// 		defer outFile.Close()
	// 		defer srcFile.Close()
	// 		if _, err := io.Copy(outFile, srcFile); err != nil {
	// 			return errors.Errorf("copying file: %w", err)
	// 		}
	// 		return nil
	// 	})
	// 	return nil
	// })
	// if err != nil {
	// 	return "", errors.Errorf("walking source dir: %w", err)
	// }

	// if err := errgrp.Wait(); err != nil {
	// 	return "", errors.Errorf("walking source dir: %w", err)
	// }

	// optz := squashfs.FinalizeOptions{
	// 	Compression: &squashfs.CompressorXz{},
	// }
	// if err := fs.Finalize(optz); err != nil {
	// 	return "", errors.Errorf("finalizing squashfs: %w", err)
	// }

	// if err := fs.Close(); err != nil {
	// 	return "", errors.Errorf("closing squashfs filesystem: %w", err)
	// }

	return rootfsDiskPath, nil
}

// loadFromCache loads a virtio device from cached data
func loadFromCache(ctx context.Context, cacheEntry *CacheEntry, opts ContainerToVirtioOptions) (*CachedContainer, *v1.Image, error) {
	// Load metadata from cache
	metadataData, err := os.ReadFile(cacheEntry.MetadataPath)
	if err != nil {
		return nil, nil, errors.Errorf("reading cached metadata: %w", err)
	}

	var metadata v1.Image
	if err := json.Unmarshal(metadataData, &metadata); err != nil {
		return nil, nil, errors.Errorf("unmarshaling cached metadata: %w", err)
	}

	// Set up mount point if specified
	if opts.MountPoint != "" {
		// Create mount point directory
		if err := os.MkdirAll(opts.MountPoint, 0755); err != nil {
			return nil, nil, errors.Errorf("creating mount point: %w", err)
		}

		// Create mount manager for cached rootfs
		mountMng := NewContainerdMountManager(cacheEntry.RootfsPath, opts.MountPoint, opts.ReadOnly, opts.ImageRef)

		// Mount the cached filesystem
		if err := mountMng.Mount(ctx); err != nil {
			return nil, nil, errors.Errorf("mounting cached filesystem: %w", err)
		}

		return &CachedContainer{
			ImageRef:         opts.ImageRef,
			Platform:         opts.Platform,
			ReadonlyFSPath:   opts.MountPoint,
			ReadonlyExt4Path: cacheEntry.RootfsDiskPath,
		}, &metadata, nil
	}

	slog.InfoContext(ctx, "successfully loaded container from cache",
		"image", opts.ImageRef,
		"rootfs_disk_path", cacheEntry.RootfsDiskPath,
		"rootfs_path", cacheEntry.RootfsPath,
		"cached_at", cacheEntry.CachedAt)

	return &CachedContainer{
		ImageRef:         opts.ImageRef,
		Platform:         opts.Platform,
		ReadonlyFSPath:   cacheEntry.RootfsPath,
		ReadonlyExt4Path: cacheEntry.RootfsDiskPath,
	}, &metadata, nil
}

// getPlatformString converts a SystemContext to a string representation
func getPlatformString(platform *types.SystemContext) string {
	if platform == nil {
		return "linux-amd64"
	}

	os := platform.OSChoice
	if os == "" {
		os = "linux"
	}

	arch := platform.ArchitectureChoice
	if arch == "" {
		arch = "amd64"
	}

	return fmt.Sprintf("%s-%s", os, arch)
}

// ClearCache removes all cached OCI images
func ClearCache(ctx context.Context) error {
	cacheDir, err := getCacheDir()
	if err != nil {
		return errors.Errorf("getting cache directory: %w", err)
	}

	slog.InfoContext(ctx, "clearing OCI image cache", "cache_dir", cacheDir)

	if err := os.RemoveAll(cacheDir); err != nil {
		return errors.Errorf("removing cache directory: %w", err)
	}

	slog.InfoContext(ctx, "successfully cleared OCI image cache")
	return nil
}

// ListCachedImages returns information about all cached images
func ListCachedImages(ctx context.Context) ([]*CacheEntry, error) {
	cacheDir, err := getCacheDir()
	if err != nil {
		return nil, errors.Errorf("getting cache directory: %w", err)
	}

	entries, err := os.ReadDir(cacheDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*CacheEntry{}, nil
		}
		return nil, errors.Errorf("reading cache directory: %w", err)
	}

	var cachedImages []*CacheEntry

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		imageCacheDir := filepath.Join(cacheDir, entry.Name())
		cacheEntry, err := loadCacheEntry(imageCacheDir)
		if err != nil {
			slog.WarnContext(ctx, "failed to load cache entry", "dir", entry.Name(), "error", err)
			continue
		}

		if cacheEntry != nil {
			cachedImages = append(cachedImages, cacheEntry)
		}
	}

	return cachedImages, nil
}

// CleanExpiredCache removes expired cache entries
func CleanExpiredCache(ctx context.Context) error {
	cachedImages, err := ListCachedImages(ctx)
	if err != nil {
		return errors.Errorf("listing cached images: %w", err)
	}

	var removedCount int
	var reclaimedSize int64

	for _, entry := range cachedImages {
		if time.Now().After(entry.ExpiresAt) {
			slog.InfoContext(ctx, "removing expired cache entry",
				"image", entry.ImageRef,
				"expired_since", time.Since(entry.ExpiresAt),
				"size_mb", entry.Size/1024/1024)

			// Remove the entire cache directory for this image
			cacheDir := filepath.Dir(entry.RootfsPath)
			if err := os.RemoveAll(cacheDir); err != nil {
				slog.WarnContext(ctx, "failed to remove expired cache", "dir", cacheDir, "error", err)
			} else {
				removedCount++
				reclaimedSize += entry.Size
			}
		}
	}

	slog.InfoContext(ctx, "cleaned expired cache entries",
		"removed_count", removedCount,
		"reclaimed_mb", reclaimedSize/1024/1024)

	return nil
}
