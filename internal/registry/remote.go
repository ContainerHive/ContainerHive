package registry

import (
	"context"
	"errors"
	"os"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/layout"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/timo-reymann/ContainerHive/internal/utils"
)

// RemoteRegistry is a passthrough registry for CI environments.
// Start and Stop are no-ops; Push pushes to the configured remote registry.
type RemoteRegistry struct {
	address          string
	dockerMediaTypes *bool // explicit override; nil = auto-detect
}

// NewRemoteRegistry creates a remote registry with the given address. Media
// types default to auto-detection (Docker Hub → Docker-scheme, everything
// else → OCI). Callers wanting an explicit override should use
// NewRemoteRegistryWithMediaTypes.
func NewRemoteRegistry(address string) *RemoteRegistry {
	return &RemoteRegistry{address: address}
}

// NewRemoteRegistryWithMediaTypes creates a remote registry with an explicit
// Docker-media-types override. Pass nil to fall back to auto-detection, a
// non-nil value to force the choice.
func NewRemoteRegistryWithMediaTypes(address string, dockerMediaTypes *bool) *RemoteRegistry {
	return &RemoteRegistry{address: address, dockerMediaTypes: dockerMediaTypes}
}

func (r *RemoteRegistry) Start(_ context.Context) error {
	return nil
}

func (r *RemoteRegistry) Stop(_ context.Context) error {
	return nil
}

func (r *RemoteRegistry) Address() string {
	return r.address
}

func (r *RemoteRegistry) IsLocal() bool {
	return false
}

// UseDockerMediaTypes reports whether image manifests and the manifest list
// should use Docker-scheme media types. An explicit override from the registry
// config wins; otherwise the address is auto-detected.
func (r *RemoteRegistry) UseDockerMediaTypes() bool {
	return resolveDockerMediaTypes(r.address, r.dockerMediaTypes)
}

func (r *RemoteRegistry) Push(_ context.Context, imageName, tag, ociTarPath string) error {
	tmpDir, err := os.MkdirTemp("", "oci-push-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	if err := utils.ExtractTar(ociTarPath, tmpDir); err != nil {
		return errors.Join(errors.New("failed to extract OCI tar for push"), err)
	}

	layoutPath, err := layout.FromPath(tmpDir)
	if err != nil {
		return errors.Join(errors.New("failed to read OCI layout"), err)
	}

	idx, err := layoutPath.ImageIndex()
	if err != nil {
		return err
	}

	idxManifest, err := idx.IndexManifest()
	if err != nil {
		return err
	}

	if len(idxManifest.Manifests) == 0 {
		return errors.New("no manifests in OCI layout")
	}

	img, err := layoutPath.Image(idxManifest.Manifests[0].Digest)
	if err != nil {
		return errors.Join(errors.New("failed to read image from layout"), err)
	}

	ref, err := name.NewTag(r.address + "/" + imageName + ":" + tag)
	if err != nil {
		return errors.Join(errors.New("invalid image reference"), err)
	}

	if err := remote.Write(ref, img); err != nil {
		return errors.Join(errors.New("failed to push image to remote registry"), err)
	}

	return nil
}
