package main

import (
	"context"
	"fmt"
	"log"

	"github.com/timo-reymann/ContainerHive/pkg/discovery"
	"github.com/urfave/cli/v3"
)

func verifyCmd() *cli.Command {
	return &cli.Command{
		Name:  "verify",
		Usage: "Verify project structure",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			projectRoot := cmd.String("project")

			project, err := discovery.DiscoverProject(ctx, projectRoot)
			if err != nil {
				return fmt.Errorf("project verification failed: %w", err)
			}

			log.Printf("Project OK: %d image(s)", len(project.ImagesByIdentifier))
			return nil
		},
	}
}
