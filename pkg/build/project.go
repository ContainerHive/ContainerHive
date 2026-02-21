package build

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/timo-reymann/ContainerHive/pkg/cache"
	"github.com/timo-reymann/ContainerHive/pkg/deps"
	"github.com/timo-reymann/ContainerHive/pkg/model"
)

// Registry abstracts push operations so pkg/build does not import internal/registry.
type Registry interface {
	Address() string
	IsLocal() bool
	Push(ctx context.Context, imageName, tag, ociTarPath string) error
}

// Filter selects a subset of images/tags to build.
// Empty fields match everything.
type Filter struct {
	ImageName       string
	TagName         string
	IncludeVariants bool
}

// ProjectBuildOpts holds shared configuration for a project-wide build.
type ProjectBuildOpts struct {
	Project    *model.ContainerHiveProject
	BuildOrder *deps.BuildOrder
	DistPath   string
	Platform   string
	Cache      cache.BuildkitCache
	Registry   Registry // nil when no inter-image dependencies exist
	ProgressOut io.Writer
	Filters    []Filter // empty = build everything
	BuildID    string   // if set, tag directories are suffixed with +<BuildID>

	// OnBuild is called after each successful build with the image tag and tar path.
	OnBuild func(imageTag, tarFile string)
}

// dirTag returns the tag directory name, accounting for BuildID suffix.
func (o *ProjectBuildOpts) dirTag(tagName string) string {
	if o.BuildID != "" {
		return tagName + "+" + o.BuildID
	}
	return tagName
}

func matchesFilters(filters []Filter, imageName, tagName string, isVariant bool) bool {
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
		if isVariant && !f.IncludeVariants {
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
			log.Printf("Warning: Image %s not found in project", imgName)
			continue
		}

		for tagName := range imageDef.Tags {
			if !matchesFilters(opts.Filters, imgName, tagName, false) {
				continue
			}

			if err := buildTag(ctx, client, opts, imageDef, tagName); err != nil {
				return err
			}

			// Build variants
			for variantName, variantDef := range imageDef.Variants {
				variantTagName := tagName + variantDef.TagSuffix
				if !matchesFilters(opts.Filters, imgName, variantTagName, true) {
					continue
				}

				if err := buildVariant(ctx, client, opts, imageDef, tagName, variantName, variantDef); err != nil {
					return err
				}

				// Push variant if dependents exist
				if opts.Registry != nil && len(opts.BuildOrder.Dependents(imgName)) > 0 {
					variantDirTag := opts.dirTag(tagName) + variantDef.TagSuffix
					pushTag := tagName + variantDef.TagSuffix
					tf := TarFilePath(opts.DistPath, imgName, variantDirTag)
					if err := opts.Registry.Push(ctx, imgName, pushTag, tf); err != nil {
						log.Printf("Warning: Failed to push variant %s:%s to registry: %v", imgName, pushTag, err)
					}
				}
			}

			// Push base tag if dependents exist
			if opts.Registry != nil && len(opts.BuildOrder.Dependents(imgName)) > 0 {
				tf := TarFilePath(opts.DistPath, imgName, opts.dirTag(tagName))
				if err := opts.Registry.Push(ctx, imgName, tagName, tf); err != nil {
					log.Printf("Warning: Failed to push %s:%s to registry: %v", imgName, tagName, err)
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
				if !matchesFilters(opts.Filters, imageDef.Name, tagName, false) {
					continue
				}

				dirTag := opts.dirTag(tagName)
				dockerfilePath := filepath.Join(opts.DistPath, imageDef.Name, dirTag, "Dockerfile")
				if _, err := os.Stat(dockerfilePath); os.IsNotExist(err) {
					log.Printf("Warning: Dockerfile not found for %s:%s at %s", imageDef.Name, tagName, dockerfilePath)
					continue
				}

				imageTag := fmt.Sprintf("%s:%s", imageDef.Name, tagName)
				tf := TarFilePath(opts.DistPath, imageDef.Name, dirTag)

				err := client.Build(ctx, &BuildOpts{
					ImageName:  imageTag,
					Platform:   opts.Platform,
					TarFile:    tf,
					Cache:      opts.Cache,
					ContextDir: filepath.Dir(dockerfilePath),
				}, opts.ProgressOut)
				if err != nil {
					return fmt.Errorf("build failed for %s: %w", imageTag, err)
				}
				log.Printf("Built %s -> %s", imageTag, tf)

				if opts.OnBuild != nil {
					opts.OnBuild(imageTag, tf)
				}
			}
		}
	}
	return nil
}

func buildTag(ctx context.Context, client *Client, opts *ProjectBuildOpts, imageDef *model.Image, tagName string) error {
	dirTag := opts.dirTag(tagName)
	dockerfilePath := filepath.Join(opts.DistPath, imageDef.Name, dirTag, "Dockerfile")
	if _, err := os.Stat(dockerfilePath); os.IsNotExist(err) {
		return fmt.Errorf("Dockerfile not found for %s:%s at %s", imageDef.Name, tagName, dockerfilePath)
	}

	patchedPath, cleanup, err := PatchHiveRefs(dockerfilePath, opts.Registry.Address())
	if err != nil {
		return fmt.Errorf("failed to rewrite hive refs for %s:%s: %w", imageDef.Name, tagName, err)
	}
	defer cleanup()

	root, _ := filepath.Abs(filepath.Dir(patchedPath))
	imageTag := fmt.Sprintf("%s:%s", imageDef.Name, tagName)
	tf := TarFilePath(opts.DistPath, imageDef.Name, dirTag)

	config, err := ResolveTagConfig(imageDef, imageDef.Tags[tagName])
	if err != nil {
		return fmt.Errorf("failed to resolve build args for %s:%s: %w", imageDef.Name, tagName, err)
	}

	err = client.Build(ctx, &BuildOpts{
		ImageName:  imageTag,
		Platform:   opts.Platform,
		TarFile:    tf,
		Cache:      opts.Cache,
		ContextDir: root,
		Dockerfile: "Dockerfile.patched",
		BuildArgs:  config.BuildArgs,
		Secrets:    config.Secrets,
	}, opts.ProgressOut)
	if err != nil {
		log.Printf("Warning: Build failed for %s: %v", imageTag, err)
		return nil
	}
	log.Printf("Built %s -> %s", imageTag, tf)

	if opts.OnBuild != nil {
		opts.OnBuild(imageTag, tf)
	}
	return nil
}

func buildVariant(ctx context.Context, client *Client, opts *ProjectBuildOpts, imageDef *model.Image, tagName, variantName string, variantDef *model.ImageVariant) error {
	variantDirTag := opts.dirTag(tagName) + variantDef.TagSuffix
	variantDockerfilePath := filepath.Join(opts.DistPath, imageDef.Name, variantDirTag, "Dockerfile")
	if _, err := os.Stat(variantDockerfilePath); os.IsNotExist(err) {
		log.Printf("Warning: Dockerfile not found for variant %s:%s:%s at %s", imageDef.Name, tagName, variantName, variantDockerfilePath)
		return nil
	}

	patchedPath, cleanup, err := PatchHiveRefs(variantDockerfilePath, opts.Registry.Address())
	if err != nil {
		return fmt.Errorf("failed to rewrite hive refs for variant %s:%s:%s: %w", imageDef.Name, tagName, variantName, err)
	}
	defer cleanup()

	root, _ := filepath.Abs(filepath.Dir(patchedPath))
	variantTag := fmt.Sprintf("%s:%s%s", imageDef.Name, tagName, variantDef.TagSuffix)
	tf := TarFilePath(opts.DistPath, imageDef.Name, variantDirTag)

	config, err := ResolveVariantConfig(imageDef, variantDef, imageDef.Tags[tagName])
	if err != nil {
		return fmt.Errorf("failed to resolve build args for variant %s:%s:%s: %w", imageDef.Name, tagName, variantName, err)
	}

	err = client.Build(ctx, &BuildOpts{
		ImageName:  variantTag,
		Platform:   opts.Platform,
		TarFile:    tf,
		Cache:      opts.Cache,
		ContextDir: root,
		Dockerfile: "Dockerfile.patched",
		BuildArgs:  config.BuildArgs,
		Secrets:    config.Secrets,
	}, opts.ProgressOut)
	if err != nil {
		log.Printf("Warning: Build failed for variant %s: %v", variantTag, err)
		return nil
	}
	log.Printf("Built variant %s -> %s", variantTag, tf)

	if opts.OnBuild != nil {
		opts.OnBuild(variantTag, tf)
	}
	return nil
}
