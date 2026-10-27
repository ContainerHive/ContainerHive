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

// Sanitize converts a platform string like "linux/amd64" to "linux-amd64".
func Sanitize(p string) string {
	return strings.ReplaceAll(p, "/", "-")
}

// Resolve returns the platform list to use. The configured list is the most
// specific non-empty level: variant > image > global. override represents an
// explicit CLI --platform flag and, when set, selects within the configured
// list: platforms the variant/image/project does not support are dropped, so
// the result may be empty. The configured order is preserved.
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

	normalized := make([]string, len(configured))
	for i, p := range configured {
		normalized[i] = Normalize(p)
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
