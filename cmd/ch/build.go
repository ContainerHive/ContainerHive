package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/ContainerHive/ContainerHive/pkg/build"
	"github.com/ContainerHive/ContainerHive/pkg/cache"
	"github.com/ContainerHive/ContainerHive/pkg/deps"
	"github.com/ContainerHive/ContainerHive/pkg/progress"
	"github.com/ContainerHive/ContainerHive/pkg/utils"
	"github.com/urfave/cli/v3"
)

func buildCmd() *cli.Command {
	return &cli.Command{
		Name:      "build",
		Usage:     "Build container images",
		ArgsUsage: "[image:tag ...]",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "registry",
				Usage: "Use registry from config (auto-enabled in CI)",
			},
			&cli.StringSliceFlag{
				Name:  "platform",
				Usage: "Target platform(s) to build (e.g. linux/amd64), overrides hive.yml",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			buildID := cmd.String("build-id")
			useRegistry := cmd.Bool("registry") || os.Getenv("CI") != ""
			filters := utils.ParseFilters(cmd.Args().Slice())

			// Discover project
			project, err := discoverProject(ctx, cmd)
			if err != nil {
				return err
			}

			if cliPlatforms := cmd.StringSlice("platform"); len(cliPlatforms) > 0 {
				project.Config.Platforms = cliPlatforms
			}

			if len(project.Config.Platforms) == 0 {
				return fmt.Errorf("no platforms configured — set platforms in hive.yml or pass --platform")
			}

			// Expect dist/ to already exist (created by `ch generate`)
			distPath := getDistPath(cmd)
			if _, err := os.Stat(distPath); err != nil {
				return fmt.Errorf("dist/ not found — run 'ch generate' first: %w", err)
			}

			// Resolve dependency build order
			buildOrder, err := deps.ResolveOrder(distPath, project)
			if err != nil {
				return fmt.Errorf("dependency resolution failed: %w", err)
			}
			slog.Info("Build order resolved", "order", buildOrder.Order())

			// Connect to BuildKit (BUILDKIT_HOST env > hive.yml > default)
			buildkitAddr := ""
			if project.Config.BuildKit != nil && project.Config.BuildKit.Address != "" {
				buildkitAddr = project.Config.BuildKit.Address
			}
			bkClient, err := build.NewClient(ctx, buildkitAddr)
			if err != nil {
				return fmt.Errorf("failed to connect to BuildKit at %s: %w", buildkitAddr, err)
			}
			defer bkClient.Close()

			// Configure cache
			buildCache, err := cache.BuildCacheFromConfig(project.Config.Cache, "ch-build")
			if err != nil {
				return fmt.Errorf("cache configuration failed: %w", err)
			}

			// Select progress display mode: linear for CI, auto-detect otherwise.
			// Colors are enabled by default; suppressed only when NO_COLOR is set.
			progressMode := progress.AutoMode
			if os.Getenv("CI") != "" {
				progressMode = progress.LinearMode
			}

			// Build
			buildOpts := &build.ProjectBuildOpts{
				Project:     project,
				BuildOrder:  buildOrder,
				DistPath:    distPath,
				Cache:       buildCache,
				ProgressOut: os.Stdout,
				ProgressConfig: progress.Config{
					Mode:    progressMode,
					Writer:  os.Stdout,
					Colors:  progress.DefaultColors(),
					NoColor: os.Getenv("NO_COLOR") != "",
				},
				Filters: filters,
				BuildID: buildID,
			}

			if buildOrder.HasDependencies() {
				slog.Info("Inter-image dependencies detected, using OCI layout named contexts")
			}

			if useRegistry {
				reg, err := setupRegistry(ctx, distPath, project.Config.Registry)
				if err != nil {
					return err
				}
				defer reg.Stop(ctx)

				buildOpts.Registry = reg
			}

			if err := build.BuildProject(ctx, bkClient, buildOpts); err != nil {
				return fmt.Errorf("build failed: %w", err)
			}

			return nil
		},
	}
}
