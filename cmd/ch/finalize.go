package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/ContainerHive/ContainerHive/pkg/utils"
	"github.com/urfave/cli/v3"
)

func finalizeCmd() *cli.Command {
	return &cli.Command{
		Name:      "finalize",
		Usage:     "Create multi-arch manifests and semantic version alias tags in the registry",
		ArgsUsage: "[image:tag ...]",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			buildID := cmd.String("build-id")
			filters := utils.ParseFilters(cmd.Args().Slice())

			// Discover project
			project, err := discoverProject(ctx, cmd)
			if err != nil {
				return err
			}

			distPath := getDistPath(cmd)
			if _, err := os.Stat(distPath); err != nil {
				return fmt.Errorf("dist/ not found — run 'ch generate' first: %w", err)
			}

			// Create registry (same as build)
			reg, err := setupRegistry(ctx, distPath, project.Config.Registry)
			if err != nil {
				return err
			}
			defer reg.Stop(ctx)

			// Step 1: Create multi-arch manifests from platform-specific images
			slog.Info("Creating multi-arch manifests...")
			if err := reg.CreateAllManifests(project, filters, buildID, distPath); err != nil {
				return fmt.Errorf("manifest creation failed: %w", err)
			}

			// Step 2: Retag manifests for semantic version aliases
			slog.Info("Retagging aliases...")
			if err := reg.RetagAllAliases(project, filters, buildID); err != nil {
				return fmt.Errorf("retagging failed: %w", err)
			}

			slog.Info("Finalize complete")
			return nil
		},
	}
}
