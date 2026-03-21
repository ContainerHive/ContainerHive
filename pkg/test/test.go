package test

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	cst "github.com/timo-reymann/ContainerHive/internal/container_structure_test"
	"github.com/timo-reymann/ContainerHive/pkg/build"
	"github.com/timo-reymann/ContainerHive/pkg/model"
	"github.com/timo-reymann/ContainerHive/pkg/platform"
	"github.com/timo-reymann/ContainerHive/pkg/utils"
)

// Registry provides image references for pulling from a remote registry.
type Registry interface {
	ImageRef(imageName, tag, platformStr, buildID string) string
}

// Opts holds configuration for running project tests.
type Opts struct {
	DistPath string
	Project  *model.ContainerHiveProject
	Filters  []build.Filter
	Registry Registry // if set, use registry refs when no local tar exists
	BuildID  string
}

// RunProjectTests executes container structure tests for all images in a project
// that match the given filters. Returns the number of tests run and failed.
func RunProjectTests(ctx context.Context, opts *Opts) (tested, failed int, err error) {
	for _, img := range opts.Project.ImagesByIdentifier {
		for tagName := range img.Tags {
			if utils.MatchesFilter(opts.Filters, img.Name, tagName) {
				t, f, err := runTestsForTag(opts, img.Name, tagName,
					platform.Resolve(opts.Project.Config.Platforms, img.Platforms, nil))
				if err != nil {
					return tested, failed, err
				}
				tested += t
				failed += f
			}

			for _, variantDef := range img.Variants {
				variantTag := tagName + variantDef.TagSuffix
				if !utils.MatchesFilter(opts.Filters, img.Name, variantTag) {
					continue
				}
				t, f, err := runTestsForTag(opts, img.Name, variantTag,
					platform.Resolve(opts.Project.Config.Platforms, img.Platforms, variantDef.Platforms))
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
func runTestsForTag(opts *Opts, imageName, tagName string, platforms []string) (tested, failed int, _ error) {
	tagDir := filepath.Join(opts.DistPath, imageName, tagName)
	testDefs := cst.CollectTestDefinitions(tagDir)
	if len(testDefs) == 0 {
		log.Printf("No test definitions for %s:%s, skipping", imageName, tagName)
		return 0, 0, nil
	}

	for _, platformStr := range platforms {
		platDir := filepath.Join(tagDir, platform.Sanitize(platformStr))
		imageSource := filepath.Join(platDir, "image.tar")

		if _, err := os.Stat(imageSource); err != nil {
			if opts.Registry == nil {
				log.Printf("Skipping %s:%s [%s] — no image.tar found", imageName, tagName, platformStr)
				continue
			}
			imageSource = opts.Registry.ImageRef(imageName, tagName, platformStr, opts.BuildID)
			log.Printf("No local tar for %s:%s [%s], using registry ref: %s", imageName, tagName, platformStr, imageSource)
		}

		if err := os.MkdirAll(platDir, 0755); err != nil {
			return tested, failed, fmt.Errorf("failed to create platform dir for %s:%s [%s]: %w", imageName, tagName, platformStr, err)
		}

		cstRunner, err := cst.NewRunner(platformStr)
		if err != nil {
			return tested, failed, fmt.Errorf("failed to initialize CST runner for %s: %w", platformStr, err)
		}

		reportFile := cst.ReportFileName(platDir, imageName+":"+tagName)
		log.Printf("Testing %s:%s [%s] (%d test file(s))...", imageName, tagName, platformStr, len(testDefs))
		tested++
		if err := cstRunner.RunTestsForImage(imageSource, testDefs, reportFile); err != nil {
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
