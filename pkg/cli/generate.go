package cli

import (
	"context"

	"github.com/urfave/cli/v3"
)

func generateCmd() *cli.Command {
	return &cli.Command{
		Name:  "generate",
		Usage: "Discover project and render to dist/",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return generateProject(ctx, cmd)
		},
	}
}
