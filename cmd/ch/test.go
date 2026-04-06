package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/timo-reymann/ContainerHive/pkg/test"
	"github.com/timo-reymann/ContainerHive/pkg/utils"
	"github.com/urfave/cli/v3"
)

func testCmd() *cli.Command {
	return &cli.Command{
		Name:      "test",
		Usage:     "Run container structure tests on built images",
		ArgsUsage: "[image:tag ...]",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			filters := utils.ParseFilters(cmd.Args().Slice())
			distPath := getDistPath(cmd)
			if _, err := os.Stat(distPath); err != nil {
				return fmt.Errorf("dist/ not found — run 'ch generate' first: %w", err)
			}

			project, err := discoverProject(ctx, cmd)
			if err != nil {
				return err
			}

			opts := &test.Opts{
				DistPath: distPath,
				Project:  project,
				Filters:  filters,
				BuildID:  cmd.String("build-id"),
			}

			if os.Getenv("CI") != "" {
				reg, err := setupRegistry(ctx, distPath, project.Config.Registry)
				if err != nil {
					return err
				}
				defer reg.Stop(ctx)
				opts.Registry = reg
			}

			tested, failed, err := test.RunProjectTests(ctx, opts)
			if err != nil {
				return err
			}

			slog.Info("Test results", "tested", tested, "failed", failed)
			if failed > 0 {
				return fmt.Errorf("%d test(s) failed", failed)
			}
			return nil
		},
	}
}
