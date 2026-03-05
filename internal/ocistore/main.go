package ocistore

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/containerd/containerd/v2/core/content"
	contentlocal "github.com/containerd/containerd/v2/plugins/content/local"
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
