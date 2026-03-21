package ci

import (
	"fmt"
	"sort"

	"github.com/timo-reymann/ContainerHive/internal/dependency"
	"github.com/timo-reymann/ContainerHive/pkg/model"
)

// CIImage represents an image in the CI context.
type CIImage struct {
	Name         string
	Tags         []string
	Dependencies []string
	Depth        int
	Platforms    []string
}

// CIContext holds all data needed to render CI templates.
type CIContext struct {
	Images      []CIImage
	Platforms   []string
	Stages      []string
	Config      CIConfigContext
	Artifacts   bool
	Command     string
	Version     string
	ImageName   string
	ProjectPath string // non-empty when --project is not the default "."
}

// ChCmd returns the ch command prefix, including -p flag when a project path is set.
func (c *CIContext) ChCmd() string {
	if c.ProjectPath != "" {
		return "ch -p " + c.ProjectPath
	}
	return "ch"
}

// Dist returns the dist directory path, prefixed with the project path when set.
func (c *CIContext) Dist() string {
	if c.ProjectPath != "" {
		return c.ProjectPath + "/dist"
	}
	return "dist"
}

// CIConfigContext holds project configuration relevant to CI.
type CIConfigContext struct {
	Registry *model.RegistryConfig
	Cache    *model.CacheConfig
}

// BuildCIContext creates a CIContext from a ContainerHive project.
func BuildCIContext(project *model.ContainerHiveProject, artifacts bool) (*CIContext, error) {
	imageNames := make(map[string]bool)
	for name := range project.ImagesByName {
		imageNames[name] = true
	}

	// Build dependency graph from source Dockerfiles and explicit depends_on
	scannedGraph := dependency.ScanProjectSource(project)
	mergedGraph, err := dependency.BuildDependencyGraph(scannedGraph, project)
	if err != nil {
		return nil, fmt.Errorf("failed to build dependency graph: %w", err)
	}

	dependencies := make(map[string][]string)
	for name := range project.ImagesByName {
		deps := mergedGraph.Dependencies(name)
		if len(deps) > 0 {
			sort.Strings(deps)
			dependencies[name] = deps
		}
	}

	// Calculate depths iteratively
	depths := calculateDepths(imageNames, dependencies)

	// Build CIImage list (deduplicated by name)
	allPlatforms := make(map[string]bool)
	var ciImages []CIImage

	for name, images := range project.ImagesByName {
		// Collect tags from all image variants with same name
		tagSet := make(map[string]bool)
		for _, img := range images {
			for tagName := range img.Tags {
				tagSet[tagName] = true
			}
		}
		tags := make([]string, 0, len(tagSet))
		for t := range tagSet {
			tags = append(tags, t)
		}
		sort.Strings(tags)

		// Resolve platforms: image-level or project default
		platforms := resolvePlatforms(images[0], project.Config.Platforms)
		for _, p := range platforms {
			allPlatforms[p] = true
		}

		ciImages = append(ciImages, CIImage{
			Name:         name,
			Tags:         tags,
			Dependencies: dependencies[name],
			Depth:        depths[name],
			Platforms:    platforms,
		})
	}

	// Sort by (depth, name)
	sort.Slice(ciImages, func(i, j int) bool {
		if ciImages[i].Depth != ciImages[j].Depth {
			return ciImages[i].Depth < ciImages[j].Depth
		}
		return ciImages[i].Name < ciImages[j].Name
	})

	// Generate stages
	var stages []string
	for _, img := range ciImages {
		stages = append(stages, fmt.Sprintf("build-%s", img.Name))
		stages = append(stages, fmt.Sprintf("manifest-%s", img.Name))
	}
	stages = append(stages, "test")

	// Collect all unique platforms sorted
	platformList := make([]string, 0, len(allPlatforms))
	for p := range allPlatforms {
		platformList = append(platformList, p)
	}
	sort.Strings(platformList)

	return &CIContext{
		Images:    ciImages,
		Platforms: platformList,
		Stages:    stages,
		Config: CIConfigContext{
			Registry: project.Config.Registry,
			Cache:    project.Config.Cache,
		},
		Artifacts: artifacts,
	}, nil
}

// calculateDepths computes dependency depth for each image.
// Depth 0 = no dependencies, depth 1 = depends only on depth-0 images, etc.
func calculateDepths(imageNames map[string]bool, dependencies map[string][]string) map[string]int {
	depths := make(map[string]int)
	remaining := make(map[string]bool)
	for name := range imageNames {
		remaining[name] = true
	}

	currentDepth := 0
	for len(remaining) > 0 {
		var ready []string
		for name := range remaining {
			deps := dependencies[name]
			allResolved := true
			for _, d := range deps {
				if _, ok := depths[d]; !ok {
					allResolved = false
					break
				}
			}
			if allResolved {
				ready = append(ready, name)
			}
		}

		if len(ready) == 0 {
			// Circular dependency - assign remaining to current depth
			for name := range remaining {
				depths[name] = currentDepth
			}
			break
		}

		for _, name := range ready {
			depths[name] = currentDepth
			delete(remaining, name)
		}
		currentDepth++
	}

	return depths
}

// resolvePlatforms returns the effective platforms for an image.
// Uses image-level platforms if set, otherwise falls back to project defaults.
// Platform prefixes like "linux/" are stripped for CI use (e.g. "linux/amd64" -> "amd64").
func resolvePlatforms(img *model.Image, projectPlatforms []string) []string {
	platforms := img.Platforms
	if len(platforms) == 0 {
		platforms = projectPlatforms
	}

	result := make([]string, 0, len(platforms))
	for _, p := range platforms {
		// Strip os prefix (e.g. "linux/amd64" -> "amd64")
		for i := len(p) - 1; i >= 0; i-- {
			if p[i] == '/' {
				p = p[i+1:]
				break
			}
		}
		result = append(result, p)
	}
	return result
}
