package buildkit

// DefaultImage is the default BuildKit container image name.
const DefaultImage = "moby/buildkit"

// ImageRef returns the full "image:tag" reference for the buildkitd container.
// It resolves ci_buildkit_image and ci_buildkit_version from templateOpts,
// falling back to DefaultImage and Version respectively.
func ImageRef(templateOpts map[string]string) string {
	image, version := DefaultImage, Version
	if v := templateOpts["ci_buildkit_image"]; v != "" {
		image = v
	}
	if v := templateOpts["ci_buildkit_version"]; v != "" {
		version = v
	}
	return image + ":" + version
}
