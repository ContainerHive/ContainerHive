// Package source implements VersionSource, the pluggable way tag_ranges
// fetches candidate versions from external registries and APIs.
package source

import (
	"context"
	"fmt"

	"github.com/ContainerHive/ContainerHive/pkg/model"
)

// Version is one candidate version reported by a source, before parsing.
type Version struct {
	// Raw is the version exactly as the source reported it.
	Raw string

	// Prerelease is true when the source itself knows the version is a
	// prerelease (e.g. a GitHub release with prerelease: true). A tag
	// string alone can't answer this - internal/semantic_tags treats any
	// "-suffix" as an opaque flavour, not a prerelease.
	Prerelease bool

	// Extra carries source metadata, exposed to templates as .extra.<key>.
	Extra map[string]string
}

// VersionSource fetches candidate versions for one source configuration.
// Implementations must be safe for concurrent use and must not cache -
// caching, request de-duplication and retries are the caller's job.
type VersionSource interface {
	// Name returns the value used in source.type.
	Name() string

	// Validate reports configuration problems without performing any I/O.
	// It must reject fields that belong to a different source type.
	Validate(cfg *model.SourceConfig) error

	// Descriptor returns a short, secret-free description of the query,
	// used in cache entries and error messages.
	Descriptor(cfg *model.SourceConfig) string

	// CacheKeyParts returns the secret-free values that identify this
	// query, for the caller to hash into a cache key.
	CacheKeyParts(cfg *model.SourceConfig) []string

	// Fetch retrieves every candidate version for cfg.
	Fetch(ctx context.Context, cfg *model.SourceConfig) ([]Version, error)
}

// Registry resolves a source type to its implementation. The zero value is
// not usable; construct one with DefaultRegistry or NewRegistry.
type Registry struct {
	sources map[string]VersionSource
}

// NewRegistry builds a registry from the given sources, keyed by each
// source's own Name(). Tests use this to inject a fixture source instead of
// making real network calls.
func NewRegistry(sources ...VersionSource) *Registry {
	r := &Registry{sources: make(map[string]VersionSource, len(sources))}
	for _, s := range sources {
		r.sources[s.Name()] = s
	}
	return r
}

// DefaultRegistry returns the registry of built-in sources: json, registry
// (aliased as dockerhub), and github.
func DefaultRegistry() *Registry {
	registrySrc := &RegistrySource{}
	return NewRegistry(
		&JSONSource{},
		registrySrc,
		dockerHubAlias{RegistrySource: registrySrc},
		&GitHubSource{},
	)
}

// Get returns the source registered for the given source.type.
func (r *Registry) Get(sourceType string) (VersionSource, error) {
	s, ok := r.sources[sourceType]
	if !ok {
		return nil, fmt.Errorf("unknown source type %q (known: %v)", sourceType, r.knownTypes())
	}
	return s, nil
}

func (r *Registry) knownTypes() []string {
	types := make([]string, 0, len(r.sources))
	for t := range r.sources {
		types = append(types, t)
	}
	return types
}

// dockerHubAlias registers RegistrySource a second time under the name
// "dockerhub", matching the issue's own spelling, while the honest name
// "registry" documents that it works for any OCI registry, not only
// Docker Hub.
type dockerHubAlias struct {
	*RegistrySource
}

func (dockerHubAlias) Name() string { return "dockerhub" }
