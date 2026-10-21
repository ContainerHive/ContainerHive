package discovery

import (
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/ContainerHive/ContainerHive/pkg/model"
	"gopkg.in/yaml.v3"
)

func ensureSecretsInitialized(secrets model.Secrets) model.Secrets {
	if secrets == nil {
		return make(model.Secrets)
	}
	return secrets
}

// ensureNixConfig dereferences an optional NixConfig for the value-typed field
// on model.Image. Zero values mean the build applies its own defaults.
func ensureNixConfig(cfg *model.NixConfig) model.NixConfig {
	if cfg == nil {
		return model.NixConfig{}
	}
	return *cfg
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

	entrypoint, err := resolveBuildEntrypoint(parsedImageDef.Build, imageRoot)
	if err != nil {
		return nil, errors.Join(errors.New("failed to discover build entrypoint"), err)
	}

	if entrypoint.Backend == model.BuildBackendNix {
		_, lockErr := os.Stat(filepath.Join(imageRoot, "flake.lock"))
		if os.IsNotExist(lockErr) {
			slog.Warn("No flake.lock: builds resolve inputs at build time and are not reproducible — run nix flake lock and commit flake.lock", "image", relativeRoot)
		}
	}

	readmePath, err := getReadmePath(imageRoot)
	if err != nil {
		return nil, errors.Join(errors.New("failed to discover README"), err)
	}

	return &model.Image{
		RootDir:             filepath.Join(projectRoot, relativeRoot),
		BuildEntryPointPath: entrypoint.Path,
		BuildBackend:        entrypoint.Backend,
		Nix:                 ensureNixConfig(parsedImageDef.Nix),
		RunConfig:           parsedImageDef.Config,
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
		TagRanges:           parsedImageDef.TagRanges,
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

		variantEntrypoint, err := resolveBuildEntrypoint(v.Build, variantRoot)
		if err != nil {
			return nil, errors.Join(errors.New("failed to discover build entrypoint for variant "+v.Name), err)
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
			BuildEntryPointPath: variantEntrypoint.Path,
			BuildBackend:        variantEntrypoint.Backend,
			Nix:                 v.Nix,
			RunConfig:           v.Config,
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
