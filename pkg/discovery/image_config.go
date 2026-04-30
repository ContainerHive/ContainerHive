package discovery

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/ContainerHive/ContainerHive/internal/file_resolver"
	"github.com/ContainerHive/ContainerHive/pkg/model"
	"gopkg.in/yaml.v3"
)

func ensureSecretsInitialized(secrets model.Secrets) model.Secrets {
	if secrets == nil {
		return make(model.Secrets)
	}
	return secrets
}

const rootFsDirName = "rootfs"

var imageConfigFileNames = []string{
	"image.yaml",
	"image.yml",
}

func getRoofsPath(imageRoot string) (string, error) {
	stat, err := os.Stat(filepath.Join(imageRoot, rootFsDirName))
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}

		return "", errors.Join(errors.New("failed to stat rootfs dir"), err)
	}

	if !stat.IsDir() {
		return "", errors.New("rootfs dir is not a directory")
	}

	return filepath.Join(imageRoot, rootFsDirName), nil
}

func parseImageConfigFile(configFilePath string) (*model.ImageDefinitionConfig, error) {
	f, err := os.Open(configFilePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	d := yaml.NewDecoder(f)
	d.KnownFields(true)
	var config model.ImageDefinitionConfig
	if err := d.Decode(&config); err != nil {
		return nil, err
	}
	return &config, nil
}

func processImageConfig(projectRoot, configFilePath string) (*model.Image, error) {
	imageRoot := filepath.Dir(configFilePath)
	relativeRoot, err := filepath.Rel(projectRoot, imageRoot)
	if err != nil {
		return nil, err
	}

	testConfigFilePath, err := getTestConfigFilePath(imageRoot)
	if err != nil {
		return nil, errors.Join(errors.New("failed to discover test config file"), err)
	}

	isNested := strings.ContainsRune(relativeRoot, os.PathSeparator)
	var name string
	if isNested {
		name = filepath.Dir(relativeRoot)
	} else {
		name = relativeRoot
	}

	parsedImageDef, err := parseImageConfigFile(configFilePath)
	if err != nil {
		return nil, errors.Join(errors.New("failed to parse image config file"), err)
	}

	rootFsPath, err := getRoofsPath(imageRoot)
	if err != nil {
		return nil, errors.Join(errors.New("failed to discover rootfs dir"), err)
	}

	parentIcon := ""
	if parsedImageDef.Report.Icon != nil {
		parentIcon = *parsedImageDef.Report.Icon
	}

	description := parsedImageDef.Description

	indexedVariants, err := processVariants(parsedImageDef, imageRoot, parentIcon)
	if err != nil {
		return nil, err
	}

	dockerfilePath, err := getBuildEntrypointPath(imageRoot)
	if err != nil {
		return nil, errors.Join(errors.New("failed to discover Dockerfile"), err)
	}

	readmePath, err := getReadmePath(imageRoot)
	if err != nil {
		return nil, errors.Join(errors.New("failed to discover README"), err)
	}

	return &model.Image{
		RootDir:             filepath.Join(projectRoot, relativeRoot),
		BuildEntryPointPath: dockerfilePath,
		ReadmePath:          readmePath,
		RootFSDir:           rootFsPath,
		Identifier:          relativeRoot,
		Name:                name,
		Description:         description,
		TestConfigFilePath:  testConfigFilePath,
		DefinitionFilePath:  configFilePath,
		Versions:            parsedImageDef.Versions,
		BuildArgs:           parsedImageDef.BuildArgs,
		Secrets:             ensureSecretsInitialized(parsedImageDef.Secrets),
		Variants:            indexedVariants,
		Tags:                processTags(parsedImageDef),
		DependsOn:           parsedImageDef.DependsOn,
		Platforms:           parsedImageDef.Platforms,
		LatestAlias:         parsedImageDef.LatestAlias,
		Report:              model.ReportModel{Icon: parentIcon},
		Labels:              parsedImageDef.Labels,
	}, nil
}

func processTags(imageDef *model.ImageDefinitionConfig) map[string]*model.Tag {
	tags := make(map[string]*model.Tag)
	for _, tag := range imageDef.Tags {
		tags[tag.Name] = tag
	}
	return tags
}

func processVariants(imageDef *model.ImageDefinitionConfig, imageRoot, parentIcon string) (map[string]*model.ImageVariant, error) {
	indexedVariants := make(map[string]*model.ImageVariant)
	for _, v := range imageDef.Variants {
		variantRoot := filepath.Join(imageRoot, v.Name)

		variantFsRoot, err := getRoofsPath(variantRoot)
		if err != nil {
			return nil, errors.Join(errors.New("failed to discover rootfs dir for variant "+v.Name), err)
		}

		testConfigFilePath, err := getTestConfigFilePath(variantRoot)
		if err != nil {
			return nil, errors.Join(errors.New("failed to discover test config file for variant "+v.Name), err)
		}

		dockerfilePath, err := file_resolver.ResolveFirstExistingFile(variantRoot, dockerfileConfigFileNames...)
		if err != nil {
			return nil, errors.Join(errors.New("failed to discover Dockerfile for variant "+v.Name), err)
		}

		readmePath, err := getReadmePath(variantRoot)
		if err != nil {
			return nil, errors.Join(errors.New("failed to discover README for variant "+v.Name), err)
		}

		icon := parentIcon
		if v.Report.Icon != nil {
			icon = *v.Report.Icon
		}

		variant := &model.ImageVariant{
			Name:                v.Name,
			BuildEntryPointPath: dockerfilePath,
			ReadmePath:          readmePath,
			TestConfigFilePath:  testConfigFilePath,
			RootDir:             variantRoot,
			TagSuffix:           v.TagSuffix,
			Versions:            v.Versions,
			BuildArgs:           v.BuildArgs,
			RootFSDir:           variantFsRoot,
			Platforms:           v.Platforms,
			Report: model.ReportModel{
				Icon: icon,
			},
			Labels: v.Labels,
		}

		indexedVariants[v.Name] = variant
	}
	return indexedVariants, nil
}
