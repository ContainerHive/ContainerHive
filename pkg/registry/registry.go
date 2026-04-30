package registry

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/ContainerHive/ContainerHive/internal/gcr"
	"github.com/ContainerHive/ContainerHive/internal/ocistore"
	internalregistry "github.com/ContainerHive/ContainerHive/internal/registry"
	"github.com/ContainerHive/ContainerHive/pkg/build"
	"github.com/ContainerHive/ContainerHive/pkg/model"
	"github.com/ContainerHive/ContainerHive/pkg/platform"
	"github.com/ContainerHive/ContainerHive/pkg/rendering"
)

// Registry wraps the internal registry and adds alias retagging.
type Registry struct {
	inner internalregistry.Registry
}

// NewRegistry creates a Registry based on the environment (local zot or remote).
// The dataDir parameter sets persistent storage for the local registry;
// if empty, a temporary directory is used.
// The registryConfig is read from hive.yml and provides the registry address for CI pushes.
func NewRegistry(dataDir string, registryConfig *model.RegistryConfig) (*Registry, error) {
	inner, err := internalregistry.NewRegistry(dataDir, registryConfig)
	if err != nil {
		return nil, err
	}
	return &Registry{inner: inner}, nil
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

// UseDockerMediaTypes reports whether image manifests and the manifest list
// should be emitted in Docker-scheme media types rather than OCI. Docker Hub's
// frontend rejects pure OCI indexes, so pushes to docker.io (or a configured
// mirror marked as Docker-scheme via hive.yml) need this flag.
func (r *Registry) UseDockerMediaTypes() bool {
	return r.inner.UseDockerMediaTypes()
}

// Push pushes an OCI tar to the registry.
func (r *Registry) Push(ctx context.Context, imageName, tag, ociTarPath string) error {
	return r.inner.Push(ctx, imageName, tag, ociTarPath)
}

// ImageRef returns the full registry reference for a platform-specific image.
// Format: address/imageName:tag.sanitized-platform[.buildID]
func (r *Registry) ImageRef(imageName, tag, platformStr, buildID string) string {
	return fmt.Sprintf("%s/%s:%s", r.Address(), imageName, pushTag(tag, platformStr, buildID))
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

// loadImageFromTar extracts an OCI tar and returns the first image from the
// layout. The caller must call the returned cleanup function after the image
// is no longer needed (v1.Image reads blobs lazily from disk).
func loadImageFromTar(ociTarPath string) (*ocistore.OCIImage, error) {
	return ocistore.ImageFromTar(ociTarPath)
}

// pushTag returns the platform-specific tag as used by ch build when pushing.
// Format: tag.sanitized-platform[.buildID]
func pushTag(tag, platformStr, buildID string) string {
	return build.PushTag(tag, platformStr, buildID)
}

// createManifestForTag creates an OCI image index (manifest list) for a single
// tag. When the registry is remote, it references platform images already in
// the registry (no download needed). When local, it loads from OCI tars.
func (r *Registry) createManifestForTag(imageName, tag string, platforms []string, buildID, distPath string) error {
	manifestTag := build.WithBuildID(tag, buildID)
	targetRef := fmt.Sprintf("%s/%s:%s", r.Address(), imageName, manifestTag)

	slog.Info("Creating manifest", "image", imageName, "tag", manifestTag, "platforms", len(platforms))

	if !r.IsLocal() {
		// Remote: reference images already in the registry by their push tags
		var refs []gcr.PlatformRef
		for _, p := range platforms {
			refs = append(refs, gcr.PlatformRef{
				Ref:      fmt.Sprintf("%s/%s:%s", r.Address(), imageName, pushTag(tag, p, buildID)),
				Platform: p,
			})
		}
		return gcr.CreateManifestListFromRefs(targetRef, refs, r.UseDockerMediaTypes())
	}

	// Local: load from OCI tars
	var images []gcr.PlatformImage
	var cleanups []func()
	defer func() {
		for _, fn := range cleanups {
			fn()
		}
	}()

	for _, p := range platforms {
		tarPath := build.TarFilePath(distPath, imageName, tag, p)
		ociImage, err := loadImageFromTar(tarPath)
		if err != nil {
			return fmt.Errorf("failed to load image for %s:%s [%s]: %w", imageName, tag, p, err)
		}
		cleanups = append(cleanups, ociImage.Cleanup)
		images = append(images, gcr.PlatformImage{
			Image:    ociImage.Image,
			Platform: p,
		})
	}

	return gcr.CreateManifestList(targetRef, images, r.UseDockerMediaTypes())
}

// CreateAllManifests creates multi-arch manifest lists for all tags of all
// images matching the filters. Each manifest combines the platform-specific
// images whose OCI tars are in distPath. Descriptor info is read from local
// tars to avoid registry compatibility issues with OCI manifest GET.
func (r *Registry) CreateAllManifests(project *model.ContainerHiveProject, filters []build.Filter, buildID, distPath string) error {
	for _, img := range project.ImagesByIdentifier {
		if !matchesImageFilter(filters, img.Name) {
			continue
		}

		for tagName := range img.Tags {
			if matchesTagFilter(filters, img.Name, tagName) {
				platforms := platform.Resolve(project.Config.Platforms, img.Platforms, nil)
				if err := r.createManifestForTag(img.Name, tagName, platforms, buildID, distPath); err != nil {
					return fmt.Errorf("failed to create manifest for %s:%s: %w", img.Name, tagName, err)
				}
			}

			for _, variantDef := range img.Variants {
				variantTag := tagName + variantDef.TagSuffix
				if !matchesTagFilter(filters, img.Name, variantTag) {
					continue
				}
				platforms := platform.Resolve(project.Config.Platforms, img.Platforms, variantDef.Platforms)
				if err := r.createManifestForTag(img.Name, variantTag, platforms, buildID, distPath); err != nil {
					return fmt.Errorf("failed to create manifest for %s:%s: %w", img.Name, variantTag, err)
				}
			}
		}
	}
	return nil
}

// collectBaseTags returns the base tag names for an image, excluding variant suffixes.
func collectBaseTags(imageDef *model.Image) []string {
	tags := make([]string, 0, len(imageDef.Tags))
	for tagName := range imageDef.Tags {
		tags = append(tags, tagName)
	}
	return tags
}

// retagAliases creates semantic version tag aliases in the registry for a
// single image. Aliases are retagged from the multi-arch manifest (without
// platform suffix). If buildID is set, it is appended to match pushed tags.
// Only tags matching the filters are retagged.
func (r *Registry) retagAliases(imageDef *model.Image, filters []build.Filter, buildID string) error {
	allTags := collectAllTags(imageDef)
	aliases := rendering.ResolveAliases(allTags)

	if imageDef.LatestAlias != nil {
		latestTarget, err := rendering.ResolveLatestAlias(collectBaseTags(imageDef), imageDef.LatestAlias.Tag)
		if err != nil {
			switch imageDef.LatestAlias.OnMissing {
			case "silent":
				// do nothing
			case "warning":
				slog.Warn("Latest alias resolution failed", "error", err)
			default: // "error" or unset
				return err
			}
		} else {
			aliases[imageDef.LatestAlias.Tag] = latestTarget
		}

		for _, variantDef := range imageDef.Variants {
			variantTags := make([]string, 0, len(imageDef.Tags))
			for tagName := range imageDef.Tags {
				variantTags = append(variantTags, tagName+variantDef.TagSuffix)
			}
			variantTarget, err := rendering.ResolveLatestAlias(variantTags, imageDef.LatestAlias.Tag)
			if err != nil {
				switch imageDef.LatestAlias.OnMissing {
				case "silent":
					// do nothing
				case "warning":
					slog.Warn("Latest alias resolution failed for variant", "variant", variantDef.Name, "error", err)
				default: // "error" or unset
					return err
				}
			} else {
				aliases[imageDef.LatestAlias.Tag+variantDef.TagSuffix] = variantTarget
			}
		}
	}

	for alias, tag := range aliases {
		if !matchesTagFilter(filters, imageDef.Name, tag) {
			continue
		}
		sourceTag := build.WithBuildID(tag, buildID)
		targetTag := build.WithBuildID(alias, buildID)
		sourceRef := fmt.Sprintf("%s/%s:%s", r.Address(), imageDef.Name, sourceTag)
		targetRef := fmt.Sprintf("%s/%s:%s", r.Address(), imageDef.Name, targetTag)
		slog.Info("Tagging alias", "image", imageDef.Name, "target", targetTag, "source", sourceTag)
		if err := gcr.Retag(sourceRef, targetRef); err != nil {
			slog.Warn("Failed to retag", "source", sourceRef, "target", targetRef, "error", err)
		}
	}
	return nil
}

// RetagAllAliases retags aliases for all images in the project. Aliases point
// to multi-arch manifests (created by CreateAllManifests), not to
// platform-specific images. If filters is non-empty, only images matching at
// least one filter are processed. If buildID is set, tags are suffixed with
// .<buildID> to match pushed tags.
func (r *Registry) RetagAllAliases(project *model.ContainerHiveProject, filters []build.Filter, buildID string) error {
	for _, img := range project.ImagesByIdentifier {
		if !matchesImageFilter(filters, img.Name) {
			continue
		}
		if err := r.retagAliases(img, filters, buildID); err != nil {
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

// matchesTagFilter returns true if the image:tag matches at least one filter,
// or if the filter list is empty (match all). Variant tags are matched by
// checking if the tag starts with the filter's tag name.
func matchesTagFilter(filters []build.Filter, imageName, tagName string) bool {
	if len(filters) == 0 {
		return true
	}
	for _, f := range filters {
		if f.ImageName != "" && f.ImageName != imageName {
			continue
		}
		if f.TagName != "" && f.TagName != tagName {
			continue
		}
		return true
	}
	return false
}
