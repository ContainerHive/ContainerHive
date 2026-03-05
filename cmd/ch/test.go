package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/timo-reymann/ContainerHive/pkg/build"
	"github.com/timo-reymann/ContainerHive/pkg/cst"
	"github.com/timo-reymann/ContainerHive/pkg/discovery"
	"github.com/timo-reymann/ContainerHive/pkg/platform"
	"github.com/urfave/cli/v3"
)

func testCmd() *cli.Command {
	return &cli.Command{
		Name:      "test",
		Usage:     "Run container structure tests on built images",
		ArgsUsage: "[image:tag ...]",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			projectRoot := cmd.String("project")
			filters := parseFilters(cmd.Args().Slice())
			distPath := filepath.Join(projectRoot, "dist")
			if _, err := os.Stat(distPath); err != nil {
				return fmt.Errorf("dist/ not found — run 'ch generate' first: %w", err)
			}

			project, err := discovery.DiscoverProject(ctx, projectRoot)
			if err != nil {
				return fmt.Errorf("discovery failed: %w", err)
			}

			var tested, failed int
			for _, img := range project.ImagesByIdentifier {
				for tagName := range img.Tags {
					if matchesFilter(filters, img.Name, tagName) {
						t, f, err := runTestsForTag(distPath, img.Name, tagName,
							platform.Resolve(project.Config.Platforms, img.Platforms, nil))
						if err != nil {
							return err
						}
						tested += t
						failed += f
					}

					for _, variantDef := range img.Variants {
						variantTag := tagName + variantDef.TagSuffix
						if !matchesFilter(filters, img.Name, variantTag) {
							continue
						}
						t, f, err := runTestsForTag(distPath, img.Name, variantTag,
							platform.Resolve(project.Config.Platforms, img.Platforms, variantDef.Platforms))
						if err != nil {
							return err
						}
						tested += t
						failed += f
					}
				}
			}

			log.Printf("Tested %d image(s), %d failed", tested, failed)
			if failed > 0 {
				return fmt.Errorf("%d test(s) failed", failed)
			}
			return nil
		},
	}
}

// runTestsForTag runs container structure tests for a single tag directory
// across all given platforms. Returns the number of images tested and failed.
func runTestsForTag(distPath, imageName, tagName string, platforms []string) (tested, failed int, _ error) {
	tagDir := filepath.Join(distPath, imageName, tagName)
	testDefs := cst.CollectTestDefinitions(tagDir)
	if len(testDefs) == 0 {
		log.Printf("No test definitions for %s:%s, skipping", imageName, tagName)
		return 0, 0, nil
	}

	for _, platformStr := range platforms {
		platDir := filepath.Join(tagDir, platform.Sanitize(platformStr))
		tarFile := filepath.Join(platDir, "image.tar")
		if _, err := os.Stat(tarFile); err != nil {
			log.Printf("Skipping %s:%s [%s] — no image.tar found", imageName, tagName, platformStr)
			continue
		}

		cstRunner, err := cst.NewRunner(platformStr)
		if err != nil {
			return tested, failed, fmt.Errorf("failed to initialize CST runner for %s: %w", platformStr, err)
		}

		reportFile := cst.ReportFileName(platDir, imageName+":"+tagName)
		log.Printf("Testing %s:%s [%s] (%d test file(s))...", imageName, tagName, platformStr, len(testDefs))
		tested++
		if err := cstRunner.RunTests(tarFile, testDefs, reportFile); err != nil {
			log.Printf("FAIL %s:%s [%s]: %v", imageName, tagName, platformStr, err)
			failed++
			cstRunner.Close()
			continue
		}
		log.Printf("PASS %s:%s [%s] -> %s", imageName, tagName, platformStr, reportFile)
		cstRunner.Close()
	}
	return tested, failed, nil
}

// matchesFilter checks if an image:tag matches the given filters.
// Empty filters match everything.
func matchesFilter(filters []build.Filter, imageName, tagName string) bool {
	if len(filters) == 0 {
		return true
	}
	for _, f := range filters {
		if f.ImageName != "" && f.ImageName != imageName {
			continue
		}
		if f.TagName != "" && f.TagName != tagName {
			continue
		}
		return true
	}
	return false
}
