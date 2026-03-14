package main

import (
	"context"
	"embed"
	"fmt"

	"github.com/urfave/cli/v3"
)

//go:embed NOTICE
var noticeFS embed.FS

func licenseCmd() *cli.Command {
	return &cli.Command{
		Name:  "license",
		Usage: "Show third-party license notices",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			content, err := noticeFS.ReadFile("NOTICE")
			if err != nil {
				return fmt.Errorf("license notices not available: %w", err)
			}
			fmt.Print(string(content))
			return nil
		},
	}
}
