package source

import (
	"context"
	"fmt"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"

	"github.com/ContainerHive/ContainerHive/pkg/model"
)

// maxRegistryTags caps how many tags Fetch accepts from a single
// repository, so an unexpectedly huge mirror (e.g. "library/*") can't blow
// up memory during discovery.
const maxRegistryTags = 10000

// RegistrySource lists tags of an OCI repository via go-containerregistry's
// remote.List, rather than the Docker Hub v2 API directly. This reuses an
// existing direct dependency, inherits credentials from the same docker
// config `ch login` already writes (authn.DefaultKeychain), and works for
// any registry - GHCR, ECR, a private mirror - not only Docker Hub. Tags
// carry no source metadata, so Extra is always empty.
type RegistrySource struct {
	// listTags is overridden by tests to avoid a real network call.
	listTags func(ctx context.Context, image string) ([]string, error)
}

func (RegistrySource) Name() string { return "registry" }

func (RegistrySource) Validate(cfg *model.SourceConfig) error {
	if cfg.Image == "" {
		return fmt.Errorf("source type %q requires image", cfg.Type)
	}
	if cfg.URL != "" || cfg.Transform != "" || cfg.Repo != "" || cfg.Kind != "" {
		return fmt.Errorf("source type %q does not accept url/transform/repo/kind", cfg.Type)
	}
	return nil
}

func (RegistrySource) Descriptor(cfg *model.SourceConfig) string {
	return "registry (" + cfg.Image + ")"
}

func (RegistrySource) CacheKeyParts(cfg *model.SourceConfig) []string {
	return []string{"registry", cfg.Image}
}

func (r RegistrySource) Fetch(ctx context.Context, cfg *model.SourceConfig) ([]Version, error) {
	list := r.listTags
	if list == nil {
		list = remoteListTags
	}
	tags, err := list(ctx, cfg.Image)
	if err != nil {
		return nil, fmt.Errorf("registry source %q: %w", cfg.Image, err)
	}
	if len(tags) > maxRegistryTags {
		return nil, fmt.Errorf("registry source %q returned %d tags, exceeding the %d limit", cfg.Image, len(tags), maxRegistryTags)
	}

	versions := make([]Version, len(tags))
	for i, tag := range tags {
		versions[i] = Version{Raw: tag}
	}
	return versions, nil
}

func remoteListTags(ctx context.Context, image string) ([]string, error) {
	repo, err := name.NewRepository(image)
	if err != nil {
		return nil, fmt.Errorf("invalid repository %q: %w", image, err)
	}
	return remote.List(repo, remote.WithContext(ctx), remote.WithAuthFromKeychain(authn.DefaultKeychain))
}
