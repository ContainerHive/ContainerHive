package discovery

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sync"

	"github.com/ContainerHive/ContainerHive/internal/tagrange"
	"github.com/ContainerHive/ContainerHive/pkg/model"
	"github.com/ContainerHive/ContainerHive/pkg/platform"
	"golang.org/x/sync/errgroup"
)

// discoverOptions holds the settings Option functions configure. The zero
// value resolves tag_ranges with a default (network-capable) Resolver,
// which only actually runs - and only then touches a cache or the network
// - when some image declares tag_ranges.
type discoverOptions struct {
	resolver *tagrange.Resolver
}

// Option configures optional DiscoverProject behavior.
type Option func(*discoverOptions)

// WithTagRangeResolver overrides the Resolver used to expand tag_ranges,
// primarily so tests can inject a fixture source.Registry instead of
// making real network calls.
func WithTagRangeResolver(r *tagrange.Resolver) Option {
	return func(o *discoverOptions) { o.resolver = r }
}

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

func DiscoverProject(ctx context.Context, root string, opts ...Option) (*model.ContainerHiveProject, error) {
	options := &discoverOptions{}
	for _, opt := range opts {
		opt(options)
	}

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

	if len(hiveConfig.Platforms) == 0 {
		hiveConfig.Platforms = platform.DefaultPlatforms
	}

	imagesPath, err := filepath.EvalSymlinks(filepath.Join(absoluteRoot, "images"))
	if err != nil {
		return nil, errors.Join(errors.New("failed to resolve images path"), err)
	}

	images, err := discoverImages(ctx, imagesPath)
	if err != nil {
		return nil, errors.Join(errors.New("failed to discover images"), err)
	}

	if err := resolveTagRanges(ctx, images, hiveConfig, absoluteRoot, options); err != nil {
		return nil, errors.Join(errors.New("failed to resolve tag_ranges"), err)
	}

	for _, img := range images {
		if len(img.Platforms) == 0 {
			img.Platforms = hiveConfig.Platforms
		}
		for _, variant := range img.Variants {
			if len(variant.Platforms) == 0 {
				variant.Platforms = img.Platforms
			}
		}
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
