package docker

import (
	"github.com/ContainerHive/ContainerHive/internal/utils"
)

func extractTar(tarPath, destDir string) error {
	return utils.ExtractTar(tarPath, destDir)
}
