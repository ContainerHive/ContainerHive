package gcr

import (
	"errors"
	"fmt"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/partial"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

// Retag creates an additional tag alias for an existing image in a registry.
// It copies the manifest from sourceRef to targetRef without re-uploading layers.
func Retag(sourceRef, targetRef string) error {
	src, err := name.ParseReference(sourceRef)
	if err != nil {
		return errors.Join(fmt.Errorf("invalid source reference %q", sourceRef), err)
	}

	dst, err := name.NewTag(targetRef)
	if err != nil {
		return errors.Join(fmt.Errorf("invalid target reference %q", targetRef), err)
	}

	desc, err := remote.Get(src)
	if err != nil {
		return errors.Join(fmt.Errorf("failed to fetch %q", sourceRef), err)
	}

	return remote.Tag(dst, desc)
}

// PlatformImage pairs a v1.Image (loaded from a local OCI tar) with its
// target platform string.
type PlatformImage struct {
	Image    v1.Image
	Platform string // e.g. "linux/amd64"
}

// CreateManifestList builds an OCI image index (manifest list) from the given
// platform images and pushes it to targetRef. It uses mutate.AppendManifests
// and remote.WriteIndex (same approach as crane index append), which pushes
// all child image manifests and layers before pushing the index itself.
func CreateManifestList(targetRef string, images []PlatformImage) error {
	dst, err := name.NewTag(targetRef)
	if err != nil {
		return fmt.Errorf("invalid target reference %q: %w", targetRef, err)
	}

	var adds []mutate.IndexAddendum
	for _, pi := range images {
		plat, err := v1.ParsePlatform(pi.Platform)
		if err != nil {
			return fmt.Errorf("invalid platform %q: %w", pi.Platform, err)
		}

		desc, err := partial.Descriptor(pi.Image)
		if err != nil {
			return fmt.Errorf("failed to get descriptor for %q image: %w", pi.Platform, err)
		}
		desc.Platform = plat

		adds = append(adds, mutate.IndexAddendum{
			Add:        pi.Image,
			Descriptor: *desc,
		})
	}

	idx := mutate.AppendManifests(empty.Index, adds...)
	return remote.WriteIndex(dst, idx)
}

// PlatformRef pairs a registry reference with its target platform.
type PlatformRef struct {
	Ref      string // e.g. "registry.example.com/app:v1.linux-amd64.42"
	Platform string // e.g. "linux/amd64"
}

// CreateManifestListFromRefs builds an OCI image index from images that already
// exist in the registry. It fetches only the descriptors (no layers) and creates
// a manifest list pointing to them.
func CreateManifestListFromRefs(targetRef string, refs []PlatformRef) error {
	dst, err := name.NewTag(targetRef)
	if err != nil {
		return fmt.Errorf("invalid target reference %q: %w", targetRef, err)
	}

	var adds []mutate.IndexAddendum
	for _, pr := range refs {
		plat, err := v1.ParsePlatform(pr.Platform)
		if err != nil {
			return fmt.Errorf("invalid platform %q: %w", pr.Platform, err)
		}

		src, err := name.ParseReference(pr.Ref)
		if err != nil {
			return fmt.Errorf("invalid reference %q: %w", pr.Ref, err)
		}

		img, err := remote.Image(src)
		if err != nil {
			return fmt.Errorf("failed to fetch %q: %w", pr.Ref, err)
		}

		desc, err := partial.Descriptor(img)
		if err != nil {
			return fmt.Errorf("failed to get descriptor for %q: %w", pr.Ref, err)
		}
		desc.Platform = plat

		adds = append(adds, mutate.IndexAddendum{
			Add:        img,
			Descriptor: *desc,
		})
	}

	idx := mutate.AppendManifests(empty.Index, adds...)
	return remote.WriteIndex(dst, idx)
}
