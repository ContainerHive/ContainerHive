package main

import (
	"context"
	"fmt"
	"os"

	"github.com/timo-reymann/ContainerHive/pkg/ci"
	"github.com/timo-reymann/ContainerHive/pkg/templating"
	"github.com/urfave/cli/v3"
)

func templateCmd() *cli.Command {
	return &cli.Command{
		Name:  "template",
		Usage: "Generate files from templates",
		Commands: []*cli.Command{
			templateCICmd(),
			templateCustomCmd(),
		},
	}
}

func templateCICmd() *cli.Command {
	return &cli.Command{
		Name:  "ci",
		Usage: "Generate CI pipeline configuration",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "provider",
				Usage:    "CI provider (gitlab, github)",
				Required: true,
			},
			&cli.StringFlag{
				Name:  "output",
				Usage: "Output file (default: stdout)",
			},
			&cli.StringFlag{
				Name:  "template-dir",
				Usage: "Custom template directory (overrides built-in templates)",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			project, err := discoverProject(ctx, cmd)
			if err != nil {
				return err
			}

			ciCtx, err := ci.BuildCIContext(project)
			if err != nil {
				return fmt.Errorf("failed to build CI context: %w", err)
			}

			result, err := ci.Generate(cmd.String("provider"), ciCtx, cmd.String("template-dir"))
			if err != nil {
				return fmt.Errorf("failed to generate CI config: %w", err)
			}

			return writeOutput(cmd.String("output"), result)
		},
	}
}

func templateCustomCmd() *cli.Command {
	return &cli.Command{
		Name:  "custom",
		Usage: "Render a custom Go template with project context",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "template",
				Usage:    "Path to Go template file (.gotpl)",
				Required: true,
			},
			&cli.StringFlag{
				Name:  "output",
				Usage: "Output file (default: stdout)",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			project, err := discoverProject(ctx, cmd)
			if err != nil {
				return err
			}

			templatePath := cmd.String("template")
			content, err := os.ReadFile(templatePath)
			if err != nil {
				return fmt.Errorf("failed to read template: %w", err)
			}

			ciCtx, err := ci.BuildCIContext(project)
			if err != nil {
				return fmt.Errorf("failed to build CI context: %w", err)
			}

			result, err := templating.RenderString(templatePath, string(content), ciCtx)
			if err != nil {
				return fmt.Errorf("failed to render template: %w", err)
			}

			return writeOutput(cmd.String("output"), result)
		},
	}
}

func writeOutput(outputPath string, data []byte) error {
	if outputPath == "" {
		_, err := os.Stdout.Write(data)
		return err
	}
	return os.WriteFile(outputPath, data, 0644)
}
