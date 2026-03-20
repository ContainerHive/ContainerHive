package discovery

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sync"

	"github.com/timo-reymann/ContainerHive/pkg/model"
	"golang.org/x/sync/errgroup"
)

func verifyProjectRoot(root string) error {
	stat, err := os.Stat(root)
	if err != nil {
		if os.IsNotExist(err) {
			return errors.New("project root does not exist")
		}
		return errors.Join(errors.New("failed to determine project root"), err)
	}

	if !stat.IsDir() {
		return errors.New("project root is not a directory")
	}

	return nil
}

func discoverImages(ctx context.Context, rootPath string) (map[string]*model.Image, error) {
	eg, ctx := errgroup.WithContext(ctx)
	images := map[string]*model.Image{}
	foundImageConfigs := make(chan string)
	var mutex sync.Mutex

	eg.Go(func() error {
		err := filepath.WalkDir(rootPath, func(path string, d fs.DirEntry, err error) error {
			if err := ctx.Err(); err != nil {
				return filepath.SkipDir
			}
			if err != nil {
				return err
			}

			if d.IsDir() {
				if d.Name() == "rootfs" {
					return filepath.SkipDir
				}
				return nil
			}

			name := d.Name()
			if slices.Contains(imageConfigFileNames, name) {
				foundImageConfigs <- path
				return filepath.SkipDir
			}

			return nil
		})
		close(foundImageConfigs)
		return err
	})
	eg.Go(func() error {
		for image := range foundImageConfigs {
			eg.Go(func() error {
				config, err := processImageConfig(rootPath, image)
				if err != nil {
					return err
				}

				mutex.Lock()
				images[config.Identifier] = config
				mutex.Unlock()
				return nil
			})

		}
		return nil
	})

	return images, eg.Wait()
}

func DiscoverProject(ctx context.Context, root string) (*model.ContainerHiveProject, error) {
	if err := verifyProjectRoot(root); err != nil {
		return nil, errors.Join(errors.New("failed to verify project root"), err)
	}
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, errors.Join(errors.New("failed to determine absolute project root"), err)
	}

	configPath, err := getContainerHiveConfigFile(root)
	if err != nil {
		return nil, errors.Join(errors.New("failed to discover ContainerHive config file"), err)
	}
	absoluteConfigPath, err := filepath.Abs(configPath)
	if err != nil {
		return nil, errors.Join(errors.New("failed to determine absolute config path"), err)
	}

	hiveConfig, err := parseHiveConfigFile(absoluteConfigPath)
	if err != nil {
		return nil, errors.Join(errors.New("failed to parse ContainerHive config"), err)
	}

	imagesPath, err := filepath.EvalSymlinks(filepath.Join(absoluteRoot, "images"))
	if err != nil {
		return nil, errors.Join(errors.New("failed to resolve images path"), err)
	}

	images, err := discoverImages(ctx, imagesPath)
	if err != nil {
		return nil, errors.Join(errors.New("failed to discover images"), err)
	}
	imagesByName := make(map[string][]*model.Image)
	for _, image := range images {
		imagesByName[image.Name] = append(imagesByName[image.Name], image)
	}

	project := &model.ContainerHiveProject{
		RootDir:            absoluteRoot,
		ConfigFilePath:     absoluteConfigPath,
		Config:             *hiveConfig,
		ImagesByIdentifier: images,
		ImagesByName:       imagesByName,
	}

	return project, nil
}
