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
			&cli.StringFlag{
				Name:  "codeclimate-report",
				Usage: "Write findings to a Code Climate / GitLab Code Quality JSON report at this path",
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

			// Resolve symlinks so report paths match what hadolint sees
			// (discovery resolves the root via filepath.EvalSymlinks).
			projectRoot := project.RootDir
			if resolved, evalErr := filepath.EvalSymlinks(projectRoot); evalErr == nil {
				projectRoot = resolved
			}

			result, err := linter.Lint(project, projectRoot)
			if err != nil {
				return err
			}

			if len(result.Findings) > 0 {
				if err := lint.RenderFindings(os.Stdout, result.Findings, lint.StdoutSupportsColor()); err != nil {
					return fmt.Errorf("failed to render findings: %w", err)
				}
			}

			if reportPath := cmd.String("codeclimate-report"); reportPath != "" {
				if err := lint.WriteCodeQualityReport(reportPath, result.CodeQuality); err != nil {
					return fmt.Errorf("failed to write code quality report: %w", err)
				}
				slog.Info("Wrote code quality report", "path", reportPath, "entries", len(result.CodeQuality))
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
