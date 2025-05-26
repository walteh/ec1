package oci

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/containerd/containerd/v2/core/mount"
	"github.com/containers/image/v5/copy"
	"github.com/containers/image/v5/directory"
	"github.com/containers/image/v5/docker"
	"github.com/containers/image/v5/signature"
	"github.com/containers/image/v5/types"
	"gitlab.com/tozd/go/errors"

	"github.com/walteh/ec1/pkg/virtio"
)

// ContainerToVirtioOptions configures how a container image is converted to a virtio device
type ContainerToVirtioOptions struct {
	// ImageRef is the container image reference (e.g., "docker.io/library/alpine:latest")
	ImageRef string

	// Platform specifies the target platform (e.g., "linux/arm64")
	Platform *types.SystemContext

	// OutputDir is where to create the rootfs mount point
	OutputDir string

	// ReadOnly specifies if the virtio device should be read-only
	ReadOnly bool

	// MountPoint is the directory where the filesystem will be mounted
	MountPoint string
}

// ContainerToVirtioDevice converts an OCI container image to a virtio device using containerd's mount system
func ContainerToVirtioDevice(ctx context.Context, opts ContainerToVirtioOptions) (virtio.VirtioDevice, error) {
	slog.InfoContext(ctx, "converting OCI container to virtio device (containerd mount)",
		"image", opts.ImageRef,
		"output", opts.OutputDir,
		"mount_point", opts.MountPoint)

	// Set defaults
	if opts.Platform == nil {
		opts.Platform = &types.SystemContext{}
	}
	if opts.MountPoint == "" {
		opts.MountPoint = filepath.Join(opts.OutputDir, "rootfs")
	}

	os.MkdirAll(opts.OutputDir, 0755)
	os.MkdirAll(opts.MountPoint, 0755)

	// Create temporary directory for extraction
	tempDir, err := os.MkdirTemp(opts.OutputDir, "oci-extract-*")
	if err != nil {
		return nil, errors.Errorf("creating temp directory: %w", err)
	}

	// Pull and extract the container image
	err = pullAndExtractImage(ctx, opts.ImageRef, tempDir, opts.Platform)
	if err != nil {
		return nil, errors.Errorf("pulling and extracting image: %w", err)
	}

	// Create containerd mount manager
	assembledFS := filepath.Join(tempDir, "rootfs")
	mountMng := NewContainerdMountManager(assembledFS, opts.MountPoint, opts.ReadOnly)

	// Mount the filesystem using containerd's mount system
	err = mountMng.Mount(ctx)
	if err != nil {
		return nil, errors.Errorf("mounting containerd filesystem: %w", err)
	}

	// Create virtio device pointing to the mounted filesystem
	device, err := virtio.VirtioFsNew(opts.MountPoint, "rootfs")
	if err != nil {
		// Cleanup on failure
		mountMng.Unmount(ctx)
		return nil, errors.Errorf("creating virtio device: %w", err)
	}

	slog.InfoContext(ctx, "successfully created containerd-mounted virtio device from container",
		"image", opts.ImageRef,
		"mount_point", opts.MountPoint)

	return device, nil
}

// ContainerdMountManager manages a containerd mount for OCI container content
type ContainerdMountManager struct {
	sourceDir  string
	mountPoint string
	readOnly   bool
	mounted    bool
}

// NewContainerdMountManager creates a new containerd mount manager for OCI container content
func NewContainerdMountManager(sourceDir, mountPoint string, readOnly bool) *ContainerdMountManager {
	return &ContainerdMountManager{
		sourceDir:  sourceDir,
		mountPoint: mountPoint,
		readOnly:   readOnly,
	}
}

// Mount mounts the OCI container content using containerd's mount system
func (m *ContainerdMountManager) Mount(ctx context.Context) error {
	if m.mounted {
		return errors.New("filesystem already mounted")
	}

	// Create mount options
	var options []string
	if m.readOnly {
		options = append(options, "ro")
	}

	// Create containerd mount
	mounts := []mount.Mount{
		{
			Type:    "bind",
			Source:  m.sourceDir,
			Options: options,
		},
	}

	slog.InfoContext(ctx, "mounting with containerd mount system",
		"source", m.sourceDir,
		"mount", m.mountPoint,
		"readonly", m.readOnly,
		"options", options)

	// Use containerd's mount.All to perform the mount
	err := mount.All(mounts, m.mountPoint)
	if err != nil {
		return errors.Errorf("containerd mount failed: %w", err)
	}

	m.mounted = true
	slog.InfoContext(ctx, "successfully mounted with containerd", "mount_point", m.mountPoint)
	return nil
}

// Unmount unmounts the containerd filesystem
func (m *ContainerdMountManager) Unmount(ctx context.Context) error {
	if !m.mounted {
		return nil
	}

	// Use containerd's unmount function
	err := mount.UnmountAll(m.mountPoint, 0)
	if err != nil {
		return errors.Errorf("unmounting containerd filesystem: %w", err)
	}

	m.mounted = false
	slog.InfoContext(ctx, "successfully unmounted containerd mount", "mount_point", m.mountPoint)
	return nil
}

// pullAndExtractImage pulls a container image and extracts it to a directory
func pullAndExtractImage(ctx context.Context, imageRef, destDir string, sysCtx *types.SystemContext) error {
	slog.InfoContext(ctx, "pulling container image", "image", imageRef)

	// Create temporary directory for raw image extraction
	rawDir, err := os.MkdirTemp(destDir, "raw-image-*")
	if err != nil {
		return errors.Errorf("creating temp directory: %w", err)
	}
	defer os.RemoveAll(rawDir)

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

	// Create destination directory reference for raw extraction
	destRef, err := directory.NewReference(rawDir)
	if err != nil {
		return errors.Errorf("creating destination reference: %w", err)
	}

	// Copy image from source to directory (this gets the raw layers)
	_, err = copy.Image(ctx, policyContext, destRef, srcRef, &copy.Options{
		SourceCtx:      sysCtx,
		DestinationCtx: sysCtx,
		ReportWriter:   os.Stdout,
	})
	if err != nil {
		return errors.Errorf("copying image: %w", err)
	}

	// Now assemble the layers into the final filesystem
	err = assembleContainerFilesystem(ctx, rawDir, destDir)
	if err != nil {
		return errors.Errorf("assembling container filesystem: %w", err)
	}

	slog.InfoContext(ctx, "successfully extracted and assembled container image", "dest", destDir)
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

// assembleContainerFilesystem reads the OCI image layers and assembles them into a complete filesystem
func assembleContainerFilesystem(ctx context.Context, rawDir, destDir string) error {
	slog.InfoContext(ctx, "assembling container filesystem", "raw", rawDir, "dest", destDir)

	// Read the manifest to get layer information
	manifestPath := filepath.Join(rawDir, "manifest.json")
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		return errors.Errorf("reading manifest: %w", err)
	}

	var manifest struct {
		Layers []struct {
			Digest string `json:"digest"`
		} `json:"layers"`
	}
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		return errors.Errorf("parsing manifest: %w", err)
	}

	// Create the destination filesystem directory
	fsDir := filepath.Join(destDir, "filesystem")
	if err := os.MkdirAll(fsDir, 0755); err != nil {
		return errors.Errorf("creating filesystem directory: %w", err)
	}

	// Extract each layer in order
	for i, layer := range manifest.Layers {
		// Remove the "sha256:" prefix from digest to get the filename
		layerFile := strings.TrimPrefix(layer.Digest, "sha256:")
		layerPath := filepath.Join(rawDir, layerFile)

		slog.InfoContext(ctx, "extracting layer", "layer", i+1, "total", len(manifest.Layers), "file", layerFile)

		if err := extractLayer(ctx, layerPath, fsDir); err != nil {
			return errors.Errorf("extracting layer %d: %w", i+1, err)
		}
	}

	// Move the assembled filesystem to the final destination
	finalDest := filepath.Join(destDir, "rootfs")
	if err := os.Rename(fsDir, finalDest); err != nil {
		return errors.Errorf("moving assembled filesystem: %w", err)
	}

	slog.InfoContext(ctx, "successfully assembled container filesystem", "dest", finalDest)
	return nil
}

// extractLayer extracts a single compressed layer tar file to the destination
func extractLayer(ctx context.Context, layerPath, destDir string) error {
	file, err := os.Open(layerPath)
	if err != nil {
		return errors.Errorf("opening layer file: %w", err)
	}
	defer file.Close()

	// Decompress the gzipped layer
	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return errors.Errorf("creating gzip reader: %w", err)
	}
	defer gzReader.Close()

	// Extract the tar archive
	tarReader := tar.NewReader(gzReader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return errors.Errorf("reading tar header: %w", err)
		}

		// Skip whiteout files (used for deletions in overlay filesystems)
		if strings.HasPrefix(filepath.Base(header.Name), ".wh.") {
			continue
		}

		targetPath := filepath.Join(destDir, header.Name)

		// Ensure the target directory exists
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return errors.Errorf("creating directory: %w", err)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, os.FileMode(header.Mode)); err != nil {
				return errors.Errorf("creating directory %s: %w", targetPath, err)
			}
		case tar.TypeReg:
			if err := extractFile(tarReader, targetPath, header); err != nil {
				return errors.Errorf("extracting file %s: %w", targetPath, err)
			}
		case tar.TypeSymlink:
			if err := os.Symlink(header.Linkname, targetPath); err != nil && !os.IsExist(err) {
				return errors.Errorf("creating symlink %s: %w", targetPath, err)
			}
		case tar.TypeLink:
			linkTarget := filepath.Join(destDir, header.Linkname)
			if err := os.Link(linkTarget, targetPath); err != nil && !os.IsExist(err) {
				return errors.Errorf("creating hard link %s: %w", targetPath, err)
			}
		default:
			slog.WarnContext(ctx, "skipping unsupported file type", "type", header.Typeflag, "file", header.Name)
		}
	}

	return nil
}

// extractFile extracts a single file from the tar reader
func extractFile(tarReader *tar.Reader, targetPath string, header *tar.Header) error {
	file, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
	if err != nil {
		return errors.Errorf("creating file: %w", err)
	}
	defer file.Close()

	if _, err := io.Copy(file, tarReader); err != nil {
		return errors.Errorf("copying file content: %w", err)
	}

	return nil
}
