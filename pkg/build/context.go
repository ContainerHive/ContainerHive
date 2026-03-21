package build

import (
	"path/filepath"

	"github.com/timo-reymann/ContainerHive/pkg/platform"
)

// TarFilePath returns the OCI tar output path for a given image tag and platform
// inside the rendered dist directory.
// The layout is: dist/<name>/<tag>/<sanitized-platform>/image.tar
func TarFilePath(distPath, name, tag, platformStr string) string {
	return filepath.Join(distPath, name, tag, platform.Sanitize(platformStr), "image.tar")
}

// WithBuildID appends an optional build ID suffix to a tag.
// Returns the tag unchanged if buildID is empty.
func WithBuildID(tag, buildID string) string {
	if buildID != "" {
		return tag + "." + buildID
	}
	return tag
}

// PushTag returns the tag to use when pushing to the registry, with platform
// suffix and optional build-id suffix.
// Format: tagName.sanitized-platform[.buildID]
func PushTag(tagName, platformStr, buildID string) string {
	return WithBuildID(tagName+"."+platform.Sanitize(platformStr), buildID)
}
