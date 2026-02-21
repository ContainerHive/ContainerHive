package build

import (
	"os"
	"path/filepath"

	"github.com/timo-reymann/ContainerHive/internal/buildkit/build_context"
)

// PatchHiveRefs rewrites __hive__/ references in a Dockerfile to point to the
// given registry address. It returns the patched file path and a cleanup
// function that removes the patched file.
func PatchHiveRefs(dockerfilePath, registryAddr string) (string, func(), error) {
	patched := dockerfilePath + ".patched"
	if err := build_context.RewriteHiveRefs(dockerfilePath, patched, registryAddr); err != nil {
		return "", nil, err
	}
	return patched, func() { os.Remove(patched) }, nil
}

// TarFilePath returns the OCI tar output path for a given image tag inside the
// rendered dist directory.
func TarFilePath(distPath, name, tag string) string {
	return filepath.Join(distPath, name, tag, "image.tar")
}
