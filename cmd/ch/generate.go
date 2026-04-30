package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/ContainerHive/ContainerHive/pkg/rendering"
	"github.com/urfave/cli/v3"
)

func generateCmd() *cli.Command {
	return &cli.Command{
		Name:  "generate",
		Usage: "Discover project and render to dist/",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			project, err := discoverProject(ctx, cmd)
			if err != nil {
				return err
			}

			distPath := getDistPath(cmd)
			if err := rendering.RenderProject(ctx, project, distPath); err != nil {
				return fmt.Errorf("rendering failed: %w", err)
			}

			slog.Info("Rendered images to dist/", "count", len(project.ImagesByIdentifier), "path", distPath)
			return nil
		},
	}
}
