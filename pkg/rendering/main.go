package rendering

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/timo-reymann/ContainerHive/internal/buildconfig_resolver"
	"github.com/timo-reymann/ContainerHive/internal/file_resolver"
	"github.com/timo-reymann/ContainerHive/internal/semantic_tags"
	"github.com/timo-reymann/ContainerHive/pkg/model"
	"golang.org/x/sync/errgroup"
)

// ResolveLatestAlias returns the tag that latestAlias should point to — the
// highest semantic version found in tags. Returns ("", nil) if latestAlias is
// empty. Returns ("", error) if latestAlias is set but no tags parse as semantic
// versions.
func ResolveLatestAlias(tags []string, latestAlias string) (string, error) {
	if latestAlias == "" {
		return "", nil
	}
	var highest *semantic_tags.SemanticTagVersion
	var highestTag string
	for _, tag := range tags {
		ver, err := semantic_tags.NewSemanticVersion(tag)
		if err != nil {
			continue
		}
		if highest == nil || ver.Greater(highest) {
			highest = ver
			highestTag = tag
		}
	}
	if highestTag == "" {
		return "", fmt.Errorf("latest_alias %q configured but no semantic version tags found", latestAlias)
	}
	return highestTag, nil
}

// ResolveAliases computes the alias map for a set of tags, ensuring each alias
// points to the highest version that claims it.
func ResolveAliases(tags []string) map[string]string {
	aliases := make(map[string]string)
	aliasVersions := make(map[string]*semantic_tags.SemanticTagVersion)

	for _, tag := range tags {
		ver, err := semantic_tags.NewSemanticVersion(tag)
		if err != nil {
			continue
		}

		for _, alias := range ver.GetLowerVariants() {
			existing, ok := aliasVersions[alias]
			if !ok || ver.Greater(existing) {
				aliases[alias] = tag
				aliasVersions[alias] = ver
			}
		}
	}

	return aliases
}

func processImagesForName(ctx context.Context, rootPath string, images []*model.Image) error {
	// Build tag and variant directories in parallel
	eg, _ := errgroup.WithContext(ctx)
	for _, imageDef := range images {
		imageDef := imageDef

		for tag, tagDef := range imageDef.Tags {
			tag := tag
			tagDef := tagDef

			eg.Go(func() error {
				tagPath := filepath.Join(rootPath, tag)
				if err := setupImageTagDir(tagPath, imageDef, tagDef); err != nil {
					return err
				}

				for _, variantDef := range imageDef.Variants {
					rootPath := rootPath
					variantDef := variantDef
					variantTag := tag + variantDef.TagSuffix

					eg.Go(func() error {
						variantPath := filepath.Join(rootPath, variantTag)
						return setupVariantDir(variantPath, imageDef, tag, tagDef, variantDef)
					})
				}
				return nil
			})
		}
	}

	return eg.Wait()
}

func createTestsFolder(rootPath string) (string, error) {
	testsRoot := filepath.Join(rootPath, "tests")
	if err := mkdir(testsRoot); err != nil {
		return "", errors.Join(errors.New("failed to create tests directory"), err)
	}
	return testsRoot, nil
}

func fixUpEntrypoint(root, entryPath string) string {
	return filepath.Join(root, filepath.Base(file_resolver.RemoveTemplateExt(entryPath)))
}

func setupImageTagDir(tagPath string, image *model.Image, tag *model.Tag) error {
	if err := mkdir(tagPath); err != nil {
		return errors.Join(errors.New("failed to create tag directory"), err)
	}

	resolved, err := buildconfig_resolver.ForTag(image, tag)
	if err != nil {
		return errors.Join(errors.New("failed to resolve build configuration"), err)
	}

	tmplCtx := newTemplateContext(image, resolved)

	if image.BuildEntryPointPath != "" {
		// Strip template extension for output filename
		if err := file_resolver.CopyAndRenderFile(tmplCtx, image.BuildEntryPointPath, fixUpEntrypoint(tagPath, image.BuildEntryPointPath)); err != nil {
			return errors.Join(errors.New("failed to copy build entrypoint"), err)
		}
	}

	if image.RootFSDir != "" {
		if err := copyRootFs(image.RootFSDir, tagPath); err != nil {
			return errors.Join(errors.New("failed to copy rootfs"), err)
		}
	}

	if image.TestConfigFilePath != "" {
		testsRoot, err := createTestsFolder(tagPath)
		if err != nil {
			return err
		}

		if err := file_resolver.CopyAndRenderFile(tmplCtx, image.TestConfigFilePath, filepath.Join(testsRoot, "image.yml")); err != nil {
			return err
		}
	}

	return nil
}

func setupVariantDir(variantPath string, image *model.Image, tagName string, tag *model.Tag, variantDef *model.ImageVariant) error {
	resolved, err := buildconfig_resolver.ForTagVariant(image, variantDef, tag)
	if err != nil {
		return errors.Join(errors.New("failed to resolve build configuration for variant"), err)
	}

	tmplCtx := newTemplateContext(image, resolved)

	if err := mkdir(variantPath); err != nil {
		return errors.Join(errors.New("failed to create variant directory"), err)
	}

	if variantDef.BuildEntryPointPath != "" {
		entrypoint := fixUpEntrypoint(variantPath, variantDef.BuildEntryPointPath)
		if err := file_resolver.CopyAndRenderFile(tmplCtx, variantDef.BuildEntryPointPath, entrypoint); err != nil {
			return errors.Join(errors.New("failed to copy build entrypoint"), err)
		}
		if err := replaceHiveParent(entrypoint, image.Name, tagName); err != nil {
			return errors.Join(errors.New("failed to resolve __hive_parent__ in variant entrypoint"), err)
		}
	}

	if image.RootFSDir != "" {
		if err := copyRootFs(image.RootFSDir, variantPath); err != nil {
			return errors.Join(errors.New("failed to copy rootfs for variant from base version"), err)
		}
	}

	if variantDef.RootFSDir != "" {
		if err := copyRootFs(variantDef.RootFSDir, variantPath); err != nil {
			return errors.Join(errors.New("failed to copy rootfs for variant"), err)
		}
	}

	if image.TestConfigFilePath != "" || variantDef.TestConfigFilePath != "" {
		testsRoot, err := createTestsFolder(variantPath)
		if err != nil {
			return err
		}

		if image.TestConfigFilePath != "" {
			if err := file_resolver.CopyAndRenderFile(tmplCtx, image.TestConfigFilePath, filepath.Join(testsRoot, "image.yml")); err != nil {
				return errors.Join(errors.New("failed to copy test config file"), err)
			}
		}

		if variantDef.TestConfigFilePath != "" {
			if err := file_resolver.CopyAndRenderFile(tmplCtx, variantDef.TestConfigFilePath, filepath.Join(testsRoot, "variant.yml")); err != nil {
				return errors.Join(errors.New("failed to copy test config file"), err)
			}
		}
	}

	return nil
}

const hiveParentPlaceholder = "__hive_parent__"

// replaceHiveParent replaces __hive_parent__ in a rendered file with the
// concrete __hive__/imageName:tagName reference.
func replaceHiveParent(filePath, imageName, tagName string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	content := string(data)
	if !strings.Contains(content, hiveParentPlaceholder) {
		return nil
	}

	parentRef := fmt.Sprintf("__hive__/%s:%s", imageName, tagName)
	replaced := strings.ReplaceAll(content, hiveParentPlaceholder, parentRef)
	return os.WriteFile(filePath, []byte(replaced), 0644)
}

// RenderProject renders all image Dockerfiles and test configs into the target directory.
func RenderProject(ctx context.Context, project *model.ContainerHiveProject, targetPath string) error {
	_ = os.RemoveAll(targetPath)

	err := mkdir(targetPath)
	if err != nil {
		return errors.Join(errors.New("failed to create target directory"), err)
	}
	eg, _ := errgroup.WithContext(ctx)

	for name, images := range project.ImagesByName {
		images := images
		nameRootPath := filepath.Join(targetPath, name)
		err := mkdir(nameRootPath)
		if err != nil {
			return errors.Join(errors.New("failed to create image directory for "+name), err)
		}

		eg.Go(func() error {
			return processImagesForName(ctx, nameRootPath, images)
		})
	}

	return eg.Wait()
}
