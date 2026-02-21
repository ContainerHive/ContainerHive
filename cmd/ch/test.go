package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/timo-reymann/ContainerHive/pkg/build"
	"github.com/timo-reymann/ContainerHive/pkg/cst"
	"github.com/timo-reymann/ContainerHive/pkg/discovery"
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
			platform := "linux/" + runtime.GOARCH
			distPath := filepath.Join(projectRoot, "dist")

			project, err := discovery.DiscoverProject(ctx, projectRoot)
			if err != nil {
				return fmt.Errorf("discovery failed: %w", err)
			}

			cstRunner, err := cst.NewRunner(platform)
			if err != nil {
				return fmt.Errorf("failed to initialize CST runner: %w", err)
			}
			defer cstRunner.Close()

			var tested, failed int
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

					testDefs := cst.CollectTestDefinitions(tagDir)
					if len(testDefs) == 0 {
						log.Printf("No test definitions for %s:%s, skipping", img.Name, tagName)
						continue
					}

					reportFile := cst.ReportFileName(tagDir, img.Name+":"+tagName)
					log.Printf("Testing %s:%s (%d test file(s))...", img.Name, tagName, len(testDefs))
					tested++
					if err := cstRunner.RunTests(tarFile, testDefs, reportFile); err != nil {
						log.Printf("FAIL %s:%s: %v", img.Name, tagName, err)
						failed++
						continue
					}
					log.Printf("PASS %s:%s -> %s", img.Name, tagName, reportFile)
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
