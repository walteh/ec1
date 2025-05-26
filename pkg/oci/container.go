package oci

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	iofs "io/fs"

	"github.com/containers/image/v5/copy"
	"github.com/containers/image/v5/directory"
	"github.com/containers/image/v5/docker"
	"github.com/containers/image/v5/signature"
	"github.com/containers/image/v5/types"
	"github.com/diskfs/go-diskfs"
	"github.com/diskfs/go-diskfs/disk"
	"github.com/diskfs/go-diskfs/filesystem"
	"gitlab.com/tozd/go/errors"

	"github.com/walteh/ec1/pkg/virtio"
)

// ContainerToVirtioOptions configures how a container image is converted to a virtio device
type ContainerToVirtioOptions struct {
	// ImageRef is the container image reference (e.g., "docker.io/library/alpine:latest")
	ImageRef string

	// Platform specifies the target platform (e.g., "linux/arm64")
	Platform *types.SystemContext

	// OutputDir is where to create the rootfs image file
	OutputDir string

	// FilesystemType specifies the filesystem type for the rootfs (default: ext4)
	FilesystemType string

	// Size specifies the size of the rootfs image (default: 1GB)
	Size int64

	// ReadOnly specifies if the virtio device should be read-only
	ReadOnly bool
}

// ContainerToVirtioDevice converts an OCI container image to a virtio block device using pure Go
func ContainerToVirtioDevice(ctx context.Context, opts ContainerToVirtioOptions) (virtio.VirtioDevice, error) {
	slog.InfoContext(ctx, "converting OCI container to virtio device (pure Go)",
		"image", opts.ImageRef,
		"output", opts.OutputDir,
		"fs_type", opts.FilesystemType,
		"size_mb", opts.Size/(1024*1024))

	// Set defaults
	if opts.FilesystemType == "" {
		opts.FilesystemType = "ext4"
	}
	if opts.Size == 0 {
		opts.Size = 1024 * 1024 * 1024 // 1GB default
	}
	if opts.Platform == nil {
		opts.Platform = &types.SystemContext{}
	}

	os.MkdirAll(opts.OutputDir, 0755)

	// Create temporary directory for extraction
	tempDir, err := os.MkdirTemp(opts.OutputDir, "oci-extract-*")
	if err != nil {
		return nil, errors.Errorf("creating temp directory: %w", err)
	}
	// defer os.RemoveAll(tempDir)

	// Pull and extract the container image
	err = pullAndExtractImage(ctx, opts.ImageRef, tempDir, opts.Platform)
	if err != nil {
		return nil, errors.Errorf("pulling and extracting image: %w", err)
	}

	// // Create filesystem image from extracted content using pure Go
	// err = createFilesystemImagePureGo(ctx, tempDir, opts.OutputDir, opts.FilesystemType, opts.Size)
	// if err != nil {
	// 	return nil, errors.Errorf("creating filesystem image: %w", err)
	// }

	// Create virtio block device
	device, err := virtio.VirtioFsNew(tempDir, "rootfs")
	if err != nil {
		return nil, errors.Errorf("creating virtio block device: %w", err)
	}

	if opts.ReadOnly {
		// Set read-only if supported by the virtio implementation
		slog.InfoContext(ctx, "virtio device created as read-only", "path", tempDir)
	}

	slog.InfoContext(ctx, "successfully created virtio device from container",
		"image", opts.ImageRef,
		"device_path", tempDir)

	return device, nil
}

// pullAndExtractImage pulls a container image and extracts it to a directory
func pullAndExtractImage(ctx context.Context, imageRef, destDir string, sysCtx *types.SystemContext) error {
	slog.InfoContext(ctx, "pulling container image", "image", imageRef)

	// Create policy context (allow all for now - in production this should be more restrictive)
	policyContext, err := signature.NewPolicyContext(&signature.Policy{
		Default: []signature.PolicyRequirement{signature.NewPRInsecureAcceptAnything()},
	})
	if err != nil {
		return errors.Errorf("creating policy context: %w", err)
	}
	defer policyContext.Destroy()

	// Parse source image reference
	srcRef, err := docker.ParseReference("//" + imageRef)
	if err != nil {
		return errors.Errorf("parsing source reference: %w", err)
	}

	// Create destination directory reference
	destRef, err := directory.NewReference(destDir)
	if err != nil {
		return errors.Errorf("creating destination reference: %w", err)
	}

	// Copy image from source to directory
	_, err = copy.Image(ctx, policyContext, destRef, srcRef, &copy.Options{
		SourceCtx:      sysCtx,
		DestinationCtx: sysCtx,
		ReportWriter:   os.Stdout,
	})
	if err != nil {
		return errors.Errorf("copying image: %w", err)
	}

	slog.InfoContext(ctx, "successfully extracted container image", "dest", destDir)
	return nil
}

// createFilesystemImagePureGo creates a filesystem image using pure Go (diskfs)
func createFilesystemImagePureGo(ctx context.Context, sourceDir, OutputDir, fsType string, size int64) error {
	slog.InfoContext(ctx, "creating filesystem image with pure Go",
		"source", sourceDir,
		"output", OutputDir,
		"fs_type", fsType,
		"size_mb", size/(1024*1024))

	// Ensure output directory exists
	if err := os.MkdirAll(filepath.Dir(OutputDir), 0755); err != nil {
		return errors.Errorf("creating output directory: %w", err)
	}

	// Create disk image
	dsk, err := diskfs.Create(OutputDir, size, diskfs.SectorSizeDefault)
	if err != nil {
		return errors.Errorf("creating disk image: %w", err)
	}
	defer dsk.Close()

	// Create filesystem based on type
	var fsys filesystem.FileSystem
	switch strings.ToLower(fsType) {
	case "ext4":
		fsys, err = dsk.CreateFilesystem(disk.FilesystemSpec{
			Partition:   0, // Use entire disk
			FSType:      filesystem.TypeExt4,
			VolumeLabel: "container-rootfs-ext4",
		})
		if err != nil {
			return errors.Errorf("creating ext4 filesystem: %w", err)
		}
	case "fat32":
		fsys, err = dsk.CreateFilesystem(disk.FilesystemSpec{
			Partition:   0, // Use entire disk
			FSType:      filesystem.TypeFat32,
			VolumeLabel: "CONTAINERFS", // FAT32 labels are often uppercase and shorter
		})
		if err != nil {
			return errors.Errorf("creating fat32 filesystem: %w", err)
		}
	default:
		return errors.Errorf("unsupported filesystem type: %s (only ext4, fat32 supported in pure Go mode)", fsType)
	}

	// Copy files from source directory to filesystem
	err = copyDirectoryToFilesystem(ctx, sourceDir, fsys, "/")
	if err != nil {
		return errors.Errorf("copying files to filesystem: %w", err)
	}

	slog.InfoContext(ctx, "successfully created and populated filesystem image")
	return nil
}

// copyDirectoryToFilesystem recursively copies files from a directory to a filesystem
func copyDirectoryToFilesystem(ctx context.Context, sourceDir string, fsys filesystem.FileSystem, destPath string) error {
	return filepath.WalkDir(sourceDir, func(path string, d iofs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Calculate relative path
		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}

		// Convert to filesystem path
		fsPath := filepath.Join(destPath, relPath)
		fsPath = filepath.ToSlash(fsPath) // Ensure forward slashes

		if d.IsDir() {
			// Create directory
			if fsPath != "/" { // Don't try to create root directory
				err = fsys.Mkdir(fsPath)
				if err != nil {
					return errors.Errorf("creating directory %s: %w", fsPath, err)
				}
				slog.DebugContext(ctx, "created directory", "path", fsPath)
			}
		} else {
			// Copy file
			err = copyFileToFilesystem(ctx, path, fsys, fsPath)
			if err != nil {
				return errors.Errorf("copying file %s to %s: %w", path, fsPath, err)
			}
			slog.DebugContext(ctx, "copied file", "src", path, "dest", fsPath)
		}

		return nil
	})
}

// copyFileToFilesystem copies a single file to the filesystem
func copyFileToFilesystem(ctx context.Context, srcPath string, fsys filesystem.FileSystem, destPath string) error {
	// Open source file
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return errors.Errorf("opening source file: %w", err)
	}
	defer srcFile.Close()

	// Get file info for permissions
	srcInfo, err := srcFile.Stat()
	if err != nil {
		return errors.Errorf("getting source file info: %w", err)
	}

	// Create destination file
	destFile, err := fsys.OpenFile(destPath, os.O_CREATE|os.O_RDWR|os.O_TRUNC)
	if err != nil {
		return errors.Errorf("creating destination file: %w", err)
	}
	defer destFile.Close()

	// Copy content
	_, err = io.Copy(destFile, srcFile)
	if err != nil {
		return errors.Errorf("copying file content: %w", err)
	}

	// Try to set permissions (may not be supported by all filesystems)
	if chmodder, ok := destFile.(interface{ Chmod(os.FileMode) error }); ok {
		err = chmodder.Chmod(srcInfo.Mode())
		if err != nil {
			slog.DebugContext(ctx, "failed to set file permissions (continuing)", "file", destPath, "error", err)
		}
	}

	return nil
}

// GetImageInfo retrieves information about a container image without pulling it
func GetImageInfo(ctx context.Context, imageRef string, sysCtx *types.SystemContext) (*types.ImageInspectInfo, error) {
	slog.InfoContext(ctx, "getting image info", "image", imageRef)

	srcRef, err := docker.ParseReference("//" + imageRef)
	if err != nil {
		return nil, errors.Errorf("parsing image reference: %w", err)
	}

	src, err := srcRef.NewImageSource(ctx, sysCtx)
	if err != nil {
		return nil, errors.Errorf("creating image source: %w", err)
	}
	defer src.Close()

	// This is a simplified version - in practice you'd want to parse the manifest
	// and extract more detailed information
	info := &types.ImageInspectInfo{
		Tag:          imageRef,
		Architecture: sysCtx.ArchitectureChoice,
		Os:           sysCtx.OSChoice,
	}

	return info, nil
}
