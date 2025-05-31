package main

import (
	"context"
	"encoding/json"
	"flag"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"gitlab.com/tozd/go/errors"

	v1 "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/walteh/ec1/pkg/logging"
	"github.com/walteh/ec1/pkg/units"
)

// Docker manifest types for compatibility
const (
	DockerManifestListMediaType = "application/vnd.docker.distribution.manifest.list.v2+json"
	DockerManifestMediaType     = "application/vnd.docker.distribution.manifest.v2+json"
)

func main() {
	var (
		ociLayoutDir = flag.String("oci-layout", "", "Path to OCI layout directory")
		platformsStr = flag.String("platforms", "", "Comma-separated list of platforms to keep (e.g., linux/amd64,linux/arm64)")
		verbose      = flag.Bool("verbose", false, "Enable verbose logging")
		dryRun       = flag.Bool("dry-run", false, "Show what would be done without making changes")
	)
	flag.Parse()

	ctx := context.Background()
	if *verbose {
		// Use simple logging with more detail for verbose mode
		ctx = logging.SetupSlogSimple(ctx)
	} else {
		ctx = logging.SetupSlogSimple(ctx)
	}

	if *ociLayoutDir == "" {
		slog.ErrorContext(ctx, "oci-layout directory is required")
		flag.Usage()
		os.Exit(1)
	}

	if *platformsStr == "" {
		slog.ErrorContext(ctx, "platforms list is required")
		flag.Usage()
		os.Exit(1)
	}

	// Parse platforms
	platformStrs := strings.Split(*platformsStr, ",")
	var platforms []units.Platform
	for _, p := range platformStrs {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		platform, err := units.ParsePlatform(p)
		if err != nil {
			slog.ErrorContext(ctx, "invalid platform", "platform", p, "error", err)
			os.Exit(1)
		}
		platforms = append(platforms, platform)
	}

	slog.InfoContext(ctx, "starting manifest prune",
		"oci_layout", *ociLayoutDir,
		"platforms", platforms,
		"dry_run", *dryRun)

	err := pruneManifest(ctx, *ociLayoutDir, platforms, *dryRun)
	if err != nil {
		slog.ErrorContext(ctx, "failed to prune manifest", "error", err)
		os.Exit(1)
	}

	slog.InfoContext(ctx, "manifest prune completed successfully")
}

func pruneManifest(ctx context.Context, ociLayoutDir string, keepPlatforms []units.Platform, dryRun bool) error {
	// Read the index.json file
	indexPath := filepath.Join(ociLayoutDir, "index.json")
	indexData, err := os.ReadFile(indexPath)
	if err != nil {
		return errors.Errorf("reading index.json: %w", err)
	}

	var index v1.Index
	if err := json.Unmarshal(indexData, &index); err != nil {
		return errors.Errorf("unmarshaling index.json: %w", err)
	}

	slog.InfoContext(ctx, "loaded OCI index", "manifests", len(index.Manifests))

	// Track blobs that should be kept
	keepBlobs := make(map[string]bool)
	var newManifests []v1.Descriptor

	// Process each manifest in the index
	for _, manifestDesc := range index.Manifests {
		slog.InfoContext(ctx, "processing manifest", "digest", manifestDesc.Digest.String(), "media_type", manifestDesc.MediaType)

		// Read the manifest
		manifestPath := filepath.Join(ociLayoutDir, "blobs", manifestDesc.Digest.Algorithm().String(), manifestDesc.Digest.Encoded())
		manifestData, err := os.ReadFile(manifestPath)
		if err != nil {
			return errors.Errorf("reading manifest %s: %w", manifestDesc.Digest.String(), err)
		}

		// Keep the manifest blob itself
		keepBlobs[manifestDesc.Digest.String()] = true

		if manifestDesc.MediaType == v1.MediaTypeImageManifest || manifestDesc.MediaType == DockerManifestMediaType {
			// Single-platform manifest (OCI or Docker)
			var manifest v1.Manifest
			if err := json.Unmarshal(manifestData, &manifest); err != nil {
				return errors.Errorf("unmarshaling manifest %s: %w", manifestDesc.Digest.String(), err)
			}

			// Check if this platform should be kept
			platform := getPlatformFromDescriptor(manifestDesc)
			if shouldKeepPlatform(platform, keepPlatforms) {
				slog.InfoContext(ctx, "keeping single-platform manifest", "platform", platform, "digest", manifestDesc.Digest.String())
				newManifests = append(newManifests, manifestDesc)
				markBlobsForKeeping(manifest, keepBlobs)
			} else {
				slog.InfoContext(ctx, "removing single-platform manifest", "platform", platform, "digest", manifestDesc.Digest.String())
			}

		} else if manifestDesc.MediaType == v1.MediaTypeImageIndex || manifestDesc.MediaType == DockerManifestListMediaType {
			// Multi-platform manifest list (OCI or Docker)
			var manifestList v1.Index
			if err := json.Unmarshal(manifestData, &manifestList); err != nil {
				return errors.Errorf("unmarshaling manifest list %s: %w", manifestDesc.Digest.String(), err)
			}

			var newPlatformManifests []v1.Descriptor
			for _, platformManifest := range manifestList.Manifests {
				platform := getPlatformFromDescriptor(platformManifest)
				if shouldKeepPlatform(platform, keepPlatforms) {
					slog.InfoContext(ctx, "keeping platform in manifest list", "platform", platform, "digest", platformManifest.Digest.String())
					newPlatformManifests = append(newPlatformManifests, platformManifest)

					// Read and mark blobs for this platform manifest
					platformManifestPath := filepath.Join(ociLayoutDir, "blobs", platformManifest.Digest.Algorithm().String(), platformManifest.Digest.Encoded())
					platformManifestData, err := os.ReadFile(platformManifestPath)
					if err != nil {
						return errors.Errorf("reading platform manifest %s: %w", platformManifest.Digest.String(), err)
					}

					var platformManifestObj v1.Manifest
					if err := json.Unmarshal(platformManifestData, &platformManifestObj); err != nil {
						return errors.Errorf("unmarshaling platform manifest %s: %w", platformManifest.Digest.String(), err)
					}

					keepBlobs[platformManifest.Digest.String()] = true
					markBlobsForKeeping(platformManifestObj, keepBlobs)
				} else {
					slog.InfoContext(ctx, "removing platform from manifest list", "platform", platform, "digest", platformManifest.Digest.String())
				}
			}

			if len(newPlatformManifests) > 0 {
				// Update the manifest list with only kept platforms
				manifestList.Manifests = newPlatformManifests

				if !dryRun {
					// Write the updated manifest list
					updatedManifestData, err := json.Marshal(manifestList)
					if err != nil {
						return errors.Errorf("marshaling updated manifest list: %w", err)
					}

					if err := os.WriteFile(manifestPath, updatedManifestData, 0644); err != nil {
						return errors.Errorf("writing updated manifest list: %w", err)
					}
				}

				newManifests = append(newManifests, manifestDesc)
				slog.InfoContext(ctx, "updated manifest list", "kept_platforms", len(newPlatformManifests), "digest", manifestDesc.Digest.String())
			} else {
				slog.InfoContext(ctx, "removing empty manifest list", "digest", manifestDesc.Digest.String())
			}
		} else {
			// Unknown manifest type, keep it
			slog.WarnContext(ctx, "unknown manifest type, keeping", "media_type", manifestDesc.MediaType, "digest", manifestDesc.Digest.String())
			newManifests = append(newManifests, manifestDesc)
		}
	}

	// Update the index with only kept manifests
	index.Manifests = newManifests

	if !dryRun {
		// Write the updated index
		updatedIndexData, err := json.Marshal(index)
		if err != nil {
			return errors.Errorf("marshaling updated index: %w", err)
		}

		if err := os.WriteFile(indexPath, updatedIndexData, 0644); err != nil {
			return errors.Errorf("writing updated index: %w", err)
		}
	}

	slog.InfoContext(ctx, "updated index", "kept_manifests", len(newManifests), "original_manifests", len(index.Manifests))

	// Remove unused blobs
	return removeUnusedBlobs(ctx, ociLayoutDir, keepBlobs, dryRun)
}

func getPlatformFromDescriptor(desc v1.Descriptor) string {
	if desc.Platform == nil {
		return "unknown"
	}
	platform := desc.Platform.OS + "/" + desc.Platform.Architecture
	if desc.Platform.Variant != "" {
		platform += "/" + desc.Platform.Variant
	}
	return platform
}

func shouldKeepPlatform(platform string, keepPlatforms []units.Platform) bool {
	if platform == "unknown" {
		return true // Keep unknown platforms to be safe
	}

	for _, keep := range keepPlatforms {
		if string(keep) == platform {
			return true
		}
		// Also check resolved variants
		if string(units.ResolvePlatformVariant(platform)) == string(keep) {
			return true
		}
	}
	return false
}

func markBlobsForKeeping(manifest v1.Manifest, keepBlobs map[string]bool) {
	// Keep config blob
	keepBlobs[manifest.Config.Digest.String()] = true

	// Keep all layer blobs
	for _, layer := range manifest.Layers {
		keepBlobs[layer.Digest.String()] = true
	}
}

func removeUnusedBlobs(ctx context.Context, ociLayoutDir string, keepBlobs map[string]bool, dryRun bool) error {
	blobsDir := filepath.Join(ociLayoutDir, "blobs")

	var removedCount, keptCount int
	var removedSize, keptSize int64

	// Walk through all blobs
	err := filepath.Walk(blobsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// Get relative path from blobs directory
		relPath, err := filepath.Rel(blobsDir, path)
		if err != nil {
			return err
		}

		// Convert path to digest format (algorithm:encoded)
		parts := strings.Split(relPath, string(filepath.Separator))
		if len(parts) != 2 {
			slog.WarnContext(ctx, "unexpected blob path format", "path", relPath)
			return nil
		}

		digest := parts[0] + ":" + parts[1]

		if keepBlobs[digest] {
			keptCount++
			keptSize += info.Size()
			slog.DebugContext(ctx, "keeping blob", "digest", digest, "size", info.Size())
		} else {
			removedCount++
			removedSize += info.Size()

			if dryRun {
				slog.InfoContext(ctx, "would remove blob", "digest", digest, "size", info.Size(), "path", path)
			} else {
				slog.InfoContext(ctx, "removing blob", "digest", digest, "size", info.Size(), "path", path)
				if err := os.Truncate(path, 0); err != nil {
					return errors.Errorf("removing blob %s: %w", path, err)
				}
			}
		}

		return nil
	})

	if err != nil {
		return errors.Errorf("walking blobs directory: %w", err)
	}

	slog.InfoContext(ctx, "blob cleanup summary",
		"kept_blobs", keptCount,
		"removed_blobs", removedCount,
		"kept_size_mb", float64(keptSize)/(1024*1024),
		"removed_size_mb", float64(removedSize)/(1024*1024),
		"dry_run", dryRun)

	// Clean up empty directories
	if !dryRun {
		return cleanupEmptyDirs(ctx, blobsDir)
	}

	return nil
}

func cleanupEmptyDirs(ctx context.Context, blobsDir string) error {
	// Remove empty algorithm directories
	entries, err := os.ReadDir(blobsDir)
	if err != nil {
		return errors.Errorf("reading blobs directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			dirPath := filepath.Join(blobsDir, entry.Name())
			dirEntries, err := os.ReadDir(dirPath)
			if err != nil {
				continue
			}

			if len(dirEntries) == 0 {
				slog.InfoContext(ctx, "removing empty algorithm directory", "dir", dirPath)
				if err := os.Remove(dirPath); err != nil {
					slog.WarnContext(ctx, "failed to remove empty directory", "dir", dirPath, "error", err)
				}
			}
		}
	}

	return nil
}
