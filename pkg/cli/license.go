package cli

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/ContainerHive/ContainerHive/pkg/report"
	gohadolint "github.com/timo-reymann/go-hadolint"
	"github.com/urfave/cli/v3"
)

//go:embed NOTICE
var noticeContent string

func licenseCmd() *cli.Command {
	return &cli.Command{
		Name:  "license",
		Usage: "Show third-party license notices",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			fmt.Println("--- CLI ---")
			fmt.Print(noticeContent)
			fmt.Println("--- Web Report ---")
			fmt.Println(string(report.NoticeContent))
			fmt.Println("--- Hadolint (embedded for ch lint) ---")
			fmt.Println(gohadolint.License())
			fmt.Println(gohadolint.ThirdPartyNotices())
			return nil
		},
	}
}
