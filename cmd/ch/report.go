package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/timo-reymann/ContainerHive/pkg/discovery"
	"github.com/timo-reymann/ContainerHive/pkg/report"
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
			&cli.StringFlag{
				Name:  "dist-path",
				Usage: "Path to dist/ directory",
				Value: "",
			},
			&cli.StringFlag{
				Name:  "source",
				Usage: "Data source: tar, registry, or auto (default: auto)",
				Value: "auto",
			},
			&cli.StringFlag{
				Name:  "registry",
				Usage: "Registry address (required for registry source)",
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
			distPath := cmd.String("dist-path")
			sourceStr := cmd.String("source")
			registryAddr := cmd.String("registry")
			jsonOnly := cmd.Bool("json")

			if distPath == "" {
				distPath = filepath.Join(projectRoot, "dist")
			}

			if outputPath == "" {
				if jsonOnly {
					outputPath = filepath.Join(distPath, "report.json")
				} else {
					outputPath = filepath.Join(distPath, "report.html")
				}
			}

			source := report.SourceType(sourceStr)
			if source == "" {
				source = report.SourceAuto
			}

			project, err := discovery.DiscoverProject(ctx, projectRoot)
			if err != nil {
				return fmt.Errorf("discovery failed: %w", err)
			}

			if registryAddr == "" && project.Config.Registry != nil {
				registryAddr = project.Config.Registry.Address
			}

			if registryAddr == "" && source == report.SourceRegistry {
				return fmt.Errorf("registry address required for registry source")
			}

			gen := report.NewGenerator(source, distPath)
			reportData, err := gen.Generate(ctx, project, registryAddr)
			if err != nil {
				return fmt.Errorf("failed to generate report: %w", err)
			}

			slog.Info("Report generated", "images", len(reportData.Images), "source", reportData.Source)

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
