package registry

import (
	"context"
	"fmt"
	"log"

	"github.com/timo-reymann/ContainerHive/internal/gcr"
	internalregistry "github.com/timo-reymann/ContainerHive/internal/registry"
	"github.com/timo-reymann/ContainerHive/pkg/build"
	"github.com/timo-reymann/ContainerHive/pkg/model"
	"github.com/timo-reymann/ContainerHive/pkg/rendering"
)

// Registry wraps the internal registry and adds alias retagging.
type Registry struct {
	inner internalregistry.Registry
}

// NewRegistry creates a Registry based on the environment (local zot or remote).
func NewRegistry() *Registry {
	return &Registry{inner: internalregistry.NewRegistry()}
}

// Start initializes the registry.
func (r *Registry) Start(ctx context.Context) error {
	return r.inner.Start(ctx)
}

// Stop shuts down the registry.
func (r *Registry) Stop(ctx context.Context) error {
	return r.inner.Stop(ctx)
}

// Address returns the registry address (host:port).
func (r *Registry) Address() string {
	return r.inner.Address()
}

// IsLocal returns true if the registry is an embedded local instance.
func (r *Registry) IsLocal() bool {
	return r.inner.IsLocal()
}

// Push pushes an OCI tar to the registry.
func (r *Registry) Push(ctx context.Context, imageName, tag, ociTarPath string) error {
	return r.inner.Push(ctx, imageName, tag, ociTarPath)
}

// collectAllTags returns all tags for an image, including variant tags.
func collectAllTags(imageDef *model.Image) []string {
	var allTags []string
	for tagName := range imageDef.Tags {
		allTags = append(allTags, tagName)
		for _, variantDef := range imageDef.Variants {
			allTags = append(allTags, tagName+variantDef.TagSuffix)
		}
	}
	return allTags
}

// RetagAliases creates semantic version tag aliases in the registry for a
// single image. It collects all tags (including variant tags), resolves
// aliases, and retags them.
func (r *Registry) RetagAliases(imageDef *model.Image) error {
	allTags := collectAllTags(imageDef)
	aliases := rendering.ResolveAliases(allTags)

	for alias, tag := range aliases {
		sourceRef := fmt.Sprintf("%s/%s:%s", r.Address(), imageDef.Name, tag)
		targetRef := fmt.Sprintf("%s/%s:%s", r.Address(), imageDef.Name, alias)
		log.Printf("Tagging alias %s:%s -> %s:%s", imageDef.Name, alias, imageDef.Name, tag)
		if err := gcr.Retag(sourceRef, targetRef); err != nil {
			log.Printf("Warning: Failed to retag %s -> %s: %v", sourceRef, targetRef, err)
		}
	}
	return nil
}

// RetagAllAliases retags aliases for all images in the project. If filters
// is non-empty, only images matching at least one filter are processed.
func (r *Registry) RetagAllAliases(project *model.ContainerHiveProject, filters []build.Filter) error {
	for _, img := range project.ImagesByIdentifier {
		if !matchesImageFilter(filters, img.Name) {
			continue
		}
		if err := r.RetagAliases(img); err != nil {
			return err
		}
	}
	return nil
}

// matchesImageFilter returns true if the image name matches at least one
// filter, or if the filter list is empty (match all).
func matchesImageFilter(filters []build.Filter, imageName string) bool {
	if len(filters) == 0 {
		return true
	}
	for _, f := range filters {
		if f.ImageName == "" || f.ImageName == imageName {
			return true
		}
	}
	return false
}
