package cli

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/urfave/cli/v3"
)

func verifyCmd() *cli.Command {
	return &cli.Command{
		Name:  "verify",
		Usage: "Verify project structure",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			project, err := discoverProject(ctx, cmd)
			if err != nil {
				return fmt.Errorf("project verification failed: %w", err)
			}

			slog.Info("Project OK", "images", len(project.ImagesByIdentifier))
			return nil
		},
	}
}
