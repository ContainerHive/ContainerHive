package semantic_tags

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Pattern: optional prefix + major.minor.patch.build + optional suffix (with -) + optional build metadata (with +)
// Where prefix can be any non-digit characters at the start
// All version components except major are optional
// Supports formats like: 1.2.3, 1.2.3.4, v1.2.3, 1.2.3-alpine, version1.2.3+build123, etc.
var versionPattern = regexp.MustCompile(`^([^\d]*)(\d+)(?:\.(\d+))?(?:\.(\d+))?(?:\.(\d+))?(?:-([a-zA-Z0-9][a-zA-Z0-9.\-]*))?(?:\+([^\s]+))?$`)

type SemanticTagVersion struct {
	Prefix string
	Major  string
	Minor  string
	Patch  string
	Build  string
	Suffix string
}

func NewSemanticVersion(tag string) (*SemanticTagVersion, error) {
	// Remove any leading/trailing whitespace
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return nil, fmt.Errorf("empty tag")
	}

	matches := versionPattern.FindStringSubmatch(tag)

	if matches == nil {
		return nil, fmt.Errorf("invalid version format: %s", tag)
	}

	// Extract components
	prefix := matches[1]
	major := matches[2]
	minor := matches[3]
	patch := matches[4]
	build := matches[5]
	suffix := matches[6]
	metadata := matches[7]

	// Validate that we have at least major version
	if major == "" {
		return nil, fmt.Errorf("missing major version in: %s", tag)
	}

	// If we have metadata (build info with +), use that for Build field
	// Otherwise, use the 4th version component if present
	finalBuild := metadata
	if finalBuild == "" {
		finalBuild = build
	}

	return &SemanticTagVersion{
		Prefix: prefix,
		Major:  major,
		Minor:  minor,
		Patch:  patch,
		Build:  finalBuild,
		Suffix: suffix,
	}, nil
}

// GetLowerVariants returns all the "lower" variants of this semantic version
// For example, version "1.2.3" returns ["1.2", "1"]
// Version "1.2.3.4" returns ["1.2.3", "1.2", "1"]
func (v *SemanticTagVersion) GetLowerVariants() []string {
	variants := []string{}

	suffix := ""
	if v.Suffix != "" {
		suffix = "-" + v.Suffix
	}

	buildVariant := func(parts ...string) string {
		return v.Prefix + strings.Join(parts, ".") + suffix
	}

	if v.Build != "" {
		variants = append(variants, buildVariant(v.Major, v.Minor, v.Patch))
		variants = append(variants, buildVariant(v.Major, v.Minor))
		variants = append(variants, buildVariant(v.Major))
	} else if v.Patch != "" {
		variants = append(variants, buildVariant(v.Major, v.Minor))
		variants = append(variants, buildVariant(v.Major))
	} else if v.Minor != "" {
		variants = append(variants, buildVariant(v.Major))
	}

	return variants
}

// compareNumeric compares two numeric version components
// Returns -1 if a < b, 0 if a == b, 1 if a > b
// Empty string is considered less than any non-empty string
func compareNumeric(a, b string) int {
	if a == "" && b != "" {
		return -1
	}
	if a != "" && b == "" {
		return 1
	}
	if a != "" && b != "" {
		num1, _ := strconv.Atoi(a)
		num2, _ := strconv.Atoi(b)
		if num1 < num2 {
			return -1
		}
		if num1 > num2 {
			return 1
		}
	}
	return 0
}

// Compare compares this version with another version
// Returns:
// -1 if this version is less than the other
//
//	0 if this version is equal to the other
//	1 if this version is greater than the other
func (v *SemanticTagVersion) Compare(other *SemanticTagVersion) int {
	// Compare major, minor, patch
	if result := compareNumeric(v.Major, other.Major); result != 0 {
		return result
	}
	if result := compareNumeric(v.Minor, other.Minor); result != 0 {
		return result
	}
	if result := compareNumeric(v.Patch, other.Patch); result != 0 {
		return result
	}

	// Compare build (string comparison)
	if v.Build == "" && other.Build != "" {
		return -1
	}
	if v.Build != "" && other.Build == "" {
		return 1
	}
	if v.Build < other.Build {
		return -1
	}
	if v.Build > other.Build {
		return 1
	}

	return 0
}

// Less returns true if this version is less than the other version
func (v *SemanticTagVersion) Less(other *SemanticTagVersion) bool {
	return v.Compare(other) < 0
}

// Greater returns true if this version is greater than the other version
func (v *SemanticTagVersion) Greater(other *SemanticTagVersion) bool {
	return v.Compare(other) > 0
}

// Equal returns true if this version is equal to the other version
func (v *SemanticTagVersion) Equal(other *SemanticTagVersion) bool {
	return v.Compare(other) == 0
}

// String returns the string representation of the semantic version
func (v *SemanticTagVersion) String() string {
	versionStr := v.Prefix + v.Major

	if v.Minor != "" {
		versionStr += "." + v.Minor
	}

	if v.Patch != "" {
		versionStr += "." + v.Patch
	}

	if v.Build != "" {
		versionStr += "." + v.Build
	}

	if v.Suffix != "" {
		versionStr += "-" + v.Suffix
	}

	return versionStr
}
