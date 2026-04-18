package main

import (
	"context"
	"embed"
	"fmt"

	"github.com/timo-reymann/ContainerHive/pkg/report"
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
				panic(err)
			}
			fmt.Println("--- CLI ---")
			fmt.Print(string(content))
			fmt.Println("--- Web Report ---")
			fmt.Println(string(report.NoticeContent))
			return nil
		},
	}
}
