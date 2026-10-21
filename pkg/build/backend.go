package build

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/ContainerHive/ContainerHive/internal/buildkit/build_context"
	"github.com/ContainerHive/ContainerHive/pkg/model"
)

// variantBackend returns the build backend for a variant, falling back to the
// image backend.
func variantBackend(imageDef *model.Image, variantDef *model.ImageVariant) string {
	if variantDef.BuildBackend != "" {
		return variantDef.BuildBackend
	}
	return imageDef.BuildBackend
}

// entrypointFileName returns the build entrypoint file name inside the
// rendered tag or variant directory for a build backend.
func entrypointFileName(backend string) string {
	if backend == model.BuildBackendNix {
		return model.FlakeFileName
	}
	return "Dockerfile"
}

// hashSource returns the content bytes driving the cache scope hash. For Nix
// flakes, flake.lock pins the flake resolution and belongs into the hash.
// Dockerfile builds have no flake.lock, so the source reduces to the file
// content alone.
func hashSource(entrypointPath string) ([]byte, error) {
	content, err := os.ReadFile(entrypointPath)
	if err != nil {
		return nil, err
	}

	lock, err := os.ReadFile(filepath.Join(filepath.Dir(entrypointPath), "flake.lock"))
	if errors.Is(err, os.ErrNotExist) {
		return content, nil
	}
	if err != nil {
		return nil, err
	}
	return append(content, lock...), nil
}

// nixConfigFor merges a variant Nix config over the image level config.
func nixConfigFor(imageDef *model.Image, variantDef *model.ImageVariant) model.NixConfig {
	if variantDef != nil && variantDef.Nix != nil {
		return *variantDef.Nix
	}
	return imageDef.Nix
}

// runConfigFor merges a variant run config over the image level config.
func runConfigFor(imageDef *model.Image, variantDef *model.ImageVariant) *model.ImageRunConfig {
	if variantDef != nil && variantDef.RunConfig != nil {
		return variantDef.RunConfig
	}
	return imageDef.RunConfig
}

// nixBuildContext assembles the NixBuildContext for the backend from a
// resolved Nix config and run config.
func nixBuildContext(root string, nix model.NixConfig, run *model.ImageRunConfig) *build_context.NixBuildContext {
	buildCtx := &build_context.NixBuildContext{
		Root:         root,
		BuilderImage: nix.BuilderImage,
		ExtraArgs:    nix.ExtraArgs,
	}
	if run != nil {
		buildCtx.Entrypoint = run.Entrypoint
		buildCtx.Cmd = run.Cmd
		buildCtx.Env = run.Env
		buildCtx.User = run.User
		buildCtx.WorkingDir = run.WorkingDir
	}
	return buildCtx
}
