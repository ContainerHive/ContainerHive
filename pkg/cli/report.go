package cli

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/ContainerHive/ContainerHive/pkg/report"
	"github.com/urfave/cli/v3"
)

func reportCmd() *cli.Command {
	return &cli.Command{
		Name:      "report",
		Usage:     "Generate an HTML report of container images",
		ArgsUsage: "",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "output",
				Usage: "Output file path (default: dist/report.html)",
				Value: "",
			},
			&cli.BoolFlag{
				Name:  "json",
				Usage: "Output JSON instead of HTML",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			projectRoot := cmd.String("project")
			outputPath := cmd.String("output")
			jsonOnly := cmd.Bool("json")

			if outputPath == "" {
				outputPath = filepath.Join(projectRoot, "dist", "report.html")
				if jsonOnly {
					outputPath = filepath.Join(projectRoot, "dist", "report.json")
				}
			}

			project, err := discoverProject(ctx, cmd)
			if err != nil {
				return err
			}

			gen := report.NewGenerator()
			reportData, err := gen.Generate(project)
			if err != nil {
				return fmt.Errorf("failed to generate report: %w", err)
			}

			slog.Info("Report generated", "images", len(reportData.Images))

			if jsonOnly {
				data, err := gen.GenerateJSON(reportData)
				if err != nil {
					return fmt.Errorf("failed to serialize JSON: %w", err)
				}
				if err := os.WriteFile(outputPath, data, 0644); err != nil {
					return fmt.Errorf("failed to write report: %w", err)
				}
			} else {
				data, err := gen.GenerateHTMLFromAssets(reportData)
				if err != nil {
					return fmt.Errorf("failed to generate HTML: %w", err)
				}
				if err := os.WriteFile(outputPath, data, 0644); err != nil {
					return fmt.Errorf("failed to write report: %w", err)
				}
			}

			slog.Info("Report written", "path", outputPath)
			return nil
		},
	}
}
