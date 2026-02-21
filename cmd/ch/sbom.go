package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/timo-reymann/ContainerHive/pkg/discovery"
	"github.com/timo-reymann/ContainerHive/pkg/sbom"
	"github.com/urfave/cli/v3"
)

func sbomCmd() *cli.Command {
	return &cli.Command{
		Name:      "sbom",
		Usage:     "Generate SBOMs for built images",
		ArgsUsage: "[image:tag ...]",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			projectRoot := cmd.String("project")
			filters := parseFilters(cmd.Args().Slice())
			distPath := filepath.Join(projectRoot, "dist")

			project, err := discovery.DiscoverProject(ctx, projectRoot)
			if err != nil {
				return fmt.Errorf("discovery failed: %w", err)
			}

			sbomGen, err := sbom.NewGenerator()
			if err != nil {
				return fmt.Errorf("failed to initialize SBOM generator: %w", err)
			}

			var generated int
			for _, img := range project.ImagesByIdentifier {
				for tagName := range img.Tags {
					if !matchesFilter(filters, img.Name, tagName) {
						continue
					}

					tagDir := filepath.Join(distPath, img.Name, tagName)
					tarFile := filepath.Join(tagDir, "image.tar")
					if _, err := os.Stat(tarFile); err != nil {
						log.Printf("Skipping %s:%s — no image.tar found", img.Name, tagName)
						continue
					}

					log.Printf("Generating SBOM for %s:%s ...", img.Name, tagName)
					sbomData, err := sbomGen.Generate(ctx, tarFile, "spdx-json")
					if err != nil {
						return fmt.Errorf("SBOM generation failed for %s:%s: %w", img.Name, tagName, err)
					}

					sbomPath := filepath.Join(tagDir, "sbom.spdx.json")
					if err := os.WriteFile(sbomPath, sbomData, 0644); err != nil {
						return fmt.Errorf("failed to write SBOM for %s:%s: %w", img.Name, tagName, err)
					}
					log.Printf("SBOM written: %s (%d bytes)", sbomPath, len(sbomData))
					generated++
				}
			}

			log.Printf("Generated %d SBOM(s)", generated)
			return nil
		},
	}
}
