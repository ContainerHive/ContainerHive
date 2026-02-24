package discovery

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/timo-reymann/ContainerHive/pkg/model"
	"gopkg.in/yaml.v3"
)

var hiveConfigFileNames = []string{
	"hive.yaml",
	"hive.yml",
	"container-hive.yaml",
	"container-hive.yml",
}

func getContainerHiveConfigFile(root string) (string, error) {
	for _, name := range hiveConfigFileNames {
		path := filepath.Join(root, name)
		_, err := os.Stat(path)
		if err != nil && !os.IsNotExist(err) {
			return "", errors.Join(errors.New("failed to state ContainerHive config file path "+path), err)
		}

		if err == nil {
			return path, nil
		}
	}

	return "", errors.New("no ContainerHive config file found")
}

func parseHiveConfigFile(path string) (*model.HiveProjectConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.Join(errors.New("failed to read ContainerHive config file"), err)
	}

	var config model.HiveProjectConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, errors.Join(errors.New("failed to parse ContainerHive config file"), err)
	}

	return &config, nil
}
