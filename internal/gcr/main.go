package gcr

import (
	"errors"
	"fmt"

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

	img, err := desc.Image()
	if err != nil {
		return errors.Join(fmt.Errorf("failed to read image from %q", sourceRef), err)
	}

	if err := remote.Write(dst, img); err != nil {
		return errors.Join(fmt.Errorf("failed to write tag %q", targetRef), err)
	}

	return nil
}
