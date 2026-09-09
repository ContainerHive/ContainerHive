package tagrange

import (
	"context"
	"fmt"
	"os"

	"github.com/ContainerHive/ContainerHive/internal/tagrange/source"
	"github.com/ContainerHive/ContainerHive/pkg/model"
)

// Resolver resolves tag_ranges for a whole project. One Resolver owns one
// Cache and one source Registry, so every range in a discovery run shares
// the same singleflight group and in-memory cache layer.
type Resolver struct {
	cache    *Cache
	registry *source.Registry
}

// NewResolver constructs a Resolver. registry defaults to
// source.DefaultRegistry() when nil, which is the only case tests need to
// override (with a fixture source, so no real network call is made).
func NewResolver(cache *Cache, registry *source.Registry) *Resolver {
	if registry == nil {
		registry = source.DefaultRegistry()
	}
	return &Resolver{cache: cache, registry: registry}
}

// ResolveImage resolves every tag_range on one image into GeneratedTags.
// rangeLabel (typically the image name) is used only in error messages.
func (r *Resolver) ResolveImage(ctx context.Context, rangeLabel string, ranges []*model.TagRange) ([]GeneratedTag, error) {
	var all []GeneratedTag
	for i, tr := range ranges {
		tags, err := r.resolveOne(ctx, fmt.Sprintf("%s[%d]", rangeLabel, i), tr)
		if err != nil {
			return nil, err
		}
		all = append(all, tags...)
	}
	return all, nil
}

func (r *Resolver) resolveOne(ctx context.Context, rangeLabel string, tr *model.TagRange) ([]GeneratedTag, error) {
	cfg, sel, filter, err := r.effectiveConfig(tr)
	if err != nil {
		return nil, fmt.Errorf("tag_range %q: %w", rangeLabel, err)
	}

	src, err := r.registry.Get(cfg.Type)
	if err != nil {
		return nil, fmt.Errorf("tag_range %q: %w", rangeLabel, err)
	}
	if err := src.Validate(cfg); err != nil {
		return nil, fmt.Errorf("tag_range %q: %w", rangeLabel, err)
	}

	fingerprint := TokenFingerprint(os.Getenv(cfg.TokenEnv))
	raw, err := r.cache.FetchVersions(ctx, src, cfg, fingerprint)
	if err != nil {
		return nil, fmt.Errorf("tag_range %q: %w", rangeLabel, err)
	}

	rawVersions := make([]RawVersion, len(raw))
	for i, v := range raw {
		rawVersions[i] = RawVersion{Raw: v.Raw, Prerelease: v.Prerelease, Extra: v.Extra}
	}

	effective := &model.TagRange{
		TagName:   tr.TagName,
		Select:    sel,
		Filter:    filter,
		MaxTags:   tr.MaxTags,
		Versions:  tr.Versions,
		BuildArgs: tr.BuildArgs,
		Labels:    tr.Labels,
	}
	return GenerateTags(rangeLabel, effective, rawVersions)
}

// effectiveConfig normalizes a TagRange into a concrete SourceConfig plus
// select/filter: either tr.Source is used as-is, or tr.Generator is
// expanded into one. Exactly one of Source/Generator must be set, and
// select is required for an explicit source (an implicit default could
// silently emit hundreds of tags).
func (r *Resolver) effectiveConfig(tr *model.TagRange) (*model.SourceConfig, *model.SelectConfig, *model.FilterConfig, error) {
	hasSource := tr.Source != nil
	hasGenerator := tr.Generator != ""
	switch {
	case hasSource && hasGenerator:
		return nil, nil, nil, fmt.Errorf("source and generator are mutually exclusive")
	case !hasSource && !hasGenerator:
		return nil, nil, nil, fmt.Errorf("exactly one of source or generator is required")
	case hasGenerator:
		cfg, sel, filter, err := expandGenerator(tr.Generator, tr.Params)
		if err != nil {
			return nil, nil, nil, err
		}
		if tr.Select != nil {
			sel = tr.Select
		}
		if tr.Filter != nil {
			filter = tr.Filter
		}
		return cfg, sel, filter, nil
	default:
		if tr.Select == nil {
			return nil, nil, nil, fmt.Errorf("select is required when source is set")
		}
		return tr.Source, tr.Select, tr.Filter, nil
	}
}
