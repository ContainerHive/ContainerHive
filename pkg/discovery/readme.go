package discovery

import (
	"errors"

	"github.com/ContainerHive/ContainerHive/internal/file_resolver"
)

var readmeFileNames = file_resolver.GetFileCandidates("README.md")

func getReadmePath(root string) (string, error) {
	path, err := file_resolver.ResolveFirstExistingFile(root, readmeFileNames...)
	if err != nil && errors.Is(err, file_resolver.NoFileCandidatesErr) {
		return "", nil
	}
	return path, err
}
