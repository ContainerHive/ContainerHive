package discovery

import (
	"errors"

	"github.com/ContainerHive/ContainerHive/internal/file_resolver"
)

var testConfigFileNames = file_resolver.GetFileCandidates("test", "yml", "yaml")

func getTestConfigFilePath(root string) (string, error) {
	path, err := file_resolver.ResolveFirstExistingFile(root, testConfigFileNames...)
	if err != nil && errors.Is(file_resolver.NoFileCandidatesErr, err) {
		return "", nil
	}

	return path, err
}
