package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/timo-reymann/ContainerHive/pkg/build"
	"github.com/timo-reymann/ContainerHive/pkg/deps"
	"github.com/timo-reymann/ContainerHive/pkg/discovery"
	"github.com/timo-reymann/ContainerHive/pkg/registry"
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
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			projectRoot := cmd.String("project")
			buildID := cmd.String("build-id")
			useRegistry := cmd.Bool("registry") || os.Getenv("CI") != ""
			filters := parseFilters(cmd.Args().Slice())
			platform := "linux/" + runtime.GOARCH

			// Discover project
			project, err := discovery.DiscoverProject(ctx, projectRoot)
			if err != nil {
				return fmt.Errorf("discovery failed: %w", err)
			}

			// Expect dist/ to already exist (created by `ch generate`)
			distPath := filepath.Join(projectRoot, "dist")
			if _, err := os.Stat(distPath); err != nil {
				return fmt.Errorf("dist/ not found — run 'ch generate' first: %w", err)
			}

			// Resolve dependency build order
			buildOrder, err := deps.ResolveOrder(distPath, project)
			if err != nil {
				return fmt.Errorf("dependency resolution failed: %w", err)
			}
			log.Printf("Build order: %v", buildOrder.Order())

			// Connect to BuildKit (BUILDKIT_HOST env > hive.yml > default)
			buildkitAddr := "tcp://127.0.0.1:8502"
			if project.Config.BuildKit != nil && project.Config.BuildKit.Address != "" {
				buildkitAddr = project.Config.BuildKit.Address
			}
			if envAddr := os.Getenv("BUILDKIT_HOST"); envAddr != "" {
				buildkitAddr = envAddr
			}
			log.Printf("Connecting to BuildKit at %s...", buildkitAddr)
			bkClient, err := build.NewClient(ctx, buildkitAddr)
			if err != nil {
				return fmt.Errorf("failed to connect to BuildKit at %s: %w", buildkitAddr, err)
			}
			defer bkClient.Close()

			// Configure cache
			buildCache, err := buildCacheFromConfig(project.Config.Cache, "ch-build")
			if err != nil {
				return fmt.Errorf("cache configuration failed: %w", err)
			}

			// Build
			buildOpts := &build.ProjectBuildOpts{
				Project:     project,
				BuildOrder:  buildOrder,
				DistPath:    distPath,
				Platform:    platform,
				Cache:       buildCache,
				ProgressOut: os.Stdout,
				Filters:     filters,
				BuildID:     buildID,
			}

			if useRegistry || buildOrder.HasDependencies() {
				reg := registry.NewRegistry()
				if err := reg.Start(ctx); err != nil {
					return fmt.Errorf("failed to start registry: %w", err)
				}
				defer reg.Stop(ctx)
				log.Printf("Registry started: local=%v address=%s", reg.IsLocal(), reg.Address())

				buildOpts.Registry = reg
				if err := build.BuildProject(ctx, bkClient, buildOpts); err != nil {
					return fmt.Errorf("build failed: %w", err)
				}

				if err := reg.RetagAllAliases(project, filters, buildID); err != nil {
					return fmt.Errorf("retagging failed: %w", err)
				}
			} else {
				log.Println("No inter-image dependencies, building without registry")
				if err := build.BuildProject(ctx, bkClient, buildOpts); err != nil {
					return fmt.Errorf("build failed: %w", err)
				}
			}

			return nil
		},
	}
}
