package buildconfig_resolver

import (
	"fmt"

	"github.com/ContainerHive/ContainerHive/internal/secrets"
	"github.com/ContainerHive/ContainerHive/pkg/model"
)

type ResolvedBuildValues struct {
	BuildArgs model.BuildArgs
	Versions  model.Versions
	Secrets   map[string][]byte
}

func ForTag(image *model.Image, tag *model.Tag) (*ResolvedBuildValues, error) {
	resolved := &ResolvedBuildValues{
		BuildArgs: make(model.BuildArgs),
		Versions:  make(model.Versions),
		Secrets:   make(map[string][]byte),
	}

	for k, v := range image.Versions {
		resolved.Versions[k] = v
	}

	for k, v := range tag.Versions {
		resolved.Versions[k] = v
	}

	for k, v := range tag.BuildArgs {
		resolved.BuildArgs[k] = v
	}

	for k, v := range image.BuildArgs {
		resolved.BuildArgs[k] = v
	}

	// Resolve secrets
	for k, secret := range image.Secrets {
		resolvedValue, err := secrets.Resolve(secret.SourceType, secret.Value)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve secret '%s': %w", k, err)
		}
		resolved.Secrets[k] = []byte(resolvedValue)
	}

	return resolved, nil
}

func ForTagVariant(image *model.Image, variant *model.ImageVariant, tag *model.Tag) (*ResolvedBuildValues, error) {
	resolved, err := ForTag(image, tag)
	if err != nil {
		return nil, err
	}

	for k, v := range variant.Versions {
		resolved.Versions[k] = v
	}

	for k, v := range variant.BuildArgs {
		resolved.BuildArgs[k] = v
	}

	return resolved, nil
}

func (r *ResolvedBuildValues) ToBuildArgs() model.BuildArgs {
	var buildArgs = map[string]string{}

	for k, v := range r.BuildArgs {
		buildArgs[normalizeKey(k)] = v
	}

	for k, v := range r.Versions {
		buildArgs[fmt.Sprintf("%s_VERSION", normalizeKey(k))] = v
	}

	return buildArgs
}
