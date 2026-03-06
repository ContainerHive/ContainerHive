package main

import (
	"context"
	"fmt"
	"log"
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

			tested, failed, err := test.RunProjectTests(ctx, distPath, project, filters)
			if err != nil {
				return err
			}

			log.Printf("Tested %d image(s), %d failed", tested, failed)
			if failed > 0 {
				return fmt.Errorf("%d test(s) failed", failed)
			}
			return nil
		},
	}
}
