package mcp

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ContainerHive/ContainerHive/pkg/deps"
	"github.com/ContainerHive/ContainerHive/pkg/discovery"
	"github.com/ContainerHive/ContainerHive/pkg/model"
	"github.com/goccy/go-yaml"
)

type imageInfo struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
	Variants    []string `json:"variants"`
	Versions    []string `json:"versions"`
	Platforms   []string `json:"platforms"`
}

type imageDetail struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Tags        []*model.Tag      `json:"tags"`
	Variants    map[string]string `json:"variants"`
	Versions    map[string]string `json:"versions"`
	BuildArgs   map[string]string `json:"build_args"`
	DependsOn   []string          `json:"depends_on"`
	Platforms   []string          `json:"platforms"`
}

func listImages(ctx context.Context, projectRoot string) ([]imageInfo, error) {
	project, err := discovery.DiscoverProject(ctx, projectRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to discover project: %w", err)
	}

	var images []imageInfo
	for _, imageList := range project.ImagesByName {
		for _, img := range imageList {
			tags := make([]string, 0, len(img.Tags))
			for name := range img.Tags {
				tags = append(tags, name)
			}

			variants := make([]string, 0, len(img.Variants))
			for name := range img.Variants {
				variants = append(variants, name)
			}

			versions := make([]string, 0, len(img.Versions))
			for name := range img.Versions {
				versions = append(versions, name)
			}

			images = append(images, imageInfo{
				Name:        img.Name,
				Description: img.Description,
				Tags:        tags,
				Variants:    variants,
				Versions:    versions,
				Platforms:   img.Platforms,
			})
		}
	}

	return images, nil
}

func getImage(ctx context.Context, projectRoot, name string) (*imageDetail, error) {
	project, err := discovery.DiscoverProject(ctx, projectRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to discover project: %w", err)
	}

	images, ok := project.ImagesByName[name]
	if !ok || len(images) == 0 {
		return nil, fmt.Errorf("image not found: %s", name)
	}

	img := images[0]

	variants := make(map[string]string)
	for n, v := range img.Variants {
		variants[n] = v.TagSuffix
	}

	tagsSlice := make([]*model.Tag, 0, len(img.Tags))
	for _, tag := range img.Tags {
		tagsSlice = append(tagsSlice, tag)
	}

	return &imageDetail{
		Name:        img.Name,
		Description: img.Description,
		Tags:        tagsSlice,
		Variants:    variants,
		Versions:    img.Versions,
		BuildArgs:   img.BuildArgs,
		DependsOn:   img.DependsOn,
		Platforms:   img.Platforms,
	}, nil
}

func getDependencies(ctx context.Context, projectRoot, imageName, direction string) ([]string, error) {
	project, err := discovery.DiscoverProject(ctx, projectRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to discover project: %w", err)
	}

	distPath := filepath.Join(projectRoot, model.DistDirName)
	buildOrder, err := deps.ResolveOrder(distPath, project)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve dependencies: %w", err)
	}

	if direction == "reverse" {
		return buildOrder.Dependents(imageName), nil
	}

	order := buildOrder.Order()
	var result []string

	for _, name := range order {
		if name == imageName {
			break
		}
		result = append(result, name)
	}

	return result, nil
}

func addImage(ctx context.Context, projectRoot, name, description, baseTag, dockerfileContent string) error {
	if !isValidImageName(name) {
		return fmt.Errorf("invalid image name: must not contain '..' or be absolute")
	}

	imagesDir := filepath.Join(projectRoot, "images", name)

	if _, err := os.Stat(imagesDir); err == nil {
		return fmt.Errorf("image %q already exists", name)
	}

	if err := os.MkdirAll(imagesDir, 0755); err != nil {
		return fmt.Errorf("failed to create image directory: %w", err)
	}

	dockerfilePath := filepath.Join(imagesDir, "Dockerfile")
	if dockerfileContent == "" {
		dockerfileContent = fmt.Sprintf("FROM %s\n\n# Add your layers here\n", baseTag)
	}
	if err := os.WriteFile(dockerfilePath, []byte(dockerfileContent), 0644); err != nil {
		return fmt.Errorf("failed to write Dockerfile: %w", err)
	}

	imageYAMLPath := filepath.Join(imagesDir, "image.yml")
	config := model.ImageDefinitionConfig{
		Description: description,
		Tags:        []*model.Tag{{Name: baseTag}},
	}
	imageYAMLContent, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal image.yml: %w", err)
	}
	if err := os.WriteFile(imageYAMLPath, imageYAMLContent, 0644); err != nil {
		return fmt.Errorf("failed to write image.yml: %w", err)
	}

	return nil
}

func addImageVariant(ctx context.Context, projectRoot, imageName, variantName, tagSuffix string, versions, buildArgs map[string]string) error {
	if !isValidImageName(imageName) {
		return fmt.Errorf("invalid image name: must not contain '..' or be absolute")
	}
	if !isValidImageName(variantName) {
		return fmt.Errorf("invalid variant name: must not contain '..' or be absolute")
	}

	imagesDir := filepath.Join(projectRoot, "images", imageName)
	imageYAMLPath := filepath.Join(imagesDir, "image.yml")

	data, err := os.ReadFile(imageYAMLPath)
	if err != nil {
		return fmt.Errorf("failed to read image.yml: %w", err)
	}

	var config model.ImageDefinitionConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse image.yml: %w", err)
	}

	variant := model.VariantConfig{
		Name:      variantName,
		TagSuffix: tagSuffix,
	}
	if len(versions) > 0 {
		variant.Versions = versions
	}
	if len(buildArgs) > 0 {
		variant.BuildArgs = buildArgs
	}

	config.Variants = append(config.Variants, variant)

	outputData, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal image config: %w", err)
	}

	if err := os.WriteFile(imageYAMLPath, outputData, 0644); err != nil {
		return fmt.Errorf("failed to write image.yml: %w", err)
	}

	variantConfig := model.ImageDefinitionConfig{}
	if len(versions) > 0 {
		variantConfig.Versions = versions
	}
	if len(buildArgs) > 0 {
		variantConfig.BuildArgs = buildArgs
	}
	variantYAMLContent, err := yaml.Marshal(variantConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal variant image.yml: %w", err)
	}

	variantDir := filepath.Join(imagesDir, variantName)
	if err := os.MkdirAll(variantDir, 0755); err != nil {
		return fmt.Errorf("failed to create variant directory: %w", err)
	}

	dockerfilePath := filepath.Join(variantDir, "Dockerfile")
	dockerfileContent := "FROM __hive_parent__\n\n# Add your layers here\n"
	if err := os.WriteFile(dockerfilePath, []byte(dockerfileContent), 0644); err != nil {
		return fmt.Errorf("failed to write variant Dockerfile: %w", err)
	}

	variantImageYAMLPath := filepath.Join(variantDir, "image.yml")
	if err := os.WriteFile(variantImageYAMLPath, []byte(variantYAMLContent), 0644); err != nil {
		return fmt.Errorf("failed to write variant image.yml: %w", err)
	}

	return nil
}

func isValidImageName(name string) bool {
	if filepath.IsAbs(name) {
		return false
	}
	if strings.Contains(name, "..") {
		return false
	}
	if strings.Contains(name, "/") {
		return false
	}
	return true
}
