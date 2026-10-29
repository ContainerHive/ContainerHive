package buildkit

import "strings"

// DefaultImage is the default BuildKit container image name. It is the
// ContainerHive helper image, which tracks the upstream moby/buildkit releases.
const DefaultImage = "ghcr.io/containerhive/helper-images/buildkit"

// DefaultTag is the tag of DefaultImage matching Version. The helper image is
// tagged per release series (e.g. "v0.34") and points to the latest patch.
var DefaultTag = releaseSeries(Version)

// TagFor returns the default tag for the given image. Only DefaultImage is
// tagged per release series, any other image (e.g. a moby/buildkit mirror) is
// expected to carry the exact Version tag.
func TagFor(image string) string {
	if image == DefaultImage {
		return DefaultTag
	}
	return Version
}

// ImageRef returns the full "image:tag" reference for the buildkitd container.
// It resolves ci_buildkit_image and ci_buildkit_version from templateOpts,
// falling back to DefaultImage and the matching default tag respectively.
func ImageRef(templateOpts map[string]string) string {
	image := DefaultImage
	if v := templateOpts["ci_buildkit_image"]; v != "" {
		image = v
	}
	version := TagFor(image)
	if v := templateOpts["ci_buildkit_version"]; v != "" {
		version = v
	}
	return image + ":" + version
}

// releaseSeries strips the patch level from a version, e.g. "v0.34.0" -> "v0.34".
func releaseSeries(version string) string {
	parts := strings.SplitN(version, ".", 3)
	if len(parts) < 3 {
		return version
	}
	return parts[0] + "." + parts[1]
}
