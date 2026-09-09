package tagrange

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Masterminds/semver/v3"

	"github.com/ContainerHive/ContainerHive/pkg/model"
)

// applyFilters narrows candidates before selection, in a fixed order:
// exclude_prerelease, exclude_suffixes, min/max_version, then
// max_major_increment. Returns an error if filtering removes every
// candidate.
func applyFilters(candidates []candidate, filter *model.FilterConfig) ([]candidate, error) {
	excludePrerelease := true
	var suffixes []string
	var minVersion, maxVersion *semver.Version
	maxMajorIncrement := 0

	if filter != nil {
		if filter.ExcludePrerelease != nil {
			excludePrerelease = *filter.ExcludePrerelease
		}
		suffixes = filter.ExcludeSuffixes
		maxMajorIncrement = filter.MaxMajorIncrement

		var err error
		if filter.MinVersion != "" {
			if minVersion, err = semver.NewVersion(filter.MinVersion); err != nil {
				return nil, fmt.Errorf("invalid filter.min_version %q: %w", filter.MinVersion, err)
			}
		}
		if filter.MaxVersion != "" {
			if maxVersion, err = semver.NewVersion(filter.MaxVersion); err != nil {
				return nil, fmt.Errorf("invalid filter.max_version %q: %w", filter.MaxVersion, err)
			}
		}
	}

	filtered := make([]candidate, 0, len(candidates))
	for _, c := range candidates {
		if excludePrerelease && c.prerelease {
			continue
		}
		if hasAnySuffix(c.raw, suffixes) {
			continue
		}
		if minVersion != nil && c.ver.Compare(minVersion) < 0 {
			continue
		}
		if maxVersion != nil && c.ver.Compare(maxVersion) > 0 {
			continue
		}
		filtered = append(filtered, c)
	}

	filtered = applyMaxMajorIncrement(filtered, maxMajorIncrement)

	if len(filtered) == 0 {
		return nil, fmt.Errorf("no candidates left after filtering %d input version(s)", len(candidates))
	}
	return filtered, nil
}

func hasAnySuffix(raw string, suffixes []string) bool {
	for _, s := range suffixes {
		if strings.Contains(raw, s) {
			return true
		}
	}
	return false
}

// applyMaxMajorIncrement is a sanity guard against outlier majors (a rogue
// nightly channel, a calver-style entry mixed into a semver feed), not a
// window: majors are sorted ascending and, starting from the lowest, the
// first adjacent gap larger than max wipes out that major and every major
// above it. max <= 0 disables the guard.
func applyMaxMajorIncrement(candidates []candidate, max int) []candidate {
	if max <= 0 || len(candidates) == 0 {
		return candidates
	}

	majors := distinctMajorsAscending(candidates)
	cutoff := int64(-1) // majors >= cutoff are dropped; -1 means none dropped
	for i := 1; i < len(majors); i++ {
		if majors[i]-majors[i-1] > int64(max) {
			cutoff = majors[i]
			break
		}
	}
	if cutoff < 0 {
		return candidates
	}

	kept := make([]candidate, 0, len(candidates))
	for _, c := range candidates {
		if int64(c.ver.Major()) < cutoff {
			kept = append(kept, c)
		}
	}
	return kept
}

func distinctMajorsAscending(candidates []candidate) []int64 {
	seen := make(map[int64]struct{})
	for _, c := range candidates {
		seen[int64(c.ver.Major())] = struct{}{}
	}
	majors := make([]int64, 0, len(seen))
	for m := range seen {
		majors = append(majors, m)
	}
	sort.Slice(majors, func(i, j int) bool { return majors[i] < majors[j] })
	return majors
}
