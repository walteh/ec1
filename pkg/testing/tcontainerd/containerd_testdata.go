package tcontainerd

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/containerd/containerd/v2/client"
	"github.com/containerd/containerd/v2/core/content"
	"github.com/containerd/containerd/v2/core/images"
	"github.com/containerd/containerd/v2/pkg/namespaces"
	"github.com/opencontainers/go-digest"
	"gitlab.com/tozd/go/errors"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// ContainerdTestdataImporter handles importing compressed OCI layouts into containerd
type ContainerdTestdataImporter struct {
	client    *client.Client
	namespace string
}

// NewContainerdTestdataImporter creates a new importer for loading testdata into containerd
func NewContainerdTestdataImporter(client *client.Client, namespace string) (*ContainerdTestdataImporter, error) {
	if namespace == "" {
		namespace = "default"
	}

	return &ContainerdTestdataImporter{
		client:    client,
		namespace: namespace,
	}, nil
}

// PreloadTestImages loads all images from the testdata registry into containerd
func (i *ContainerdTestdataImporter) PreloadTestImages(ctx context.Context, registry map[string][]byte) error {
	ctx = namespaces.WithNamespace(ctx, i.namespace)

	for imageRef, compressedData := range registry {
		slog.InfoContext(ctx, "Preloading test image into containerd", "image", imageRef)

		if err := i.ImportImage(ctx, imageRef, compressedData); err != nil {
			return errors.Errorf("importing test image %s: %w", imageRef, err)
		}

		slog.InfoContext(ctx, "Successfully preloaded test image", "image", imageRef)
	}

	return nil
}

// ImportImage imports a single compressed OCI layout into containerd
func (i *ContainerdTestdataImporter) ImportImage(ctx context.Context, imageRef string, compressedData []byte) error {
	// Extract to temporary directory
	// tempDir, err := os.MkdirTemp("", "oci-import-*")
	// if err != nil {
	// 	return errors.Errorf("creating temp directory: %w", err)
	// }
	// defer os.RemoveAll(tempDir)

	data := bytes.NewReader(compressedData)

	gzdata, err := gzip.NewReader(data)
	if err != nil {
		return errors.Errorf("creating gzip reader: %w", err)
	}

	_, err = i.client.Import(ctx, gzdata, client.WithAllPlatforms(true), client.WithDigestRef(func(dgst digest.Digest) string {
		return imageRef
	}))
	if err != nil {
		return errors.Errorf("importing OCI layout: %w", err)
	}

	// if err := oci.ExtractCompressedOCI(ctx, compressedData, tempDir); err != nil {
	// 	return errors.Errorf("extracting compressed OCI layout: %w", err)
	// }

	return nil

	// // Import the OCI layout into containerd
	// return i.importOCILayout(ctx, imageRef, tempDir)
}

// importOCILayout imports an OCI layout directory into containerd's content and image stores
func (i *ContainerdTestdataImporter) importOCILayout(ctx context.Context, imageRef, ociLayoutPath string) error {
	// Read the OCI index
	indexPath := filepath.Join(ociLayoutPath, "index.json")
	indexData, err := os.ReadFile(indexPath)
	if err != nil {
		return errors.Errorf("reading OCI index: %w", err)
	}

	var index ocispec.Index
	if err := json.Unmarshal(indexData, &index); err != nil {
		return errors.Errorf("parsing OCI index: %w", err)
	}

	contentStore := i.client.ContentStore()

	// Import all blobs first
	blobsDir := filepath.Join(ociLayoutPath, "blobs")
	if err := i.importBlobs(ctx, contentStore, blobsDir); err != nil {
		return errors.Errorf("importing blobs: %w", err)
	}

	// Create image record for each manifest in the index
	for _, manifest := range index.Manifests {
		if err := i.createImageRecord(ctx, imageRef, manifest); err != nil {
			return errors.Errorf("creating image record for %s: %w", imageRef, err)
		}
	}

	return nil
}

// importBlobs walks the blobs directory and imports all blobs into containerd's content store
func (i *ContainerdTestdataImporter) importBlobs(ctx context.Context, store content.Store, blobsDir string) error {
	return filepath.Walk(blobsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// Extract digest from file path (blobs/sha256/abc123...)
		relPath, err := filepath.Rel(blobsDir, path)
		if err != nil {
			return errors.Errorf("getting relative path: %w", err)
		}

		// Parse digest from path
		digestStr := filepath.Dir(relPath) + ":" + filepath.Base(relPath)
		dgst, err := digest.Parse(digestStr)
		if err != nil {
			return errors.Errorf("parsing digest from path %s: %w", relPath, err)
		}

		// Check if blob already exists
		if _, err := store.Info(ctx, dgst); err == nil {
			// Blob already exists, skip
			return nil
		}

		// Read blob data
		blobData, err := os.ReadFile(path)
		if err != nil {
			return errors.Errorf("reading blob %s: %w", dgst, err)
		}

		// Write blob to content store
		writer, err := store.Writer(ctx, content.WithRef("import-"+dgst.String()), content.WithDescriptor(ocispec.Descriptor{
			Digest: dgst,
			Size:   int64(len(blobData)),
		}))
		if err != nil {
			return errors.Errorf("creating content writer for %s: %w", dgst, err)
		}

		if _, err := writer.Write(blobData); err != nil {
			writer.Close()
			return errors.Errorf("writing blob data for %s: %w", dgst, err)
		}

		if err := writer.Commit(ctx, int64(len(blobData)), dgst); err != nil {
			writer.Close()
			return errors.Errorf("committing blob %s: %w", dgst, err)
		}

		writer.Close()
		return nil
	})
}

// createImageRecord creates an image record in containerd's metadata store
func (i *ContainerdTestdataImporter) createImageRecord(ctx context.Context, imageRef string, manifest ocispec.Descriptor) error {
	imageStore := i.client.ImageService()

	// Create the image record
	img := images.Image{
		Name:   imageRef,
		Target: manifest,
	}

	if _, err := imageStore.Create(ctx, img); err != nil {
		// If image already exists, update it
		if _, updateErr := imageStore.Update(ctx, img); updateErr != nil {
			return errors.Errorf("creating/updating image record: %w", err)
		}
	}

	return nil
}

// Close closes the containerd client
func (i *ContainerdTestdataImporter) Close() error {
	return i.client.Close()
}

// IsImageAvailable checks if an image is available in containerd
func (i *ContainerdTestdataImporter) IsImageAvailable(ctx context.Context, imageRef string) (bool, error) {
	ctx = namespaces.WithNamespace(ctx, i.namespace)

	_, err := i.client.ImageService().Get(ctx, imageRef)
	if err != nil {
		return false, nil
	}
	return true, nil
}
