package build

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/ContainerHive/ContainerHive/pkg/cache"
	"github.com/ContainerHive/ContainerHive/pkg/deps"
	"github.com/ContainerHive/ContainerHive/pkg/model"
	"github.com/ContainerHive/ContainerHive/pkg/platform"
	"github.com/ContainerHive/ContainerHive/pkg/progress"
	"github.com/ContainerHive/ContainerHive/pkg/shard"
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
	BuildID        string   // if set, registry push/retag uses tags suffixed with -build.<BuildID>
	Shard          shard.Shard

	// OnBuild is called after each successful build with the image tag and tar path.
	OnBuild func(imageTag, tarFile string)

	// baseOwns/variantOwns are built once in BuildProject from Shard and
	// reused across buildWithDeps/buildWithoutDeps, so TagIndex is computed
	// only once per run rather than once per tag.
	baseOwns    func(identifier, baseTagName string) bool
	variantOwns func(identifier, tagName string) bool
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

// ownsBase/ownsVariant default to match-all when baseOwns/variantOwns were
// never set - e.g. by tests or other callers that build ProjectBuildOpts by
// hand rather than through BuildProject, where Shard is always disabled.
func (o *ProjectBuildOpts) ownsBase(identifier, baseTagName string) bool {
	if o.baseOwns == nil {
		return true
	}
	return o.baseOwns(identifier, baseTagName)
}

func (o *ProjectBuildOpts) ownsVariant(identifier, tagName string) bool {
	if o.variantOwns == nil {
		return true
	}
	return o.variantOwns(identifier, tagName)
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

// buildPlatforms builds every platform for a single image tag. A failing
// platform does not stop the remaining ones, so a single broken or unsupported
// platform cannot silently skip the others. All failures are joined.
func buildPlatforms(platforms []string, build func(platformStr string) error) error {
	var errs []error
	for _, platformStr := range platforms {
		if err := build(platformStr); err != nil {
			slog.Error("Build failed", "platform", platformStr, "error", err)
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// BuildProject builds all images in the project according to the dependency
// order, applying filters and pushing to the registry when dependents exist.
func BuildProject(ctx context.Context, client *Client, opts *ProjectBuildOpts) error {
	if opts.ProgressOut == nil {
		opts.ProgressOut = os.Stdout
	}

	if opts.Shard.Enabled() {
		owned := shard.OwnedUnits(opts.Project, opts.Shard)
		if len(owned) == 0 {
			slog.Warn("Nothing to do for this shard", "shard", opts.Shard.Current, "of", opts.Shard.Max)
			return nil
		}
		slog.Info("Shard selected units", "shard", opts.Shard.Current, "of", opts.Shard.Max, "units", len(owned))

		if opts.BuildOrder.HasDependencies() {
			slog.Warn("Sharding combined with inter-image dependencies: a dependency not owned by this shard resolves via the registry and must already be pushed there")
		}
	}
	opts.baseOwns = shard.NewBaseTagSharder(opts.Project, opts.Shard)
	opts.variantOwns = shard.NewTagSharder(opts.Project, opts.Shard)

	if opts.BuildOrder.HasDependencies() {
		return buildWithDeps(ctx, client, opts)
	}
	return buildWithoutDeps(ctx, client, opts)
}

func buildWithDeps(ctx context.Context, client *Client, opts *ProjectBuildOpts) error {
	for _, imgName := range opts.BuildOrder.Order() {
		images := opts.Project.ImagesByName[imgName]
		if len(images) == 0 {
			slog.Warn("Image not found in project", "image", imgName)
			continue
		}

		for _, imageDef := range images {
			for tagName := range imageDef.Tags {
				buildBase := matchesFilters(opts.Filters, imgName, tagName) && opts.ownsBase(imageDef.Identifier, tagName)

				if buildBase {
					platforms := platform.Resolve(opts.Project.Config.Platforms, imageDef.Platforms, nil)
					if err := buildPlatforms(platforms, func(platformStr string) error {
						return buildTag(ctx, client, opts, imageDef, tagName, platformStr)
					}); err != nil {
						return err
					}
				}

				// Build variants
				for variantName, variantDef := range imageDef.Variants {
					variantTagName := tagName + variantDef.TagSuffix
					if !matchesFilters(opts.Filters, imgName, variantTagName) {
						continue
					}
					if !opts.ownsVariant(imageDef.Identifier, variantTagName) {
						continue
					}

					platforms := platform.Resolve(opts.Project.Config.Platforms, imageDef.Platforms, variantDef.Platforms)
					if err := buildPlatforms(platforms, func(platformStr string) error {
						return buildVariant(ctx, client, opts, imageDef, tagName, variantName, variantDef, platformStr)
					}); err != nil {
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
				if matchesFilters(opts.Filters, imageDef.Name, tagName) && opts.ownsBase(imageDef.Identifier, tagName) {
					platforms := platform.Resolve(opts.Project.Config.Platforms, imageDef.Platforms, nil)
					if err := buildPlatforms(platforms, func(platformStr string) error {
						return buildTag(ctx, client, opts, imageDef, tagName, platformStr)
					}); err != nil {
						return err
					}
				}

				for variantName, variantDef := range imageDef.Variants {
					variantTag := tagName + variantDef.TagSuffix
					if !matchesFilters(opts.Filters, imageDef.Name, variantTag) {
						continue
					}
					if !opts.ownsVariant(imageDef.Identifier, variantTag) {
						continue
					}
					platforms := platform.Resolve(opts.Project.Config.Platforms, imageDef.Platforms, variantDef.Platforms)
					if err := buildPlatforms(platforms, func(platformStr string) error {
						return buildVariant(ctx, client, opts, imageDef, tagName, variantName, variantDef, platformStr)
					}); err != nil {
						return err
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

	builtDockerfilePath := dockerfilePath
	if hiveDeps != nil {
		builtDockerfilePath = hiveDeps.Dockerfile
	}
	dockerfileContent, err := os.ReadFile(builtDockerfilePath)
	if err != nil {
		return fmt.Errorf("failed to read Dockerfile for %s:%s: %w", imageDef.Name, tagName, err)
	}
	contentHash := cache.ComputeContentHash(dockerfileContent, config.BuildArgs, platformStr)

	scope := fmt.Sprintf("%s.%s.%s.%s", imageDef.Name, tagName, platformStr, contentHash)
	scopedCache := opts.Cache
	if opts.Cache != nil {
		scopedCache = opts.Cache.WithScope(scope)
	}

	buildOpts := &BuildOpts{
		ImageName:  imageTag,
		Platform:   platformStr,
		TarFile:    tf,
		Cache:      scopedCache,
		ContextDir: root,
		BuildArgs:  config.BuildArgs,
		Secrets:    config.Secrets,
		Labels: BuildOCILabels(OCILabelArgs{
			ImageName:   imageDef.Name,
			Tag:         tagName,
			Description: imageDef.Description,
			ProjectRoot: opts.Project.RootDir,
			Project:     opts.Project.Config.Labels,
			ImageLabels: imageDef.Labels,
			TagLabels:   imageDef.Tags[tagName].Labels,
		}),
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

	builtVariantDockerfilePath := variantDockerfilePath
	if hiveDeps != nil {
		builtVariantDockerfilePath = hiveDeps.Dockerfile
	}
	variantDockerfileContent, err := os.ReadFile(builtVariantDockerfilePath)
	if err != nil {
		return fmt.Errorf("failed to read Dockerfile for variant %s:%s:%s: %w", imageDef.Name, tagName, variantName, err)
	}
	variantContentHash := cache.ComputeContentHash(variantDockerfileContent, config.BuildArgs, platformStr)

	variantScope := fmt.Sprintf("%s.%s.%s.%s", imageDef.Name, variantTagName, platformStr, variantContentHash)
	scopedCache := opts.Cache
	if opts.Cache != nil {
		scopedCache = opts.Cache.WithScope(variantScope)
	}

	buildOpts := &BuildOpts{
		ImageName:  variantTag,
		Platform:   platformStr,
		TarFile:    tf,
		Cache:      scopedCache,
		ContextDir: root,
		BuildArgs:  config.BuildArgs,
		Secrets:    config.Secrets,
		Labels: BuildOCILabels(OCILabelArgs{
			ImageName:     imageDef.Name,
			Tag:           variantTagName,
			Description:   imageDef.Description,
			ProjectRoot:   opts.Project.RootDir,
			Project:       opts.Project.Config.Labels,
			ImageLabels:   imageDef.Labels,
			TagLabels:     imageDef.Tags[tagName].Labels,
			VariantLabels: variantDef.Labels,
		}),
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
