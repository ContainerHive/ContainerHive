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

// Resolve returns the platform list to use, in precedence order:
// override > variant > image > global. override represents an explicit
// CLI --platform flag and, when set, always wins regardless of what
// hive.yml configures at the variant/image/global level.
// If the most specific level is empty, it falls back to the next level.
// All returned platform strings are normalized to canonical os/arch form.
func Resolve(override, global, image, variant []string) []string {
	var result []string
	if len(override) > 0 {
		result = override
	} else if len(variant) > 0 {
		result = variant
	} else if len(image) > 0 {
		result = image
	} else {
		result = global
	}

	normalized := make([]string, len(result))
	for i, p := range result {
		normalized[i] = Normalize(p)
	}
	return normalized
}
