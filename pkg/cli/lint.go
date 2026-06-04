package cli

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/ContainerHive/ContainerHive/pkg/lint"
	"github.com/urfave/cli/v3"
)

func lintCmd() *cli.Command {
	return &cli.Command{
		Name:  "lint",
		Usage: "Lint Dockerfiles with hadolint (skips templated Dockerfiles)",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "failure-threshold",
				Usage: "Override lint.failure_threshold from hive.yml (error, warning, info, style, ignore)",
			},
			&cli.StringSliceFlag{
				Name:  "format",
				Usage: "Output format (terminal, github-actions, codeclimate=<path>). Can be repeated. Default: terminal",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			project, err := discoverProject(ctx, cmd)
			if err != nil {
				return err
			}

			cfg := lint.ResolveConfig(project.Config.Lint, cmd.String("failure-threshold"))

			linter, err := lint.NewLinter(cfg)
			if err != nil {
				return err
			}
			defer linter.Close()

			projectRoot := project.RootDir
			if resolved, evalErr := filepath.EvalSymlinks(projectRoot); evalErr == nil {
				projectRoot = resolved
			}

			opts, err := lint.ParseFormats(cmd.StringSlice("format"))
			if err != nil {
				return fmt.Errorf("invalid --format: %w", err)
			}

			result, err := linter.Lint(project, projectRoot)
			if err != nil {
				return err
			}

			if len(result.Findings) > 0 {
				for _, opt := range opts {
					var f lint.Format
					switch opt.Name {
					case "terminal":
						f = &lint.TerminalFormat{Color: lint.StdoutSupportsColor()}
					case "github-actions":
						f = &lint.GitHubActionsFormat{}
					case "codeclimate":
						f = lint.NewCodeClimateFormat(opt.Path)
					}
					if err := f.Render(os.Stdout, result.Findings); err != nil {
						return fmt.Errorf("failed to render %s output: %w", f.Name(), err)
					}
				}
			}

			if result.LintedCount == 0 {
				slog.Info("No plain Dockerfiles to lint")
				return nil
			}

			if result.FailedCount > 0 {
				return fmt.Errorf("hadolint reported findings in %d Dockerfile(s)", result.FailedCount)
			}

			slog.Info("Lint passed", "files", result.LintedCount)
			return nil
		},
	}
}
