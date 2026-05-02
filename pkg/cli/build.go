package cli

import (
	"context"

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
			filters := utils.ParseFilters(cmd.Args().Slice())

			if cmd.Bool("generate") {
				if err := generateProject(ctx, cmd); err != nil {
					return err
				}
			}

			project, err := discoverProject(ctx, cmd)
			if err != nil {
				return err
			}

			distPath := getDistPath(cmd)

			return buildProject(ctx, project, distPath, filters, buildID,
				cmd.StringSlice("platform"), cmd.Bool("registry"))
		},
	}
}
