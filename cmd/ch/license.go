package main

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/urfave/cli/v3"
)

//go:embed NOTICE
var noticeContent string

func licenseCmd() *cli.Command {
	return &cli.Command{
		Name:  "license",
		Usage: "Show third-party license notices",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			fmt.Print(noticeContent)
			return nil
		},
	}
}
