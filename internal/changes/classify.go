// Package changes maps changed files to the images they affect.
package changes

import (
	"fmt"
	"path"
	"slices"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

// Unmatched decides what a changed file outside every image directory, and
// matching neither exclude nor treat_as_global, does.
type Unmatched string

const (
	// UnmatchedAll rebuilds every image for an unmatched file.
	UnmatchedAll Unmatched = "all"
	// UnmatchedIgnore drops unmatched files.
	UnmatchedIgnore Unmatched = "ignore"
)

// ParseUnmatched validates an unmatched mode.
func ParseUnmatched(value string) (Unmatched, error) {
	switch Unmatched(value) {
	case UnmatchedAll, UnmatchedIgnore:
		return Unmatched(value), nil
	}
	return "", fmt.Errorf("unknown unmatched mode %q, want %s or %s", value, UnmatchedAll, UnmatchedIgnore)
}

// Rules configures Classify. All paths are slash-separated and relative to
// the project root. Paths outside the project start with ../.
type Rules struct {
	Exclude       []string
	TreatAsGlobal []string
	// ImageDirs maps an image name to its directories. Several image
	// definitions can share a name, so a name can have more than one.
	ImageDirs map[string][]string
	Unmatched Unmatched
}

// Classification is the result of Classify.
type Classification struct {
	// Global is set when a file rebuilds every image.
	Global bool
	// GlobalFile is the first file that made the classification global.
	GlobalFile string
	// Changed holds the sorted names of images with a changed file in their
	// directory. It is not expanded to dependents.
	Changed []string
}

// ValidatePatterns returns an error for the first invalid glob pattern.
func ValidatePatterns(patterns []string) error {
	for _, pattern := range patterns {
		if !doublestar.ValidatePattern(pattern) {
			return fmt.Errorf("invalid change detection pattern %q", pattern)
		}
	}
	return nil
}

// Classify sorts changed files into global changes, excluded files and image
// changes. treat_as_global wins over exclude, exclude wins over image
// directories.
func Classify(files []string, rules Rules) (Classification, error) {
	if err := ValidatePatterns(rules.Exclude); err != nil {
		return Classification{}, err
	}
	if err := ValidatePatterns(rules.TreatAsGlobal); err != nil {
		return Classification{}, err
	}

	var result Classification
	changed := make(map[string]bool)
	for _, file := range files {
		file = normalize(file)
		if file == "" {
			continue
		}
		if matchAny(rules.TreatAsGlobal, file) {
			result.markGlobal(file)
			continue
		}
		if matchAny(rules.Exclude, file) {
			continue
		}
		if names := imagesOwning(rules.ImageDirs, file); len(names) > 0 {
			for _, name := range names {
				changed[name] = true
			}
			continue
		}
		// Files outside the project only count inside an image directory.
		if file == ".." || strings.HasPrefix(file, "../") {
			continue
		}
		if rules.Unmatched != UnmatchedIgnore {
			result.markGlobal(file)
		}
	}

	for name := range changed {
		result.Changed = append(result.Changed, name)
	}
	slices.Sort(result.Changed)
	return result, nil
}

func (c *Classification) markGlobal(file string) {
	if c.Global {
		return
	}
	c.Global = true
	c.GlobalFile = file
}

func normalize(file string) string {
	file = strings.TrimSpace(file)
	if file == "" {
		return ""
	}
	return strings.TrimPrefix(path.Clean(file), "./")
}

func matchAny(patterns []string, file string) bool {
	for _, pattern := range patterns {
		if ok, _ := doublestar.Match(pattern, file); ok {
			return true
		}
	}
	return false
}

func imagesOwning(imageDirs map[string][]string, file string) []string {
	var names []string
	for name, dirs := range imageDirs {
		for _, dir := range dirs {
			if file == dir || strings.HasPrefix(file, dir+"/") {
				names = append(names, name)
				break
			}
		}
	}
	return names
}
