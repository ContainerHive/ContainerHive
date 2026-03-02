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
