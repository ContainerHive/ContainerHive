package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/timo-reymann/ContainerHive/pkg/discovery"
	"github.com/timo-reymann/ContainerHive/pkg/registry"
	"github.com/urfave/cli/v3"
)

func finalizeCmd() *cli.Command {
	return &cli.Command{
		Name:      "finalize",
		Usage:     "Create multi-arch manifests and semantic version alias tags in the registry",
		ArgsUsage: "[image:tag ...]",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			projectRoot := cmd.String("project")
			buildID := cmd.String("build-id")
			filters := parseFilters(cmd.Args().Slice())

			// Discover project
			project, err := discovery.DiscoverProject(ctx, projectRoot)
			if err != nil {
				return fmt.Errorf("discovery failed: %w", err)
			}

			distPath := filepath.Join(projectRoot, "dist")
			if _, err := os.Stat(distPath); err != nil {
				return fmt.Errorf("dist/ not found — run 'ch generate' first: %w", err)
			}

			// Create registry (same as build)
			reg, err := registry.NewRegistry(filepath.Join(distPath, ".registry"), project.Config.Registry)
			if err != nil {
				return fmt.Errorf("failed to create registry: %w", err)
			}
			if err := reg.Start(ctx); err != nil {
				return fmt.Errorf("failed to start registry: %w", err)
			}
			defer reg.Stop(ctx)
			log.Printf("Registry started: local=%v address=%s", reg.IsLocal(), reg.Address())

			// Step 1: Create multi-arch manifests from platform-specific images
			log.Println("Creating multi-arch manifests...")
			if err := reg.CreateAllManifests(project, filters, buildID, distPath); err != nil {
				return fmt.Errorf("manifest creation failed: %w", err)
			}

			// Step 2: Retag manifests for semantic version aliases
			log.Println("Retagging aliases...")
			if err := reg.RetagAllAliases(project, filters, buildID); err != nil {
				return fmt.Errorf("retagging failed: %w", err)
			}

			log.Println("Finalize complete")
			return nil
		},
	}
}
