package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"github.com/timo-reymann/ContainerHive/pkg/model"
	"github.com/timo-reymann/ContainerHive/pkg/platform"
	"github.com/timo-reymann/ContainerHive/pkg/sbom"
	"github.com/timo-reymann/ContainerHive/pkg/utils"
	"github.com/urfave/cli/v3"
	"golang.org/x/sync/errgroup"
)

func sbomCmd() *cli.Command {
	return &cli.Command{
		Name:      "sbom",
		Usage:     "Generate SBOMs for built images",
		ArgsUsage: "[image:tag ...]",
		Flags: []cli.Flag{
			&cli.StringSliceFlag{
				Name:  "platform",
				Usage: "Target platform(s) to generate SBOMs for (e.g. linux/amd64), overrides hive.yml",
			},
			&cli.IntFlag{
				Name:  "workers",
				Usage: "Number of concurrent workers for SBOM generation",
				Value: 4,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			filters := utils.ParseFilters(cmd.Args().Slice())
			distPath := getDistPath(cmd)
			if _, err := os.Stat(distPath); err != nil {
				return fmt.Errorf("dist/ not found — run 'ch generate' first: %w", err)
			}

			project, err := discoverProject(ctx, cmd)
			if err != nil {
				return err
			}

			if cliPlatforms := cmd.StringSlice("platform"); len(cliPlatforms) > 0 {
				project.Config.Platforms = cliPlatforms
			}

			workers := cmd.Int("workers")
			if workers < 1 {
				workers = 1
			}
			if workers > 4 {
				workers = 4
			}

			type workItem struct {
				img         *model.Image
				tagName     string
				platformStr string
				tarFile     string
				platDir     string
			}

			var workItems []workItem
			for _, img := range project.ImagesByIdentifier {
				for tagName := range img.Tags {
					if !utils.MatchesFilter(filters, img.Name, tagName) {
						continue
					}

					platforms := platform.Resolve(project.Config.Platforms, img.Platforms, nil)
					for _, platformStr := range platforms {
						platDir := filepath.Join(distPath, img.Name, tagName, platform.Sanitize(platformStr))
						tarFile := filepath.Join(platDir, "image.tar")
						if _, err := os.Stat(tarFile); err != nil {
							slog.Warn("Skipping, no image.tar found", "image", img.Name, "tag", tagName, "platform", platformStr)
							continue
						}

						workItems = append(workItems, workItem{
							img:         img,
							tagName:     tagName,
							platformStr: platformStr,
							tarFile:     tarFile,
							platDir:     platDir,
						})
					}
				}
			}

			if len(workItems) == 0 {
				return fmt.Errorf("no images found to generate SBOMs for")
			}

			sbomGen, err := sbom.NewGenerator()
			if err != nil {
				return fmt.Errorf("failed to initialize SBOM generator: %w", err)
			}

			var mu sync.Mutex
			var generated int
			var firstErr error

			eg, ctx := errgroup.WithContext(ctx)
			for i := 0; i < workers; i++ {
				eg.Go(func() error {
					for {
						item, ok := func() (workItem, bool) {
							mu.Lock()
							defer mu.Unlock()
							if len(workItems) == 0 {
								return workItem{}, false
							}
							item := workItems[0]
							workItems = workItems[1:]
							return item, true
						}()
						if !ok {
							return nil
						}

						slog.Info("Generating SBOM", "image", item.img.Name, "tag", item.tagName, "platform", item.platformStr)
						sbomData, err := sbomGen.Generate(ctx, item.tarFile, "cyclonedx-json")
						if err != nil {
							mu.Lock()
							if firstErr == nil {
								firstErr = fmt.Errorf("SBOM generation failed for %s:%s [%s]: %w", item.img.Name, item.tagName, item.platformStr, err)
							}
							mu.Unlock()
							return nil
						}

						sbomPath := filepath.Join(item.platDir, "cyclonedx.json")
						if err := os.WriteFile(sbomPath, sbomData, 0644); err != nil {
							mu.Lock()
							if firstErr == nil {
								firstErr = fmt.Errorf("failed to write SBOM for %s:%s [%s]: %w", item.img.Name, item.tagName, item.platformStr, err)
							}
							mu.Unlock()
							return nil
						}

						slog.Info("SBOM written", "path", sbomPath, "bytes", len(sbomData))
						mu.Lock()
						generated++
						mu.Unlock()
					}
				})
			}

			if err := eg.Wait(); err != nil {
				return err
			}

			if firstErr != nil {
				return firstErr
			}

			slog.Info("Generated SBOMs", "count", generated, "workers", workers)
			return nil
		},
	}
}
