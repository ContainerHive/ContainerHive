// Package changes detects which images a set of changed files affects. An
// image is changed when a file in its directory changed, and affected when it
// or any image it depends on changed.
package changes

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/ContainerHive/ContainerHive/internal/changes"
	"github.com/ContainerHive/ContainerHive/internal/dependency"
	"github.com/ContainerHive/ContainerHive/pkg/model"
)

// FileName is the name of the change set file in the dist directory.
const FileName = "changes.json"

// Unmatched modes, see ParseUnmatched.
const (
	UnmatchedAll    = string(changes.UnmatchedAll)
	UnmatchedIgnore = string(changes.UnmatchedIgnore)
)

// ParseUnmatched validates an unmatched mode: "all" rebuilds every image for
// a changed file outside every image directory, "ignore" drops it.
func ParseUnmatched(value string) error {
	_, err := changes.ParseUnmatched(value)
	return err
}

// Set is the result of a change detection run.
type Set struct {
	// Base is the git revision the changes were computed against, if any.
	Base string `json:"base,omitempty"`
	// Global is set when a change rebuilds every image.
	Global bool `json:"global"`
	// Reason explains a global change.
	Reason string `json:"reason,omitempty"`
	// Changed holds the images with a changed file in their directory.
	Changed []string `json:"changed"`
	// Affected holds the changed images and all their dependents.
	Affected []string `json:"affected"`
}

// IsAffected reports whether the image is affected. A nil set affects every
// image, so callers without change detection keep their behavior.
func (s *Set) IsAffected(name string) bool {
	return s == nil || slices.Contains(s.Affected, name)
}

// Options configures Detect.
type Options struct {
	// Unmatched is "all" (the default) or "ignore".
	Unmatched string
}

// Detect classifies files, given relative to the project root, and returns
// the affected images.
func Detect(project *model.ContainerHiveProject, files []string, opts Options) (*Set, error) {
	unmatched := changes.UnmatchedAll
	if opts.Unmatched != "" {
		parsed, err := changes.ParseUnmatched(opts.Unmatched)
		if err != nil {
			return nil, err
		}
		unmatched = parsed
	}

	graph, err := imageGraph(project)
	if err != nil {
		return nil, err
	}

	cfg := project.Config.ChangeDetection
	result, err := changes.Classify(files, changes.Rules{
		Exclude:       cfg.ExcludePatterns(),
		TreatAsGlobal: cfg.TreatAsGlobalPatterns(),
		ImageDirs:     ImageDirs(project),
		Unmatched:     unmatched,
	})
	if err != nil {
		return nil, err
	}

	set := &Set{Changed: nonNil(result.Changed)}
	if result.Global {
		set.Global = true
		set.Reason = fmt.Sprintf("%s rebuilds all images", result.GlobalFile)
		set.Affected = imageNames(project)
		return set, nil
	}
	set.Affected = changes.Expand(result.Changed, graph.Dependents)
	return set, nil
}

// DetectFromGit computes the files changed between the merge base of base and
// HEAD and returns the affected images. When the diff cannot be computed, for
// example in a shallow clone, every image is affected.
func DetectFromGit(project *model.ContainerHiveProject, base string, opts Options) (*Set, error) {
	diff, err := changes.DiffFiles(project.RootDir, base)
	if err != nil {
		slog.Warn("Could not compute changed files, treating every image as affected", "base", base, "error", err)
		return everything(project, base, fmt.Sprintf("changed files unknown: %v", err))
	}

	files, err := projectRelative(diff.RepoRoot, project.RootDir, diff.Files)
	if err != nil {
		return nil, err
	}

	set, err := Detect(project, files, opts)
	if err != nil {
		return nil, err
	}
	set.Base = base
	return set, nil
}

// ImageDirs maps every image name to its directories relative to the project
// root, slash-separated.
func ImageDirs(project *model.ContainerHiveProject) map[string][]string {
	dirs := make(map[string][]string, len(project.ImagesByName))
	for name, images := range project.ImagesByName {
		for _, img := range images {
			dirs[name] = append(dirs[name], imageDir(img))
		}
		slices.Sort(dirs[name])
	}
	return dirs
}

// TriggerDirs maps every image name to the directories whose changes affect
// it: its own and those of every image it depends on, transitively. Global
// patterns are not part of the result.
func TriggerDirs(project *model.ContainerHiveProject) (map[string][]string, error) {
	graph, err := imageGraph(project)
	if err != nil {
		return nil, err
	}
	imageDirs := ImageDirs(project)

	result := make(map[string][]string, len(imageDirs))
	for name := range imageDirs {
		var dirs []string
		for _, dep := range changes.Expand([]string{name}, graph.Dependencies) {
			dirs = append(dirs, imageDirs[dep]...)
		}
		slices.Sort(dirs)
		result[name] = slices.Compact(dirs)
	}
	return result, nil
}

// Write stores the set as changes.json in the dist directory.
func Write(distPath string, set *Set) error {
	data, err := json.MarshalIndent(set, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(distPath, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(distPath, FileName), append(data, '\n'), 0o644)
}

// Load reads changes.json from the dist directory. It returns nil without an
// error when the file does not exist, which means every image is affected.
func Load(distPath string) (*Set, error) {
	data, err := os.ReadFile(filepath.Join(distPath, FileName))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var set Set
	if err := json.Unmarshal(data, &set); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", FileName, err)
	}
	return &set, nil
}

func everything(project *model.ContainerHiveProject, base, reason string) (*Set, error) {
	return &Set{Base: base, Global: true, Reason: reason, Changed: []string{}, Affected: imageNames(project)}, nil
}

func imageNames(project *model.ContainerHiveProject) []string {
	names := make([]string, 0, len(project.ImagesByName))
	for name := range project.ImagesByName {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

func imageGraph(project *model.ContainerHiveProject) (*dependency.Graph, error) {
	graph, err := dependency.BuildDependencyGraph(dependency.ScanProjectSource(project), project)
	if err != nil {
		return nil, fmt.Errorf("failed to build dependency graph: %w", err)
	}
	return graph, nil
}

// imageDir returns the image directory relative to the project root. The
// identifier is the path below images/, which avoids comparing the symlink
// resolved image root with the project root.
func imageDir(img *model.Image) string {
	return "images/" + filepath.ToSlash(img.Identifier)
}

// projectRelative converts repository-relative paths to project-relative
// paths and drops files outside the project.
func projectRelative(repoRoot, projectRoot string, files []string) ([]string, error) {
	prefix, err := relativeDir(repoRoot, projectRoot)
	if err != nil {
		return nil, err
	}
	if prefix == "" {
		return files, nil
	}

	var result []string
	for _, file := range files {
		if rel, ok := strings.CutPrefix(file, prefix+"/"); ok {
			result = append(result, rel)
		}
	}
	return result, nil
}

func relativeDir(base, target string) (string, error) {
	resolvedBase, err := filepath.EvalSymlinks(base)
	if err != nil {
		return "", err
	}
	resolvedTarget, err := filepath.EvalSymlinks(target)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(resolvedBase, resolvedTarget)
	if err != nil {
		return "", err
	}
	if rel == "." {
		return "", nil
	}
	if rel == ".." || strings.HasPrefix(rel, "../") {
		return "", fmt.Errorf("project root %s is outside the git repository %s", target, base)
	}
	return filepath.ToSlash(rel), nil
}

func nonNil(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}
