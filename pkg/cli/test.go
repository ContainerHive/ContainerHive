package cli

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/ContainerHive/ContainerHive/pkg/test"
	"github.com/ContainerHive/ContainerHive/pkg/utils"
	"github.com/urfave/cli/v3"
)

func testCmd() *cli.Command {
	return &cli.Command{
		Name:      "test",
		Usage:     "Run container structure tests on built images",
		ArgsUsage: "[image:tag ...]",
		Flags: append(shardFlags(),
			&cli.BoolFlag{
				Name:  "build",
				Usage: "Run build before tests",
			},
			&cli.StringSliceFlag{
				Name:  "platform",
				Usage: "Target platform(s) to test (e.g. linux/amd64), selects within the configured platforms",
			},
		),
		Action: func(ctx context.Context, cmd *cli.Command) error {
			filters := utils.ParseFilters(cmd.Args().Slice())

			s, err := resolveShard(cmd)
			if err != nil {
				return err
			}

			if cmd.Bool("generate") {
				if err := generateProject(ctx, cmd); err != nil {
					return err
				}
			}

			distPath := getDistPath(cmd)

			project, err := discoverProject(ctx, cmd)
			if err != nil {
				return err
			}

			if err := validatePlatforms(project, cmd.StringSlice("platform")); err != nil {
				return err
			}

			if cmd.Bool("build") {
				if err := buildProject(ctx, project, distPath, filters, cmd.String("build-id"), cmd.StringSlice("platform"), false, s); err != nil {
					return err
				}
			}

			if _, err := os.Stat(distPath); err != nil {
				return fmt.Errorf("dist/ not found — run 'ch generate' first: %w", err)
			}

			opts := &test.Opts{
				DistPath:  distPath,
				Project:   project,
				Filters:   filters,
				BuildID:   cmd.String("build-id"),
				Shard:     s,
				Platforms: cmd.StringSlice("platform"),
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

			if tested == 0 {
				slog.Warn("No images were tested, check filters, shard and --platform")
			}
			slog.Info("Test results", "tested", tested, "failed", failed)
			if failed > 0 {
				return fmt.Errorf("%d test(s) failed", failed)
			}
			return nil
		},
	}
}
