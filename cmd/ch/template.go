package main

import (
	"context"
	"fmt"
	"os"
	"strings"

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
			&cli.BoolFlag{
				Name:  "artifacts",
				Usage: "Upload/download build artifacts between jobs",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			project, err := discoverProject(ctx, cmd)
			if err != nil {
				return err
			}

			ciCtx, err := ci.BuildCIContext(project, cmd.Bool("artifacts"))
			if err != nil {
				return fmt.Errorf("failed to build CI context: %w", err)
			}

			ciCtx.Command = buildCICommand(cmd)

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

			ciCtx, err := ci.BuildCIContext(project, false)
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

func buildCICommand(cmd *cli.Command) string {
	parts := []string{"ch template ci --provider", cmd.String("provider")}
	if cmd.Bool("artifacts") {
		parts = append(parts, "--artifacts")
	}
	if dir := cmd.String("template-dir"); dir != "" {
		parts = append(parts, "--template-dir", dir)
	}
	if output := cmd.String("output"); output != "" {
		parts = append(parts, "--output", output)
	}
	return strings.Join(parts, " ")
}

func writeOutput(outputPath string, data []byte) error {
	if outputPath == "" {
		_, err := os.Stdout.Write(data)
		return err
	}
	return os.WriteFile(outputPath, data, 0644)
}
