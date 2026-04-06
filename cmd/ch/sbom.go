package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/timo-reymann/ContainerHive/pkg/platform"
	"github.com/timo-reymann/ContainerHive/pkg/sbom"
	"github.com/timo-reymann/ContainerHive/pkg/utils"
	"github.com/urfave/cli/v3"
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

			sbomGen, err := sbom.NewGenerator()
			if err != nil {
				return fmt.Errorf("failed to initialize SBOM generator: %w", err)
			}

			var generated, skipped int
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
							skipped++
							continue
						}

						slog.Info("Generating SBOM", "image", img.Name, "tag", tagName, "platform", platformStr)
						sbomData, err := sbomGen.Generate(ctx, tarFile, "cyclonedx-json")
						if err != nil {
							return fmt.Errorf("SBOM generation failed for %s:%s [%s]: %w", img.Name, tagName, platformStr, err)
						}

						sbomPath := filepath.Join(platDir, "cyclonedx.json")
						if err := os.WriteFile(sbomPath, sbomData, 0644); err != nil {
							return fmt.Errorf("failed to write SBOM for %s:%s [%s]: %w", img.Name, tagName, platformStr, err)
						}
						slog.Info("SBOM written", "path", sbomPath, "bytes", len(sbomData))
						generated++
					}
				}
			}

			if generated == 0 && skipped > 0 {
				return fmt.Errorf("no SBOMs generated — %d image(s) have no image.tar, please build images first", skipped)
			}

			slog.Info("Generated SBOMs", "count", generated)
			return nil
		},
	}
}
