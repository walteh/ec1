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
	"github.com/containers/image/v5/docker"
	"github.com/containers/image/v5/image"
	"github.com/containers/image/v5/oci/layout"
	"github.com/containers/image/v5/signature"
	"github.com/containers/image/v5/types"
	"gitlab.com/tozd/go/errors"

	v1 "github.com/opencontainers/image-spec/specs-go/v1"

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

// ContainerMetadata contains OCI container metadata including entrypoint information
type ContainerMetadata struct {
	ImageRef     string              `json:"image_ref"`
	CreatedAt    string              `json:"created_at"`
	EC1Version   string              `json:"ec1_version"`
	ConfigPath   string              `json:"config_path"`
	Entrypoint   []string            `json:"entrypoint,omitempty"`
	Cmd          []string            `json:"cmd,omitempty"`
	Env          []string            `json:"env,omitempty"`
	WorkingDir   string              `json:"working_dir,omitempty"`
	User         string              `json:"user,omitempty"`
	Labels       map[string]string   `json:"labels,omitempty"`
	ExposedPorts map[string]struct{} `json:"exposed_ports,omitempty"`
}

// ContainerToVirtioDevice converts an OCI container image to a virtio device using the skopeo approach
func ContainerToVirtioDevice(ctx context.Context, opts ContainerToVirtioOptions) (virtio.VirtioDevice, *v1.Image, error) {
	slog.InfoContext(ctx, "converting OCI container to virtio device (skopeo approach)",
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

	// Create temporary directory for OCI layout extraction
	tempDir, err := os.MkdirTemp(opts.OutputDir, "oci-layout-*")
	if err != nil {
		return nil, nil, errors.Errorf("creating temp directory: %w", err)
	}

	// Pull and extract the container image using skopeo approach
	metadata, err := pullAndExtractImageSkopeo(ctx, opts.ImageRef, tempDir, opts.Platform)
	if err != nil {
		return nil, nil, errors.Errorf("pulling and extracting image: %w", err)
	}

	// Create containerd mount manager
	assembledFS := filepath.Join(tempDir, "rootfs")
	mountMng := NewContainerdMountManager(assembledFS, opts.MountPoint, opts.ReadOnly, opts.ImageRef)

	// Mount the filesystem using containerd's mount system
	err = mountMng.Mount(ctx)
	if err != nil {
		return nil, nil, errors.Errorf("mounting containerd filesystem: %w", err)
	}

	// Debug: Check if the mount actually worked
	entries, err := os.ReadDir(opts.MountPoint)
	if err != nil {
		slog.WarnContext(ctx, "failed to read mount point after mounting", "error", err)
	} else {
		slog.InfoContext(ctx, "mount point contents after mounting", "count", len(entries))
		for _, entry := range entries {
			slog.InfoContext(ctx, "mount point entry", "name", entry.Name(), "is_dir", entry.IsDir())
		}
	}

	// Also check the source directory
	sourceEntries, err := os.ReadDir(assembledFS)
	if err != nil {
		slog.WarnContext(ctx, "failed to read source directory", "error", err)
	} else {
		slog.InfoContext(ctx, "source directory contents", "count", len(sourceEntries))
		for _, entry := range sourceEntries {
			slog.InfoContext(ctx, "source entry", "name", entry.Name(), "is_dir", entry.IsDir())
		}
	}

	// Write container metadata to the rootfs for the supervisor to use
	err = writeContainerMetadataToRootfs(ctx, assembledFS, metadata)
	if err != nil {
		slog.WarnContext(ctx, "failed to write container metadata", "error", err)
		// Don't fail the whole operation, just log the warning
	}

	// Create virtio device pointing to the mounted filesystem
	device, err := virtio.VirtioFsNew(assembledFS, "rootfs")
	if err != nil {
		// Cleanup on failure
		mountMng.Unmount(ctx)
		return nil, nil, errors.Errorf("creating virtio device: %w", err)
	}

	slog.InfoContext(ctx, "successfully created containerd-mounted virtio device from container",
		"image", opts.ImageRef,
		"mount_point", opts.MountPoint)

	return device, metadata, nil
}

// ContainerdMountManager manages a containerd mount for OCI container content
type ContainerdMountManager struct {
	sourceDir  string
	mountPoint string
	readOnly   bool
	mounted    bool
	imageRef   string
}

// NewContainerdMountManager creates a new containerd mount manager for OCI container content
func NewContainerdMountManager(sourceDir, mountPoint string, readOnly bool, imageRef string) *ContainerdMountManager {
	return &ContainerdMountManager{
		sourceDir:  sourceDir,
		mountPoint: mountPoint,
		readOnly:   readOnly,
		imageRef:   imageRef,
	}
}

// Mount mounts the OCI container content using containerd's mount system
func (m *ContainerdMountManager) Mount(ctx context.Context) error {
	if m.mounted {
		return errors.New("filesystem already mounted")
	}

	// Verify source directory exists and has content
	sourceInfo, err := os.Stat(m.sourceDir)
	if err != nil {
		return errors.Errorf("source directory does not exist: %w", err)
	}
	if !sourceInfo.IsDir() {
		return errors.Errorf("source is not a directory: %s", m.sourceDir)
	}

	// Check source directory contents
	sourceEntries, err := os.ReadDir(m.sourceDir)
	if err != nil {
		return errors.Errorf("cannot read source directory: %w", err)
	}
	slog.InfoContext(ctx, "source directory before mount", "path", m.sourceDir, "entries", len(sourceEntries))

	// Verify mount point exists
	mountInfo, err := os.Stat(m.mountPoint)
	if err != nil {
		return errors.Errorf("mount point does not exist: %w", err)
	}
	if !mountInfo.IsDir() {
		return errors.Errorf("mount point is not a directory: %s", m.mountPoint)
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

	err = mount.All(mounts, m.mountPoint)
	if err != nil {
		return errors.Errorf("mounting containerd filesystem: %w", err)
	}

	// err = Runner(ctx, RunnerOpts{
	// 	Target: m.mountPoint,
	// 	Ref:    m.imageRef,
	// 	Platform: platforms.Platform{
	// 		Architecture: "arm64",
	// 		OS:           "linux",
	// 	}, // TODO: use opts.Platform
	// 	Rw:          !m.readOnly,
	// 	Snapshotter: "overlayfs",
	// })
	// if err != nil {
	// 	return errors.Errorf("mounting containerd filesystem: %w", err)
	// }

	// Retry mount operation to handle intermittent FUSE-T issues
	// var lastErr error
	// maxRetries := 3
	// for attempt := 1; attempt <= maxRetries; attempt++ {
	// 	slog.InfoContext(ctx, "attempting mount", "attempt", attempt, "max_retries", maxRetries)

	// 	// Use containerd's mount.All to perform the mount
	// 	err = mount.All(mounts, m.mountPoint)
	// 	if err != nil {
	// 		lastErr = err
	// 		slog.WarnContext(ctx, "mount attempt failed", "attempt", attempt, "error", err)
	// 		if attempt < maxRetries {
	// 			// Wait a bit before retrying
	// 			time.Sleep(time.Duration(attempt) * 500 * time.Millisecond)
	// 			continue
	// 		}
	// 		return errors.Errorf("containerd mount failed after %d attempts: %w", maxRetries, err)
	// 	}

	// 	// Verify the mount worked by checking the mount point contents
	// 	mountEntries, err := os.ReadDir(m.mountPoint)
	// 	if err != nil {
	// 		lastErr = err
	// 		slog.WarnContext(ctx, "mount verification failed", "attempt", attempt, "error", err)
	// 		if attempt < maxRetries {
	// 			// Unmount and retry
	// 			mount.UnmountAll(m.mountPoint, 0)
	// 			time.Sleep(time.Duration(attempt) * 500 * time.Millisecond)
	// 			continue
	// 		}
	// 		return errors.Errorf("mount verification failed after %d attempts: %w", maxRetries, err)
	// 	}

	// 	slog.InfoContext(ctx, "mount point after mounting", "path", m.mountPoint, "entries", len(mountEntries), "attempt", attempt)

	// 	if len(mountEntries) == 0 {
	// 		lastErr = errors.New("mount point is empty")
	// 		slog.WarnContext(ctx, "mount appears empty", "attempt", attempt)
	// 		if attempt < maxRetries {
	// 			// Unmount and retry
	// 			mount.UnmountAll(m.mountPoint, 0)
	// 			time.Sleep(time.Duration(attempt) * 500 * time.Millisecond)
	// 			continue
	// 		}
	// 		return errors.Errorf("mount appears to have failed - mount point is empty after %d attempts", maxRetries)
	// 	}

	// 	// Success!
	// 	m.mounted = true
	// 	slog.InfoContext(ctx, "successfully mounted with containerd", "mount_point", m.mountPoint, "entry_count", len(mountEntries), "attempt", attempt)
	// 	return nil
	// }

	// return errors.Errorf("mount failed after %d attempts, last error: %w", maxRetries, lastErr)
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

// pullAndExtractImageSkopeo pulls a container image using the skopeo approach and extracts it to OCI layout
func pullAndExtractImageSkopeo(ctx context.Context, imageRef, destDir string, sysCtx *types.SystemContext) (*v1.Image, error) {
	slog.InfoContext(ctx, "pulling container image using skopeo approach", "image", imageRef)

	// Create policy context (allow all for now - in production this should be more restrictive)
	policyContext, err := signature.NewPolicyContext(&signature.Policy{
		Default: []signature.PolicyRequirement{signature.NewPRInsecureAcceptAnything()},
	})
	if err != nil {
		return nil, errors.Errorf("creating policy context: %w", err)
	}
	defer policyContext.Destroy()

	// Parse source image reference
	srcRef, err := docker.ParseReference("//" + imageRef)
	if err != nil {
		return nil, errors.Errorf("parsing source reference: %w", err)
	}

	// Create OCI layout destination
	ociLayoutDir := filepath.Join(destDir, "oci-layout")
	if err := os.MkdirAll(ociLayoutDir, 0755); err != nil {
		return nil, errors.Errorf("creating oci layout directory: %w", err)
	}

	destRef, err := layout.NewReference(ociLayoutDir, imageRef)
	if err != nil {
		return nil, errors.Errorf("creating oci layout reference: %w", err)
	}

	// Copy image from source to OCI layout
	_, err = copy.Image(ctx, policyContext, destRef, srcRef, &copy.Options{
		SourceCtx:      sysCtx,
		DestinationCtx: sysCtx,
		ReportWriter:   os.Stdout,
	})
	if err != nil {
		return nil, errors.Errorf("copying image: %w", err)
	}

	// Create image source from the copied OCI layout to extract configuration
	imgSrc, err := destRef.NewImageSource(ctx, sysCtx)
	if err != nil {
		return nil, errors.Errorf("creating image source: %w", err)
	}
	defer imgSrc.Close()

	// Create image from source to get configuration
	img, err := image.FromSource(ctx, sysCtx, imgSrc)
	if err != nil {
		return nil, errors.Errorf("creating image from source: %w", err)
	}
	defer img.Close()

	// Get OCI configuration
	ociConfig, err := img.OCIConfig(ctx)
	if err != nil {
		return nil, errors.Errorf("getting OCI config: %w", err)
	}

	// Extract the filesystem layers to a rootfs directory
	err = extractLayersFromOCILayout(ctx, ociLayoutDir, destDir, img)
	if err != nil {
		return nil, errors.Errorf("extracting layers: %w", err)
	}

	// // Create metadata from OCI config
	// metadata := &ContainerMetadata{
	// 	ImageRef:   imageRef,
	// 	CreatedAt:  time.Now().Format(time.RFC3339),
	// 	EC1Version: "dev", // TODO: get actual version
	// 	ConfigPath: "/ec1/config.json",
	// }

	// // Extract configuration details
	// metadata.Entrypoint = ociConfig.Config.Entrypoint
	// metadata.Cmd = ociConfig.Config.Cmd
	// metadata.Env = ociConfig.Config.Env
	// metadata.WorkingDir = ociConfig.Config.WorkingDir
	// metadata.User = ociConfig.Config.User
	// metadata.Labels = ociConfig.Config.Labels
	// if ociConfig.Config.ExposedPorts != nil {
	// 	metadata.ExposedPorts = make(map[string]struct{})
	// 	for port := range ociConfig.Config.ExposedPorts {
	// 		metadata.ExposedPorts[port] = struct{}{}
	// 	}
	// }

	slog.InfoContext(ctx, "successfully extracted container image with metadata",
		"dest", destDir,
		"entrypoint", ociConfig.Config.Entrypoint,
		"cmd", ociConfig.Config.Cmd)

	return ociConfig, nil
}

// extractLayersFromOCILayout extracts the filesystem layers from an OCI layout to create a rootfs
func extractLayersFromOCILayout(ctx context.Context, ociLayoutDir, destDir string, img types.Image) error {
	slog.InfoContext(ctx, "extracting layers from OCI layout", "oci_layout", ociLayoutDir, "dest", destDir)

	// Create the destination filesystem directory
	fsDir := filepath.Join(destDir, "rootfs")
	if err := os.MkdirAll(fsDir, 0755); err != nil {
		return errors.Errorf("creating filesystem directory: %w", err)
	}

	// Get layer information
	layerInfos := img.LayerInfos()
	slog.InfoContext(ctx, "found layers", "count", len(layerInfos))

	// For now, we'll use a simplified approach that works with the existing assembleContainerFilesystem
	// In a full implementation, you'd want to extract each layer blob and apply it in order

	// Create a temporary manifest.json file that assembleContainerFilesystem expects
	manifestPath := filepath.Join(ociLayoutDir, "manifest.json")
	manifest := struct {
		Layers []struct {
			Digest string `json:"digest"`
		} `json:"layers"`
	}{}

	for _, layerInfo := range layerInfos {
		manifest.Layers = append(manifest.Layers, struct {
			Digest string `json:"digest"`
		}{
			Digest: layerInfo.Digest.String(),
		})
	}

	manifestData, err := json.Marshal(manifest)
	if err != nil {
		return errors.Errorf("marshaling manifest: %w", err)
	}

	if err := os.WriteFile(manifestPath, manifestData, 0644); err != nil {
		return errors.Errorf("writing manifest: %w", err)
	}

	// Use the existing assembleContainerFilesystem function
	err = assembleContainerFilesystem(ctx, ociLayoutDir, destDir)
	if err != nil {
		return errors.Errorf("assembling container filesystem: %w", err)
	}

	slog.InfoContext(ctx, "successfully extracted layers to rootfs", "dest", fsDir)
	return nil
}

// writeContainerMetadataToRootfs writes container metadata to the rootfs for the supervisor
func writeContainerMetadataToRootfs(ctx context.Context, rootfsPath string, metadata *v1.Image) error {
	slog.InfoContext(ctx, "writing container metadata to rootfs", "rootfs", rootfsPath)

	// Create ec1 directory in the rootfs
	ec1Dir := filepath.Join(rootfsPath, "ec1")
	if err := os.MkdirAll(ec1Dir, 0755); err != nil {
		return errors.Errorf("creating ec1 directory: %w", err)
	}

	// Write metadata as JSON
	metadataPath := filepath.Join(ec1Dir, "ec1-container-metadata.json")
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return errors.Errorf("marshaling metadata: %w", err)
	}

	if err := os.WriteFile(metadataPath, metadataJSON, 0644); err != nil {
		return errors.Errorf("writing metadata: %w", err)
	}

	// Also write a simplified config.json for compatibility
	// configPath := filepath.Join(ec1Dir, "ec1-container-config.json")
	// config := map[string]interface{}{
	// 	"ociVersion": "1.0.0",
	// 	"config": map[string]interface{}{
	// 		"Entrypoint": metadata.Entrypoint,
	// 		"Cmd":        metadata.Cmd,
	// 		"Env":        metadata.Env,
	// 		"WorkingDir": metadata.WorkingDir,
	// 		"User":       metadata.User,
	// 		"Labels":     metadata.Labels,
	// 	},
	// 	"rootfs": map[string]interface{}{
	// 		"type": "layers",
	// 	},
	// 	"history": []map[string]interface{}{
	// 		{
	// 			"created":    metadata.CreatedAt,
	// 			"created_by": "ec1 oci converter",
	// 			"comment":    "Converted from " + metadata.ImageRef,
	// 		},
	// 	},
	// }

	// configJSON, err := json.Marshal(metadata)
	// if err != nil {
	// 	return errors.Errorf("marshaling config: %w", err)
	// }

	// if err := os.WriteFile(configPath, configJSON, 0644); err != nil {
	// 	return errors.Errorf("writing config: %w", err)
	// }

	slog.InfoContext(ctx, "successfully wrote container metadata",
		"metadata_path", metadataPath)

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
