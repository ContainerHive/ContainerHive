package ci

import (
	"fmt"
	"log/slog"
	"path"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/ContainerHive/ContainerHive/internal/actions"
	"github.com/ContainerHive/ContainerHive/internal/buildkit"
	"github.com/ContainerHive/ContainerHive/internal/dependency"
	"github.com/ContainerHive/ContainerHive/pkg/changes"
	"github.com/ContainerHive/ContainerHive/pkg/model"
	"github.com/ContainerHive/ContainerHive/pkg/platform"
	"github.com/ContainerHive/ContainerHive/pkg/rendering"
	"github.com/ContainerHive/ContainerHive/pkg/shard"
)

// CIImage represents an image in the CI context.
type CIImage struct {
	Name         string
	Tags         []string
	Aliases      map[string]string // alias -> exact tag it points at (semver series aliases and latest_alias, incl. variants)
	Dependencies []string
	Depth        int
	Platforms    []string
	ReadmePath   string // path to the image's README.md, empty if it has none; load it with the read_file template function
	ShardUnits   int    // number of tag units (base + variant tags) — the useful max
	BuildShards  int    // effective parallel for build jobs (min(ci_build_shards, ShardUnits), ≥1)
	TestShards   int    // effective parallel for test jobs (min(ci_test_shards, ShardUnits), ≥1)
	// ChangePaths are the project-relative GitLab rules:changes globs whose
	// changes affect the image: its directories, those of all its
	// dependencies and the treat_as_global patterns. Prefix them with
	// RepoPath. Empty when change detection is off or the image has more
	// paths than GitLab accepts in one rule, so its jobs always run.
	ChangePaths []string
}

// CIContext holds all data needed to render CI templates.
type CIContext struct {
	Images          []CIImage
	Platforms       []string
	Stages          []string
	Config          CIConfigContext
	Artifacts       bool
	Command         string
	Version         string
	ImageName       string
	ProjectPath     string // non-empty when --project is not the default "."
	TemplateOptions map[string]string

	// HasTagRanges gates the version-cache template steps, so projects not
	// using tag_ranges render byte-identical CI files to before.
	HasTagRanges bool

	// TagRangeUnits is recorded in the header comment, so a stale
	// committed pipeline is visible as a diff on the next generate.
	TagRangeUnits int

	// ChangeDetection gates the change detection jobs and rules, so
	// ci_change_detection: "false" renders the pipeline without them.
	ChangeDetection bool
}

// RepoPath returns a project-relative path relative to the repository root,
// prefixed with the project path when set. The result is cleaned, so an image
// directory behind a symlink (../shared/images) becomes shared/images.
func (c *CIContext) RepoPath(p string) string {
	if c.ProjectPath != "" {
		return path.Clean(c.ProjectPath + "/" + p)
	}
	return path.Clean(p)
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

// VersionCacheDir returns the tag_ranges version cache path, prefixed with
// the project path when set. It is deliberately project-relative rather
// than XDG-default: GitHub's generate job runs ch in a container with only
// $RUNNER_TEMP/ch and $GITHUB_WORKSPACE bind-mounted, and GitLab's cache:
// only accepts paths under $CI_PROJECT_DIR - a path outside the project
// directory would be invisible to one or the other.
func (c *CIContext) VersionCacheDir() string {
	dir := c.TemplateOptions["ci_version_cache_dir"]
	if c.ProjectPath != "" {
		return c.ProjectPath + "/" + dir
	}
	return dir
}

// CIConfigContext holds project configuration relevant to CI.
type CIConfigContext struct {
	Registry *model.RegistryConfig
	Cache    *model.CacheConfig
}

// RegistryHost returns the registry hostname (without path) for login.
// E.g., "docker.io/timoreymann" -> "docker.io"
func (c *CIConfigContext) RegistryHost() string {
	if c.Registry == nil {
		return ""
	}
	addr := c.Registry.Address
	for i := 0; i < len(addr); i++ {
		if addr[i] == '/' {
			return addr[:i]
		}
	}
	return addr
}

const (
	// GitLabDefaultDindImage is the Docker-in-Docker service image used by the
	// generated GitLab test jobs. The ContainerHive helper image runs rootless,
	// so it works in unprivileged Kubernetes pods (no privileged security
	// context needed).
	GitLabDefaultDindImage = "ghcr.io/containerhive/helper-images/dind-rootless:29.9"
	// GitHubDefaultDindImage is the Docker-in-Docker image used by the generated
	// GitHub Actions test jobs. GitHub-hosted runners allow privileged
	// containers but not the user namespaces the rootless daemon needs.
	GitHubDefaultDindImage = "ghcr.io/containerhive/helper-images/dind:29.9"
)

// fixedStages are the stages the GitLab pipeline defines besides the image
// stages.
var fixedStages = []string{"generate", "lint", "report", "pages"}

// defaultTemplateOptions returns the built-in template option defaults.
func defaultTemplateOptions() map[string]string {
	return map[string]string{
		"ci_buildkit_image":                     buildkit.DefaultImage,
		"ci_buildkit_version":                   buildkit.DefaultTag,
		"ci_cancel_outdated":                    "true",
		"ci_change_detection":                   "true",
		"ci_lint":                               "true",
		"ci_report":                             "true",
		"ci_sbom_generate_cpes":                 "true",
		"ci_build_shards":                       "1",
		"ci_test_shards":                        "1",
		"ci_version_cache":                      "true",
		"ci_version_cache_dir":                  ".ch-cache",
		"actions_checkout_version":              actions.CheckoutVersion,
		"actions_upload_artifact_version":       actions.UploadArtifactVersion,
		"actions_download_artifact_version":     actions.DownloadArtifactVersion,
		"actions_upload_pages_artifact_version": actions.UploadPagesArtifactVersion,
		"actions_deploy_pages_version":          actions.DeployPagesVersion,
		"actions_junit_report_version":          actions.JunitReportVersion,
		"actions_cache_version":                 actions.CacheVersion,
	}
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
		// Collect tags from all image variants with same name, including
		// variant-suffixed tags so custom templates see the full set of
		// build units (matching shard.TagIndex).
		tagSet := make(map[string]bool)
		for _, img := range images {
			for tagName := range img.Tags {
				tagSet[tagName] = true
				for _, variantDef := range img.Variants {
					tagSet[tagName+variantDef.TagSuffix] = true
				}
			}
		}

		// Aliases are resolved by the same code that retags them in the
		// registry, so templates see exactly the names that will exist.
		aliases := make(map[string]string)
		for _, img := range images {
			imgAliases, err := rendering.ResolveAllAliases(img)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve aliases for image %s: %w", name, err)
			}
			for alias, target := range imgAliases {
				if _, ok := aliases[alias]; !ok {
					aliases[alias] = target
				}
			}
		}

		readmePath := ""
		for _, img := range images {
			if img.ReadmePath != "" {
				readmePath = img.ReadmePath
				break
			}
		}

		tags := make([]string, 0, len(tagSet))
		for t := range tagSet {
			tags = append(tags, t)
		}
		sort.Strings(tags)

		// Resolve platforms: union over base tags and variants
		platforms := resolvePlatforms(images, project.Config.Platforms)
		for _, p := range platforms {
			allPlatforms[p] = true
		}

		ciImages = append(ciImages, CIImage{
			Name:         name,
			Tags:         tags,
			Aliases:      aliases,
			Dependencies: dependencies[name],
			Depth:        depths[name],
			Platforms:    platforms,
			ReadmePath:   readmePath,
		})
	}

	// Sort by (depth, name)
	sort.Slice(ciImages, func(i, j int) bool {
		if ciImages[i].Depth != ciImages[j].Depth {
			return ciImages[i].Depth < ciImages[j].Depth
		}
		return ciImages[i].Name < ciImages[j].Name
	})

	// One stage per image, in dependency order, holds its build, test and
	// manifest jobs. Jobs run as soon as their needs finish, so the stage
	// only groups them in the pipeline view. An image named like a fixed
	// pipeline stage joins that stage instead of duplicating it.
	var stages []string
	for _, img := range ciImages {
		if slices.Contains(fixedStages, img.Name) {
			continue
		}
		stages = append(stages, img.Name)
	}

	// Collect all unique platforms sorted
	platformList := make([]string, 0, len(allPlatforms))
	for p := range allPlatforms {
		platformList = append(platformList, p)
	}
	sort.Strings(platformList)

	// Merge template options: defaults first, then user overrides
	opts := defaultTemplateOptions()
	for k, v := range project.Config.TemplateOptions {
		opts[k] = v
	}
	// Only the default image is tagged per release series, a custom image
	// without an explicit version keeps the exact BuildKit version.
	if _, ok := project.Config.TemplateOptions["ci_buildkit_version"]; !ok {
		opts["ci_buildkit_version"] = buildkit.TagFor(opts["ci_buildkit_image"])
	}

	// The gotpl template coerces these with sprig's atoi, which returns 0 for
	// a non-numeric value rather than failing - validate here so a typo in
	// template_options errors loudly instead of silently rendering as
	// unsharded.
	for _, key := range []string{"ci_build_shards", "ci_test_shards"} {
		if n, err := strconv.Atoi(opts[key]); err != nil || n < 1 {
			return nil, fmt.Errorf("template_options.%s must be a positive integer, got %q", key, opts[key])
		}
	}

	// Compute per-image shard counts: effective parallel = min(requested, units), ≥1.
	// UnitCountByName iterates ImagesByName so it works even when only that map
	// is populated (e.g. test fixtures).
	requestedBuild, _ := strconv.Atoi(opts["ci_build_shards"])
	requestedTest, _ := strconv.Atoi(opts["ci_test_shards"])
	unitCounts := shard.UnitCountByName(project)

	for i := range ciImages {
		units := unitCounts[ciImages[i].Name]
		if units < 1 {
			units = 1
		}
		ciImages[i].ShardUnits = units

		buildShards := requestedBuild
		if buildShards > units {
			buildShards = units
		}
		ciImages[i].BuildShards = buildShards

		testShards := requestedTest
		if testShards > units {
			testShards = units
		}
		ciImages[i].TestShards = testShards
	}

	hasTagRanges := false
	for _, images := range project.ImagesByName {
		for _, img := range images {
			if len(img.TagRanges) > 0 {
				hasTagRanges = true
			}
		}
	}
	totalUnits := 0
	if hasTagRanges {
		for _, units := range unitCounts {
			totalUnits += units
		}
	}

	changeDetection := opts["ci_change_detection"] != "false"
	if changeDetection {
		if err := setChangePaths(project, ciImages); err != nil {
			return nil, err
		}
	}

	return &CIContext{
		Images:    ciImages,
		Platforms: platformList,
		Stages:    stages,
		Config: CIConfigContext{
			Registry: project.Config.Registry,
			Cache:    project.Config.Cache,
		},
		Artifacts:       artifacts,
		TemplateOptions: opts,
		HasTagRanges:    hasTagRanges,
		TagRangeUnits:   totalUnits,
		ChangeDetection: changeDetection,
	}, nil
}

// setChangePaths fills CIImage.ChangePaths from the image directories, the
// directories of their dependencies and the treat_as_global patterns.
func setChangePaths(project *model.ContainerHiveProject, ciImages []CIImage) error {
	triggerDirs, err := changes.TriggerDirs(project)
	if err != nil {
		return err
	}
	var globals []string
	for _, pattern := range project.Config.ChangeDetection.TreatAsGlobalPatterns() {
		globals = append(globals, gitlabGlob(pattern))
	}

	for i := range ciImages {
		var paths []string
		for _, dir := range triggerDirs[ciImages[i].Name] {
			paths = append(paths, dir+"/**/*")
		}
		paths = append(paths, globals...)
		// GitLab rejects a pipeline with more paths in one rule. Without a
		// rule the jobs are always created, and dist/changes.json still
		// turns them into no-ops when the image is not affected.
		if len(paths) > gitlabMaxChangePaths {
			slog.Warn("Too many change paths for GitLab rules:changes, image jobs always run in branch pipelines",
				"image", ciImages[i].Name, "paths", len(paths), "max", gitlabMaxChangePaths)
			continue
		}
		ciImages[i].ChangePaths = paths
	}
	return nil
}

// gitlabMaxChangePaths is the most paths GitLab accepts in one
// rules:changes:paths list.
const gitlabMaxChangePaths = 50

// gitlabGlob adapts a doublestar pattern to GitLab rules:changes, which only
// matches files: a trailing "/**" also needs "/*" to match the files below.
func gitlabGlob(pattern string) string {
	if strings.HasSuffix(pattern, "/**") {
		return pattern + "/*"
	}
	return pattern
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

// resolvePlatforms returns the effective platforms for an image: the union of
// the platforms its base tags (image-level, falling back to project defaults)
// and its variants (variant-level, falling back to the base tag platforms)
// resolve to, in order of first appearance. This way a platform only a
// variant supports still gets a job.
// The "linux/" prefix is stripped for CI use when only an arch remains
// (e.g. "linux/amd64" -> "amd64"); multi-component or non-linux platforms
// such as "linux/arm/v7" or "windows/amd64" keep their full value.
func resolvePlatforms(images []*model.Image, projectPlatforms []string) []string {
	seen := make(map[string]bool)
	var result []string
	add := func(platforms []string) {
		for _, p := range platforms {
			p = platform.Short(p)
			if seen[p] {
				continue
			}
			seen[p] = true
			result = append(result, p)
		}
	}

	for _, img := range images {
		add(platform.Resolve(nil, projectPlatforms, img.Platforms, nil))

		variantNames := make([]string, 0, len(img.Variants))
		for name := range img.Variants {
			variantNames = append(variantNames, name)
		}
		sort.Strings(variantNames)
		for _, name := range variantNames {
			add(platform.Resolve(nil, projectPlatforms, img.Platforms, img.Variants[name].Platforms))
		}
	}
	return result
}
