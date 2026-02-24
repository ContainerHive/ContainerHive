package main

import (
	"context"
	"fmt"
	"log"
	"path/filepath"

	"github.com/timo-reymann/ContainerHive/pkg/discovery"
	"github.com/timo-reymann/ContainerHive/pkg/rendering"
	"github.com/urfave/cli/v3"
)

func generateCmd() *cli.Command {
	return &cli.Command{
		Name:  "generate",
		Usage: "Discover project and render to dist/",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			projectRoot := cmd.String("project")

			project, err := discovery.DiscoverProject(ctx, projectRoot)
			if err != nil {
				return fmt.Errorf("discovery failed: %w", err)
			}

			distPath := filepath.Join(projectRoot, "dist")
			if err := rendering.RenderProject(ctx, project, distPath); err != nil {
				return fmt.Errorf("rendering failed: %w", err)
			}

			log.Printf("Rendered %d image(s) to %s", len(project.ImagesByIdentifier), distPath)
			return nil
		},
	}
}
