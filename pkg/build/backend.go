package build

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

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
	if backend == model.BuildBackendChisel {
		return model.ChiselFileName
	}
	return "Dockerfile"
}

// hashSource returns the content bytes driving the cache scope hash. For Nix
// flakes, flake.lock pins the flake resolution and belongs into the hash.
// For chisel cut specs, the custom slices/ directory is part of the build
// input and belongs into the hash. Dockerfile builds ignore both a stray
// flake.lock and a stray slices dir, so the source reduces to the file
// content alone.
func hashSource(entrypointPath string) ([]byte, error) {
	content, err := os.ReadFile(entrypointPath)
	if err != nil {
		return nil, err
	}

	switch filepath.Base(entrypointPath) {
	case model.ChiselFileName:
		return append(content, chiselSlicesHash(filepath.Join(filepath.Dir(entrypointPath), "slices"))...), nil
	case model.FlakeFileName:
		lock, err := os.ReadFile(filepath.Join(filepath.Dir(entrypointPath), "flake.lock"))
		if errors.Is(err, os.ErrNotExist) {
			return content, nil
		}
		if err != nil {
			return nil, err
		}
		return append(content, lock...), nil
	default:
		return content, nil
	}
}

// chiselSlicesHash returns the content bytes of the slices/ directory next to
// a chisel cut spec, sorted by path for a stable hash. A missing or empty
// directory returns nil.
func chiselSlicesHash(slicesDir string) []byte {
	files, err := slicesDirFiles(slicesDir)
	if err != nil {
		return nil
	}

	var hash []byte
	for _, path := range files {
		content, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		hash = append(hash, content...)
	}
	return hash
}

// slicesDirFiles lists all files below a slices/ directory, sorted by path.
func slicesDirFiles(slicesDir string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(slicesDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}

// nixConfigFor merges a variant Nix config over the image level config.
func nixConfigFor(imageDef *model.Image, variantDef *model.ImageVariant) model.NixConfig {
	if variantDef != nil && variantDef.Nix != nil {
		return *variantDef.Nix
	}
	return imageDef.Nix
}

// chiselConfigFor merges a variant Chisel config over the image level config.
func chiselConfigFor(imageDef *model.Image, variantDef *model.ImageVariant) model.ChiselConfig {
	if variantDef != nil && variantDef.Chisel != nil {
		return *variantDef.Chisel
	}
	return imageDef.Chisel
}

// runConfigFor merges a variant run config over the image level config.
func runConfigFor(imageDef *model.Image, variantDef *model.ImageVariant) *model.ImageRunConfig {
	if variantDef != nil && variantDef.RunConfig != nil {
		return variantDef.RunConfig
	}
	return imageDef.RunConfig
}

// expandParentPlaceholder expands the __hive_parent__ placeholder of a
// variant build into the image name and tag of the variant. The image level
// build has no parent tag to resolve, so the placeholder errors there.
func expandParentPlaceholder(parent, imageName, tagName string) (string, error) {
	if parent != model.HiveParentPlaceholder {
		return parent, nil
	}
	if tagName != "" {
		return fmt.Sprintf("__hive__/%s:%s", imageName, tagName), nil
	}
	return "", fmt.Errorf("image %q cannot use the %s placeholder outside a variant: a build without a variant has no parent tag", imageName, model.HiveParentPlaceholder)
}

// resolveParentImage resolves a backend parent image reference for a build.
// An empty ref returns nil.
func resolveParentImage(parentRef string, opts *ProjectBuildOpts, platformStr string) (*ParentDeps, error) {
	if parentRef == "" {
		return nil, nil
	}
	return ResolveParentImage(parentRef, HiveDepsOpts{
		DistPath:        opts.DistPath,
		PlatformStr:     platformStr,
		RegistryAddress: opts.registryAddress(),
		BuildID:         opts.BuildID,
		Project:         opts.Project,
	})
}

// nixParentForBuild expands the parent placeholder of a Nix build.
func nixParentForBuild(config model.NixConfig, imageDef *model.Image, tagName string) (model.NixConfig, error) {
	parent, err := expandParentPlaceholder(config.Parent, imageDef.Name, tagName)
	if err != nil {
		return model.NixConfig{}, err
	}
	config.Parent = parent
	return config, nil
}

// chiselParentForBuild expands the parent placeholder of a Chisel build.
func chiselParentForBuild(config model.ChiselConfig, imageDef *model.Image, tagName string) (model.ChiselConfig, error) {
	parent, err := expandParentPlaceholder(config.Parent, imageDef.Name, tagName)
	if err != nil {
		return model.ChiselConfig{}, err
	}
	config.Parent = parent
	return config, nil
}

// nixBuildContext assembles the NixBuildContext for the backend from a
// resolved Nix config, run config and parent source.
func nixBuildContext(root string, nix model.NixConfig, run *model.ImageRunConfig, parent *build_context.ParentImage) *build_context.NixBuildContext {
	buildCtx := &build_context.NixBuildContext{
		Root:         root,
		BuilderImage: nix.BuilderImage,
		ExtraArgs:    nix.ExtraArgs,
		Parent:       parent,
		StoreCache:   nix.StoreCacheEnabled(),
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

// chiselBuildContext assembles the ChiselBuildContext for the backend from a
// parsed cut spec, a resolved Chisel config, a run config and a parent
// source. The cut spec comes from the rendered chisel.yaml of the tag or
// variant directory.
func chiselBuildContext(root string, spec *model.ChiselCutSpec, chisel model.ChiselConfig, run *model.ImageRunConfig, parent *build_context.ParentImage) (*build_context.ChiselBuildContext, error) {
	buildCtx := &build_context.ChiselBuildContext{
		Root: root,
		Release: build_context.ChiselRelease{
			Name:   spec.Release.Name,
			Git:    spec.Release.Git,
			GitRef: spec.Release.GitRef,
		},
		Slices:        spec.Slices,
		Ignore:        spec.Ignore,
		BuilderImage:  chisel.BuilderImage,
		ChiselVersion: chisel.ChiselVersion,
		ExtraArgs:     chisel.ExtraArgs,
		Parent:        parent,
		Cache:         chisel.CacheEnabled(),
	}
	if run != nil {
		buildCtx.Entrypoint = run.Entrypoint
		buildCtx.Cmd = run.Cmd
		buildCtx.Env = run.Env
		buildCtx.User = run.User
		buildCtx.WorkingDir = run.WorkingDir
	}
	if err := buildCtx.Validate(); err != nil {
		return nil, err
	}
	return buildCtx, nil
}

// parseChiselCutSpec parses the rendered chisel.yaml of a tag or variant
// directory into the build input fields.
func parseChiselCutSpec(root string) (*model.ChiselCutSpec, error) {
	data, err := os.ReadFile(filepath.Join(root, model.ChiselFileName))
	if err != nil {
		return nil, fmt.Errorf("failed to read %s in %s: %w", model.ChiselFileName, root, err)
	}
	spec, err := model.ParseChiselCutSpec(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse chisel cut spec: %w", err)
	}
	return spec, nil
}
