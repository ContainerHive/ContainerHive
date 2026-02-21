package build

import (
	"github.com/timo-reymann/ContainerHive/internal/buildconfig_resolver"
	"github.com/timo-reymann/ContainerHive/pkg/model"
)

// ResolvedConfig holds the resolved build arguments and secrets for an image build.
type ResolvedConfig struct {
	BuildArgs map[string]string
	Secrets   map[string][]byte
}

// ResolveTagConfig resolves build args and secrets for a specific image tag.
func ResolveTagConfig(image *model.Image, tag *model.Tag) (*ResolvedConfig, error) {
	resolved, err := buildconfig_resolver.ForTag(image, tag)
	if err != nil {
		return nil, err
	}
	return &ResolvedConfig{
		BuildArgs: resolved.ToBuildArgs(),
		Secrets:   resolved.Secrets,
	}, nil
}

// ResolveVariantConfig resolves build args and secrets for a specific image
// tag variant.
func ResolveVariantConfig(image *model.Image, variant *model.ImageVariant, tag *model.Tag) (*ResolvedConfig, error) {
	resolved, err := buildconfig_resolver.ForTagVariant(image, variant, tag)
	if err != nil {
		return nil, err
	}
	return &ResolvedConfig{
		BuildArgs: resolved.ToBuildArgs(),
		Secrets:   resolved.Secrets,
	}, nil
}
