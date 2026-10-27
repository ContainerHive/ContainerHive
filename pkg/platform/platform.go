package platform

import "strings"

// DefaultPlatforms is the set of platforms used when none are configured.
var DefaultPlatforms = []string{"linux/amd64", "linux/arm64"}

// Normalize ensures a platform string is in canonical os/arch form.
// If only an architecture is given (no "/"), "linux/" is prepended as the
// default OS for container images.
func Normalize(p string) string {
	if !strings.Contains(p, "/") {
		return "linux/" + p
	}
	return p
}

// Short returns the shortest form of a platform that Normalize maps back to
// the same value: the default "linux/" prefix is dropped when only an
// architecture remains (e.g. "linux/amd64" -> "amd64"). Multi-component or
// non-linux platforms such as "linux/arm/v7" or "windows/amd64" are returned
// unchanged.
func Short(p string) string {
	arch, ok := strings.CutPrefix(p, "linux/")
	if !ok || strings.Contains(arch, "/") {
		return p
	}
	return arch
}

// Sanitize converts a platform string like "linux/amd64" to "linux-amd64".
func Sanitize(p string) string {
	return strings.ReplaceAll(p, "/", "-")
}

// Resolve returns the platform list to use. The configured list is the most
// specific non-empty level: variant > image > global. override represents an
// explicit CLI --platform flag and, when set, selects within the configured
// list: platforms the variant/image/project does not support are dropped, so
// the result may be empty. The configured order is preserved and duplicates
// (e.g. "amd64" and "linux/amd64") are dropped.
// All returned platform strings are normalized to canonical os/arch form.
func Resolve(override, global, image, variant []string) []string {
	var configured []string
	if len(variant) > 0 {
		configured = variant
	} else if len(image) > 0 {
		configured = image
	} else {
		configured = global
	}

	normalized := make([]string, 0, len(configured))
	seen := make(map[string]bool, len(configured))
	for _, p := range configured {
		p = Normalize(p)
		if seen[p] {
			continue
		}
		seen[p] = true
		normalized = append(normalized, p)
	}
	if len(override) == 0 {
		return normalized
	}

	requested := make(map[string]bool, len(override))
	for _, p := range override {
		requested[Normalize(p)] = true
	}
	selected := make([]string, 0, len(normalized))
	for _, p := range normalized {
		if requested[p] {
			selected = append(selected, p)
		}
	}
	return selected
}

// Unsupported returns the requested platforms that none of the configured
// lists contain, in requested order. Both sides are normalized before
// comparing, the returned values are kept as requested.
func Unsupported(requested []string, configured ...[]string) []string {
	known := make(map[string]bool)
	for _, list := range configured {
		for _, p := range list {
			known[Normalize(p)] = true
		}
	}

	var unsupported []string
	for _, p := range requested {
		if !known[Normalize(p)] {
			unsupported = append(unsupported, p)
		}
	}
	return unsupported
}
