// Package tagrange resolves image.yml tag_ranges into concrete tags by
// fetching candidate versions from an external source, filtering and
// selecting a subset, and rendering the tag name and per-tag values from Go
// templates.
package tagrange

import (
	"sort"

	"github.com/Masterminds/semver/v3"
)

// candidate is one upstream version, parsed and carrying its source metadata.
type candidate struct {
	ver *semver.Version

	// raw is the version exactly as the source reported it (may carry a "v"
	// prefix or build metadata the source's own API does not).
	raw string

	// prerelease is true when either the source itself declared the version
	// a prerelease (e.g. a GitHub release marked prerelease) or the version
	// string parses with a semver prerelease component. It is never
	// re-derived from anything the tag_name template renders: a tag string
	// alone cannot tell a prerelease apart from a flavour suffix like
	// "-alpine".
	prerelease bool

	// extra carries source-provided metadata, exposed to templates as
	// .extra.<key>.
	extra map[string]string
}

// sortDescending sorts candidates from highest to lowest version. raw is a
// tie-breaker so build-metadata-only differences (which semver.Version.Compare
// treats as equal) still order deterministically regardless of the order the
// source returned them in.
func sortDescending(candidates []candidate) {
	sort.SliceStable(candidates, func(i, j int) bool {
		if c := candidates[i].ver.Compare(candidates[j].ver); c != 0 {
			return c > 0
		}
		return candidates[i].raw > candidates[j].raw
	})
}
