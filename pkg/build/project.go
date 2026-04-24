package build

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/timo-reymann/ContainerHive/pkg/cache"
	"github.com/timo-reymann/ContainerHive/pkg/deps"
	"github.com/timo-reymann/ContainerHive/pkg/model"
	"github.com/timo-reymann/ContainerHive/pkg/platform"
	"github.com/timo-reymann/ContainerHive/pkg/progress"
)

// Registry provides registry metadata for direct BuildKit pushes.
type Registry interface {
	Address() string
	IsLocal() bool
	UseDockerMediaTypes() bool
}

// Filter selects a subset of images/tags to build.
// Empty fields match everything.
type Filter struct {
	ImageName string
	TagName   string
}

// ProjectBuildOpts holds shared configuration for a project-wide build.
type ProjectBuildOpts struct {
	Project     *model.ContainerHiveProject
	BuildOrder  *deps.BuildOrder
	DistPath    string
	Cache       cache.BuildkitCache
	Registry    Registry // nil when no inter-image dependencies exist
	ProgressOut io.Writer
	// ProgressConfig controls build progress display (mode, colors, no-color).
	// When zero-valued, AutoMode with DefaultColors is used.
	ProgressConfig progress.Config
	Filters        []Filter // empty = build everything
	BuildID        string   // if set, registry push/retag uses tags suffixed with .<BuildID>

	// OnBuild is called after each successful build with the image tag and tar path.
	OnBuild func(imageTag, tarFile string)
}

func (o *ProjectBuildOpts) pushTag(tagName, platformStr string) string {
	return PushTag(tagName, platformStr, o.BuildID)
}

// registryRef returns the full registry reference for a build, or empty if no
// registry is configured. Format: address/imageName:pushTag
func (o *ProjectBuildOpts) registryRef(imageName, tagName, platformStr string) string {
	if o.Registry == nil {
		return ""
	}
	return fmt.Sprintf("%s/%s:%s", o.Registry.Address(), imageName, o.pushTag(tagName, platformStr))
}

// registryAddress returns the registry address, or empty if no registry is configured.
func (o *ProjectBuildOpts) registryAddress() string {
	if o.Registry == nil {
		return ""
	}
	return o.Registry.Address()
}

// registryInsecure returns true if the registry uses HTTP (local registries).
func (o *ProjectBuildOpts) registryInsecure() bool {
	return o.Registry != nil && o.Registry.IsLocal()
}

// useDockerMediaTypes returns true if BuildKit's image exporter should emit
// Docker-scheme media types for the target registry.
func (o *ProjectBuildOpts) useDockerMediaTypes() bool {
	return o.Registry != nil && o.Registry.UseDockerMediaTypes()
}

// matchesFilters checks whether a tag should be built.
// Matching rules:
//   - No tag filter (e.g. "dotnet") → matches all tags and variants
//   - Exact tag filter (e.g. "dotnet:8.0.300") → matches only that exact tag
//   - Exact variant filter (e.g. "dotnet:8.0.300-node") → matches only that variant
func matchesFilters(filters []Filter, imageName, tagName string) bool {
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

// BuildProject builds all images in the project according to the dependency
// order, applying filters and pushing to the registry when dependents exist.
func BuildProject(ctx context.Context, client *Client, opts *ProjectBuildOpts) error {
	if opts.ProgressOut == nil {
		opts.ProgressOut = os.Stdout
	}

	if opts.BuildOrder.HasDependencies() {
		return buildWithDeps(ctx, client, opts)
	}
	return buildWithoutDeps(ctx, client, opts)
}

func buildWithDeps(ctx context.Context, client *Client, opts *ProjectBuildOpts) error {
	for _, imgName := range opts.BuildOrder.Order() {
		var imageDef *model.Image
		for _, img := range opts.Project.ImagesByIdentifier {
			if img.Name == imgName {
				imageDef = img
				break
			}
		}
		if imageDef == nil {
			slog.Warn("Image not found in project", "image", imgName)
			continue
		}

		for tagName := range imageDef.Tags {
			buildBase := matchesFilters(opts.Filters, imgName, tagName)

			if buildBase {
				platforms := platform.Resolve(opts.Project.Config.Platforms, imageDef.Platforms, nil)
				for _, platformStr := range platforms {
					if err := buildTag(ctx, client, opts, imageDef, tagName, platformStr); err != nil {
						return err
					}
				}
			}

			// Build variants
			for variantName, variantDef := range imageDef.Variants {
				variantTagName := tagName + variantDef.TagSuffix
				if !matchesFilters(opts.Filters, imgName, variantTagName) {
					continue
				}

				platforms := platform.Resolve(opts.Project.Config.Platforms, imageDef.Platforms, variantDef.Platforms)
				for _, platformStr := range platforms {
					if err := buildVariant(ctx, client, opts, imageDef, tagName, variantName, variantDef, platformStr); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

func buildWithoutDeps(ctx context.Context, client *Client, opts *ProjectBuildOpts) error {
	for _, images := range opts.Project.ImagesByName {
		for _, imageDef := range images {
			for tagName := range imageDef.Tags {
				if matchesFilters(opts.Filters, imageDef.Name, tagName) {
					platforms := platform.Resolve(opts.Project.Config.Platforms, imageDef.Platforms, nil)
					for _, platformStr := range platforms {
						if err := buildTag(ctx, client, opts, imageDef, tagName, platformStr); err != nil {
							return err
						}
					}
				}

				for variantName, variantDef := range imageDef.Variants {
					variantTag := tagName + variantDef.TagSuffix
					if !matchesFilters(opts.Filters, imageDef.Name, variantTag) {
						continue
					}
					platforms := platform.Resolve(opts.Project.Config.Platforms, imageDef.Platforms, variantDef.Platforms)
					for _, platformStr := range platforms {
						if err := buildVariant(ctx, client, opts, imageDef, tagName, variantName, variantDef, platformStr); err != nil {
							return err
						}
					}
				}
			}
		}
	}
	return nil
}

func buildTag(ctx context.Context, client *Client, opts *ProjectBuildOpts, imageDef *model.Image, tagName, platformStr string) error {
	dockerfilePath := filepath.Join(opts.DistPath, imageDef.Name, tagName, "Dockerfile")
	if _, err := os.Stat(dockerfilePath); os.IsNotExist(err) {
		return fmt.Errorf("Dockerfile not found for %s:%s at %s", imageDef.Name, tagName, dockerfilePath)
	}

	hiveDeps, err := ResolveHiveDeps(HiveDepsOpts{
		DockerfilePath:  dockerfilePath,
		DistPath:        opts.DistPath,
		PlatformStr:     platformStr,
		RegistryAddress: opts.registryAddress(),
		BuildID:         opts.BuildID,
	})
	if err != nil {
		return fmt.Errorf("failed to resolve hive deps for %s:%s: %w", imageDef.Name, tagName, err)
	}
	if hiveDeps != nil {
		defer hiveDeps.Cleanup()
	}

	root, _ := filepath.Abs(filepath.Dir(dockerfilePath))
	imageTag := fmt.Sprintf("%s:%s", imageDef.Name, tagName)
	tf := TarFilePath(opts.DistPath, imageDef.Name, tagName, platformStr)

	if err := os.MkdirAll(filepath.Dir(tf), 0755); err != nil {
		return fmt.Errorf("failed to create platform dir for %s: %w", imageTag, err)
	}

	config, err := ResolveTagConfig(imageDef, imageDef.Tags[tagName])
	if err != nil {
		return fmt.Errorf("failed to resolve build args for %s:%s: %w", imageDef.Name, tagName, err)
	}

	buildOpts := &BuildOpts{
		ImageName:        imageTag,
		Platform:         platformStr,
		TarFile:          tf,
		Cache:            opts.Cache,
		ContextDir:       root,
		BuildArgs:        config.BuildArgs,
		Secrets:          config.Secrets,
		RegistryRef:      opts.registryRef(imageDef.Name, tagName, platformStr),
		RegistryInsecure: opts.registryInsecure(),
		DockerMediaTypes: opts.useDockerMediaTypes(),
		ProgressConfig:   opts.ProgressConfig,
	}
	if hiveDeps != nil {
		buildOpts.OCIStores = hiveDeps.OCIStores
		buildOpts.NamedContexts = hiveDeps.NamedContexts
		buildOpts.Dockerfile = filepath.Base(hiveDeps.Dockerfile)
	}

	err = client.Build(ctx, buildOpts, opts.ProgressOut)
	if err != nil {
		return fmt.Errorf("build failed for %s [%s]: %w", imageTag, platformStr, err)
	}
	slog.Info("Built image", "image", imageTag, "platform", platformStr, "tar", tf)

	if opts.OnBuild != nil {
		opts.OnBuild(imageTag, tf)
	}
	return nil
}

func buildVariant(ctx context.Context, client *Client, opts *ProjectBuildOpts, imageDef *model.Image, tagName, variantName string, variantDef *model.ImageVariant, platformStr string) error {
	variantTagName := tagName + variantDef.TagSuffix
	variantDockerfilePath := filepath.Join(opts.DistPath, imageDef.Name, variantTagName, "Dockerfile")
	if _, err := os.Stat(variantDockerfilePath); os.IsNotExist(err) {
		slog.Warn("Dockerfile not found for variant", "image", imageDef.Name, "tag", tagName, "variant", variantName, "path", variantDockerfilePath)
		return nil
	}

	hiveDeps, err := ResolveHiveDeps(HiveDepsOpts{
		DockerfilePath:  variantDockerfilePath,
		DistPath:        opts.DistPath,
		PlatformStr:     platformStr,
		RegistryAddress: opts.registryAddress(),
		BuildID:         opts.BuildID,
	})
	if err != nil {
		return fmt.Errorf("failed to resolve hive deps for variant %s:%s:%s: %w", imageDef.Name, tagName, variantName, err)
	}
	if hiveDeps != nil {
		defer hiveDeps.Cleanup()
	}

	root, _ := filepath.Abs(filepath.Dir(variantDockerfilePath))
	variantTag := fmt.Sprintf("%s:%s%s", imageDef.Name, tagName, variantDef.TagSuffix)
	tf := TarFilePath(opts.DistPath, imageDef.Name, variantTagName, platformStr)

	if err := os.MkdirAll(filepath.Dir(tf), 0755); err != nil {
		return fmt.Errorf("failed to create platform dir for variant %s: %w", variantTag, err)
	}

	config, err := ResolveVariantConfig(imageDef, variantDef, imageDef.Tags[tagName])
	if err != nil {
		return fmt.Errorf("failed to resolve build args for variant %s:%s:%s: %w", imageDef.Name, tagName, variantName, err)
	}

	buildOpts := &BuildOpts{
		ImageName:        variantTag,
		Platform:         platformStr,
		TarFile:          tf,
		Cache:            opts.Cache,
		ContextDir:       root,
		BuildArgs:        config.BuildArgs,
		Secrets:          config.Secrets,
		RegistryRef:      opts.registryRef(imageDef.Name, variantTagName, platformStr),
		RegistryInsecure: opts.registryInsecure(),
		DockerMediaTypes: opts.useDockerMediaTypes(),
		ProgressConfig:   opts.ProgressConfig,
	}
	if hiveDeps != nil {
		buildOpts.OCIStores = hiveDeps.OCIStores
		buildOpts.NamedContexts = hiveDeps.NamedContexts
		buildOpts.Dockerfile = filepath.Base(hiveDeps.Dockerfile)
	}

	err = client.Build(ctx, buildOpts, opts.ProgressOut)
	if err != nil {
		return fmt.Errorf("build failed for variant %s [%s]: %w", variantTag, platformStr, err)
	}
	slog.Info("Built variant", "variant", variantTag, "platform", platformStr, "tar", tf)

	if opts.OnBuild != nil {
		opts.OnBuild(variantTag, tf)
	}
	return nil
}
