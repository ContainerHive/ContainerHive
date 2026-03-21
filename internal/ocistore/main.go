package ocistore

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/containerd/containerd/v2/core/content"
	contentlocal "github.com/containerd/containerd/v2/plugins/content/local"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/layout"
	"github.com/opencontainers/go-digest"
	ocispecs "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/timo-reymann/ContainerHive/internal/utils"
)

// OCILayoutStore wraps a containerd content.Store backed by an extracted OCI
// tar layout. The Digest field holds the manifest digest from index.json, used
// to wire up BuildKit named contexts via oci-layout:<storeID>@<digest>.
type OCILayoutStore struct {
	Store  content.Store
	Digest digest.Digest
	dir    string
}

// OCIImage holds a v1.Image extracted from an OCI tar along with its
// annotations and a cleanup function. The caller must invoke Cleanup when the
// image is no longer needed (v1.Image reads blobs lazily from disk).
type OCIImage struct {
	Image       v1.Image
	Annotations map[string]string
	Cleanup     func()
}

// ImageFromTar extracts an OCI tar to a temporary directory, parses the OCI
// layout, and returns the first image from the index along with its
// annotations.
func ImageFromTar(ociTarPath string) (*OCIImage, error) {
	tmpDir, err := os.MkdirTemp("", "oci-layout-*")
	if err != nil {
		return nil, err
	}
	cleanup := func() { os.RemoveAll(tmpDir) }

	if err := utils.ExtractTar(ociTarPath, tmpDir); err != nil {
		cleanup()
		return nil, fmt.Errorf("failed to extract OCI tar: %w", err)
	}

	layoutPath, err := layout.FromPath(tmpDir)
	if err != nil {
		cleanup()
		return nil, fmt.Errorf("failed to read OCI layout: %w", err)
	}

	idx, err := layoutPath.ImageIndex()
	if err != nil {
		cleanup()
		return nil, err
	}

	idxManifest, err := idx.IndexManifest()
	if err != nil {
		cleanup()
		return nil, err
	}

	if len(idxManifest.Manifests) == 0 {
		cleanup()
		return nil, fmt.Errorf("no manifests in OCI layout")
	}

	manifest := idxManifest.Manifests[0]
	img, err := layoutPath.Image(manifest.Digest)
	if err != nil {
		cleanup()
		return nil, err
	}

	return &OCIImage{
		Image:       img,
		Annotations: manifest.Annotations,
		Cleanup:     cleanup,
	}, nil
}

// FromTar extracts an OCI image tar to destDir and returns an OCILayoutStore
// backed by the extracted blobs directory. The store can be passed directly to
// BuildKit's SolveOpt.OCIStores for named context resolution.
func FromTar(ociTarPath, destDir string) (*OCILayoutStore, error) {
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return nil, errors.Join(errors.New("failed to create OCI layout dir"), err)
	}

	if err := utils.ExtractTar(ociTarPath, destDir); err != nil {
		return nil, errors.Join(errors.New("failed to extract OCI tar"), err)
	}

	indexData, err := os.ReadFile(filepath.Join(destDir, "index.json"))
	if err != nil {
		return nil, errors.Join(errors.New("failed to read OCI index.json"), err)
	}

	var index ocispecs.Index
	if err := json.Unmarshal(indexData, &index); err != nil {
		return nil, errors.Join(errors.New("failed to parse OCI index.json"), err)
	}

	if len(index.Manifests) == 0 {
		return nil, errors.New("OCI index.json contains no manifests")
	}

	store, err := contentlocal.NewStore(destDir)
	if err != nil {
		return nil, errors.Join(errors.New("failed to create content store"), err)
	}

	return &OCILayoutStore{
		Store:  store,
		Digest: index.Manifests[0].Digest,
		dir:    destDir,
	}, nil
}
