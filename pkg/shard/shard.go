// Package shard partitions the tags of a ContainerHive project across a fixed
// number of CI shards so that build, sbom and test can each run their slice
// in a separate parallel job. The shard unit is the individual tag (base or
// variant) so that images with many variants can be split as finely as
// possible - this is what keeps per-job SBOM artifact sizes under CI
// providers' size limits.
package shard

import (
	"fmt"
	"hash/fnv"
	"sort"

	"github.com/ContainerHive/ContainerHive/pkg/model"
)

// Shard identifies one partition out of a fixed total, 1-based to match
// GitLab's CI_NODE_INDEX.
type Shard struct {
	Current int
	Max     int
}

// Enabled reports whether sharding is active. Max <= 1 means every shard
// owns everything.
func (s Shard) Enabled() bool {
	return s.Max > 1
}

// Owns reports whether the given unit belongs to this shard. Assignment
// hashes the unit's own identity rather than its position in the canonical
// TagIndex, so inserting or removing one tag only moves that tag between
// shards instead of reshuffling the whole index (as position-based modulo
// assignment did).
func (s Shard) Owns(ref TagRef) bool {
	if !s.Enabled() {
		return true
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(ref.Identifier))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(ref.TagName))
	return int(h.Sum32()%uint32(s.Max)) == s.Current-1
}

// Validate checks that Max and Current describe a well-formed shard.
func (s Shard) Validate() error {
	if s.Max < 1 {
		return fmt.Errorf("max shards must be >= 1, got %d", s.Max)
	}
	if s.Current < 1 || s.Current > s.Max {
		return fmt.Errorf("current shard must be between 1 and %d, got %d", s.Max, s.Current)
	}
	return nil
}

// TagRef identifies a single shard unit: one tag (base or variant) of one
// image, addressed by the image's unique identifier.
type TagRef struct {
	Identifier string
	TagName    string
}

// TagIndex returns the canonical, sorted list of shard units for a project:
// one entry per (image identifier, tag), including variant-suffixed tags.
// The list is sorted by (Identifier, TagName) and deduplicated so that
// UnitCountByName and hash-based Owns both see each unit exactly once,
// regardless of Go's randomized map iteration order.
func TagIndex(project *model.ContainerHiveProject) []TagRef {
	seen := make(map[TagRef]struct{})
	for _, img := range project.ImagesByIdentifier {
		for tagName := range img.Tags {
			seen[TagRef{Identifier: img.Identifier, TagName: tagName}] = struct{}{}
			for _, variantDef := range img.Variants {
				seen[TagRef{Identifier: img.Identifier, TagName: tagName + variantDef.TagSuffix}] = struct{}{}
			}
		}
	}
	refs := make([]TagRef, 0, len(seen))
	for ref := range seen {
		refs = append(refs, ref)
	}
	sort.Slice(refs, func(i, j int) bool {
		if refs[i].Identifier != refs[j].Identifier {
			return refs[i].Identifier < refs[j].Identifier
		}
		return refs[i].TagName < refs[j].TagName
	})
	return refs
}

// tagSet builds a lookup of every valid shard unit for the project, so an
// unknown (identifier, tag) pair is never treated as owned.
func tagSet(refs []TagRef) map[TagRef]struct{} {
	set := make(map[TagRef]struct{}, len(refs))
	for _, ref := range refs {
		set[ref] = struct{}{}
	}
	return set
}

// NewTagSharder returns a predicate reporting whether this shard owns the
// exact given tag (base or variant) of the given image identifier. Returns a
// match-all predicate when sharding is disabled.
func NewTagSharder(project *model.ContainerHiveProject, s Shard) func(identifier, tagName string) bool {
	if !s.Enabled() {
		return func(string, string) bool { return true }
	}
	valid := tagSet(TagIndex(project))
	return func(identifier, tagName string) bool {
		ref := TagRef{Identifier: identifier, TagName: tagName}
		if _, ok := valid[ref]; !ok {
			return false
		}
		return s.Owns(ref)
	}
}

// NewBaseTagSharder returns a predicate reporting whether this shard must
// build the given base tag: true when the shard owns the base tag itself, or
// owns any variant derived from it. A base tag may therefore be built by more
// than one shard - cheap, since BuildKit caches it and the resulting push is
// content-identical - which keeps a variant's FROM source available locally
// without ever widening what sbom/test consider "owned" by this shard.
// Returns a match-all predicate when sharding is disabled.
func NewBaseTagSharder(project *model.ContainerHiveProject, s Shard) func(identifier, baseTagName string) bool {
	if !s.Enabled() {
		return func(string, string) bool { return true }
	}
	valid := tagSet(TagIndex(project))

	// variantsOf maps identifier -> base tag -> that base tag's variant tag names.
	variantsOf := make(map[string]map[string][]string)
	for _, img := range project.ImagesByIdentifier {
		if len(img.Variants) == 0 {
			continue
		}
		perBase := make(map[string][]string, len(img.Tags))
		for tagName := range img.Tags {
			for _, variantDef := range img.Variants {
				perBase[tagName] = append(perBase[tagName], tagName+variantDef.TagSuffix)
			}
		}
		variantsOf[img.Identifier] = perBase
	}

	owns := func(identifier, tagName string) bool {
		ref := TagRef{Identifier: identifier, TagName: tagName}
		if _, ok := valid[ref]; !ok {
			return false
		}
		return s.Owns(ref)
	}

	return func(identifier, baseTagName string) bool {
		if owns(identifier, baseTagName) {
			return true
		}
		for _, variantTag := range variantsOf[identifier][baseTagName] {
			if owns(identifier, variantTag) {
				return true
			}
		}
		return false
	}
}

// UnitCountByName returns the number of shard units per image name. A unit is
// one (identifier, tag) pair including variant-suffixed tags, matching
// TagIndex. This function iterates ImagesByName so it works even when
// ImagesByIdentifier is not populated (e.g. test fixtures).
func UnitCountByName(project *model.ContainerHiveProject) map[string]int {
	counts := make(map[string]int, len(project.ImagesByName))
	for name, images := range project.ImagesByName {
		for _, img := range images {
			counts[name] += len(img.Tags) * (1 + len(img.Variants))
		}
	}
	return counts
}

// OwnedUnits returns the shard units this shard owns. Used by callers to
// detect an empty slice (e.g. more shards configured than units exist) so
// they can log and exit cleanly rather than silently doing nothing.
func OwnedUnits(project *model.ContainerHiveProject, s Shard) []TagRef {
	refs := TagIndex(project)
	if !s.Enabled() {
		return refs
	}
	owned := make([]TagRef, 0, len(refs))
	for _, ref := range refs {
		if s.Owns(ref) {
			owned = append(owned, ref)
		}
	}
	return owned
}
