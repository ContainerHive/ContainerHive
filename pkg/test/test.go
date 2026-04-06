package test

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
	cst "github.com/timo-reymann/ContainerHive/internal/container_structure_test"
	"github.com/timo-reymann/ContainerHive/pkg/build"
	"github.com/timo-reymann/ContainerHive/pkg/logging"
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
	// Format container-structure-test's logrus output identically to the tint
	// slog handler so it blends in with the rest of the application output.
	logrus.SetFormatter(&logging.TintFormatter{TimeFormat: time.DateTime})
	logrus.SetOutput(os.Stderr)

	for _, img := range opts.Project.ImagesByIdentifier {
		for tagName := range img.Tags {
			if utils.MatchesFilter(opts.Filters, img.Name, tagName) {
				t, f, err := runTestsForTag(ctx, opts, img.Name, tagName,
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
				t, f, err := runTestsForTag(ctx, opts, img.Name, variantTag,
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
func runTestsForTag(ctx context.Context, opts *Opts, imageName, tagName string, platforms []string) (tested, failed int, _ error) {
	tagDir := filepath.Join(opts.DistPath, imageName, tagName)
	testDefs := cst.CollectTestDefinitions(tagDir)
	if len(testDefs) == 0 {
		slog.Info("No test definitions, skipping", "image", imageName, "tag", tagName)
		return 0, 0, nil
	}

	for _, platformStr := range platforms {
		if err := ctx.Err(); err != nil {
			return tested, failed, err
		}

		platDir := filepath.Join(tagDir, platform.Sanitize(platformStr))
		imageSource := filepath.Join(platDir, "image.tar")

		if _, err := os.Stat(imageSource); err != nil {
			if opts.Registry == nil {
				slog.Info("Skipping, no image.tar found", "image", imageName, "tag", tagName, "platform", platformStr)
				continue
			}
			imageSource = opts.Registry.ImageRef(imageName, tagName, platformStr, opts.BuildID)
			slog.Info("No local tar, using registry ref", "image", imageName, "tag", tagName, "platform", platformStr, "ref", imageSource)
		}

		if err := os.MkdirAll(platDir, 0755); err != nil {
			return tested, failed, fmt.Errorf("failed to create platform dir for %s:%s [%s]: %w", imageName, tagName, platformStr, err)
		}

		cstRunner, err := cst.NewRunner(platformStr)
		if err != nil {
			return tested, failed, fmt.Errorf("failed to initialize CST runner for %s: %w", platformStr, err)
		}

		reportFile := cst.ReportFileName(platDir, imageName+":"+tagName)
		slog.Info("Testing image", "image", imageName, "tag", tagName, "platform", platformStr, "tests", len(testDefs))
		tested++
		if err := cstRunner.RunTestsForImage(ctx, imageSource, testDefs, reportFile); err != nil {
			slog.Error("FAIL", "image", imageName, "tag", tagName, "platform", platformStr, "error", err)
			failed++
			cstRunner.Close()
			if ctx.Err() != nil {
				return tested, failed, ctx.Err()
			}
			continue
		}
		slog.Info("PASS", "image", imageName, "tag", tagName, "platform", platformStr, "report", reportFile)
		cstRunner.Close()
	}
	return tested, failed, nil
}
