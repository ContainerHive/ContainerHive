package platform

import "strings"

// DefaultPlatforms is the set of platforms used when none are configured.
var DefaultPlatforms = []string{"linux/amd64", "linux/arm64"}

// Sanitize converts a platform string like "linux/amd64" to "linux-amd64".
func Sanitize(p string) string {
	return strings.ReplaceAll(p, "/", "-")
}

// Resolve returns the most specific platform list: variant > image > global.
// If the most specific level is empty, it falls back to the next level.
func Resolve(global, image, variant []string) []string {
	if len(variant) > 0 {
		return variant
	}
	if len(image) > 0 {
		return image
	}
	return global
}
