package build

import (
	"path/filepath"

	"github.com/ContainerHive/ContainerHive/pkg/platform"
)

// TarFilePath returns the OCI tar output path for a given image tag and platform
// inside the rendered dist directory.
// The layout is: dist/<name>/<tag>/<sanitized-platform>/image.tar
func TarFilePath(distPath, name, tag, platformStr string) string {
	return filepath.Join(distPath, name, tag, platform.Sanitize(platformStr), "image.tar")
}

// buildIDSeparator joins a tag and a build ID. The "-build." infix keeps
// suffixed tags unambiguous: "26.06-build.22" cannot be mistaken for a
// patch version "26.06.22".
const buildIDSeparator = "-build."

// WithBuildID appends an optional "-build.<id>" suffix to a tag.
// Returns the tag unchanged if buildID is empty.
func WithBuildID(tag, buildID string) string {
	if buildID == "" {
		return tag
	}
	return tag + buildIDSeparator + buildID
}

// PushTag returns the tag to use when pushing to the registry, with platform
// suffix and optional build-id suffix.
// Format: tagName.sanitized-platform[-build.<buildID>]
func PushTag(tagName, platformStr, buildID string) string {
	return WithBuildID(tagName+"."+platform.Sanitize(platformStr), buildID)
}
