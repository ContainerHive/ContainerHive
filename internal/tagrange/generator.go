package tagrange

import (
	"fmt"

	"github.com/Masterminds/semver/v3"

	"github.com/ContainerHive/ContainerHive/pkg/model"
)

// defaultMaxTags is the cap on generated tags per range when TagRange.MaxTags
// is unset. It exists so a misconfigured select/filter (or a source that
// starts returning far more versions than expected) fails loudly instead of
// silently generating an enormous build matrix.
const defaultMaxTags = 50

// RawVersion is one version as reported by a VersionSource, before parsing.
// Defined here (rather than imported from the source package) to keep this
// package's public surface independent of which sources exist.
type RawVersion struct {
	Raw        string
	Prerelease bool
	Extra      map[string]string
}

// GeneratedTag is one tag produced by resolving a tag_range.
type GeneratedTag struct {
	Name         string
	Versions     model.Versions
	BuildArgs    model.BuildArgs
	Labels       map[string]string
	IsPrerelease bool

	// Range identifies which tag_range produced this tag (the rangeLabel
	// passed to GenerateTags, which includes a per-range index), so a
	// caller merging tags from multiple ranges can name both ranges in a
	// cross-range collision error even when two ranges share a tag_name
	// template.
	Range string
}

// GenerateTags runs the full per-range pipeline — parse, filter, select,
// render, dedupe, cap — over already-fetched raw versions. rangeLabel
// identifies the range in error messages (e.g. the image name).
func GenerateTags(rangeLabel string, tr *model.TagRange, raw []RawVersion) ([]GeneratedTag, error) {
	if tr.TagName == "" {
		return nil, fmt.Errorf("tag_range %q: tag_name is required", rangeLabel)
	}

	candidates, unparsed := parseCandidates(raw)
	if len(candidates) == 0 {
		return nil, fmt.Errorf("tag_range %q: none of %d fetched version(s) parsed as semantic versions (examples: %v)", rangeLabel, len(raw), firstN(unparsed, 5))
	}

	filtered, err := applyFilters(candidates, tr.Filter)
	if err != nil {
		return nil, fmt.Errorf("tag_range %q: %w", rangeLabel, err)
	}

	sortDescending(filtered)
	selected := selectVersions(filtered, tr.Select)

	tags, err := renderTags(rangeLabel, tr, selected)
	if err != nil {
		return nil, err
	}

	maxTags := tr.MaxTags
	if maxTags <= 0 {
		maxTags = defaultMaxTags
	}
	if len(tags) > maxTags {
		return nil, fmt.Errorf("tag_range %q: generated %d tags, exceeding max_tags=%d; narrow select or raise max_tags", rangeLabel, len(tags), maxTags)
	}

	return tags, nil
}

// parseCandidates parses raw versions as semver, skipping ones that don't
// parse (registry tag lists routinely contain non-version tags like "latest"
// or "alpine") and returning their raw strings for the caller's error
// message if nothing parses at all.
func parseCandidates(raw []RawVersion) ([]candidate, []string) {
	candidates := make([]candidate, 0, len(raw))
	var unparsed []string
	for _, v := range raw {
		ver, err := semver.NewVersion(v.Raw)
		if err != nil {
			unparsed = append(unparsed, v.Raw)
			continue
		}
		candidates = append(candidates, candidate{
			ver:        ver,
			raw:        v.Raw,
			prerelease: v.Prerelease || ver.Prerelease() != "",
			extra:      v.Extra,
		})
	}
	return candidates, unparsed
}

// renderTags renders tag_name/versions/build_args/labels for each selected
// candidate, in descending-version order, and dedupes on the rendered tag
// name: since selected is descending, the first (highest version) render of
// a given name wins. This is by design, not an edge case — a tag_name
// template coarser than the select granularity (e.g. "{{.major}}.{{.minor}}"
// with patch: {last: 2}) collides on purpose, and the template is what sets
// the granularity.
func renderTags(rangeLabel string, tr *model.TagRange, selected []candidate) ([]GeneratedTag, error) {
	seen := make(map[string]struct{}, len(selected))
	tags := make([]GeneratedTag, 0, len(selected))
	for _, c := range selected {
		name, err := renderTagName(rangeLabel, tr.TagName, c)
		if err != nil {
			return nil, fmt.Errorf("tag_range %q: %w", rangeLabel, err)
		}
		if _, dup := seen[name]; dup {
			continue
		}
		seen[name] = struct{}{}

		versions, err := renderValues(rangeLabel, "versions", tr.Versions, c)
		if err != nil {
			return nil, fmt.Errorf("tag_range %q: %w", rangeLabel, err)
		}
		buildArgs, err := renderValues(rangeLabel, "build_args", tr.BuildArgs, c)
		if err != nil {
			return nil, fmt.Errorf("tag_range %q: %w", rangeLabel, err)
		}
		labels, err := renderValues(rangeLabel, "labels", tr.Labels, c)
		if err != nil {
			return nil, fmt.Errorf("tag_range %q: %w", rangeLabel, err)
		}

		tags = append(tags, GeneratedTag{
			Name:         name,
			Versions:     model.Versions(versions),
			BuildArgs:    model.BuildArgs(buildArgs),
			Labels:       labels,
			IsPrerelease: c.prerelease,
			Range:        rangeLabel,
		})
	}
	return tags, nil
}

func firstN(s []string, n int) []string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
