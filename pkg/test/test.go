package test

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/timo-reymann/ContainerHive/pkg/build"
	"github.com/timo-reymann/ContainerHive/pkg/cst"
	"github.com/timo-reymann/ContainerHive/pkg/model"
	"github.com/timo-reymann/ContainerHive/pkg/platform"
	"github.com/timo-reymann/ContainerHive/pkg/utils"
)

// RunProjectTests executes container structure tests for all images in a project
// that match the given filters. Returns the number of tests run and failed.
func RunProjectTests(ctx context.Context, distPath string, project *model.ContainerHiveProject, filters []build.Filter) (tested, failed int, err error) {
	for _, img := range project.ImagesByIdentifier {
		for tagName := range img.Tags {
			if utils.MatchesFilter(filters, img.Name, tagName) {
				t, f, err := runTestsForTag(distPath, img.Name, tagName,
					platform.Resolve(project.Config.Platforms, img.Platforms, nil))
				if err != nil {
					return tested, failed, err
				}
				tested += t
				failed += f
			}

			for _, variantDef := range img.Variants {
				variantTag := tagName + variantDef.TagSuffix
				if !utils.MatchesFilter(filters, img.Name, variantTag) {
					continue
				}
				t, f, err := runTestsForTag(distPath, img.Name, variantTag,
					platform.Resolve(project.Config.Platforms, img.Platforms, variantDef.Platforms))
				if err != nil {
					return tested, failed, err
				}
				tested += t
				failed += f
			}
		}
	}
	return tested, failed, nil
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
