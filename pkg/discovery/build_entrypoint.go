package discovery

import (
	"github.com/ContainerHive/ContainerHive/internal/file_resolver"
)

var dockerfileConfigFileNames = file_resolver.GetFileCandidates("Dockerfile")

func getBuildEntrypointPath(root string) (string, error) {
	return file_resolver.ResolveFirstExistingFile(root, dockerfileConfigFileNames...)
}
