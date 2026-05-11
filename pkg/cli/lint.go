package cli

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/ContainerHive/ContainerHive/internal/file_resolver"
	"github.com/ContainerHive/ContainerHive/internal/hadolint"
	"github.com/ContainerHive/ContainerHive/pkg/model"
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
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			project, err := discoverProject(ctx, cmd)
			if err != nil {
				return err
			}

			cfg := resolveLintConfig(project.Config.Lint, cmd.String("failure-threshold"))

			linter, err := hadolint.NewLinter(cfg)
			if err != nil {
				return fmt.Errorf("failed to initialize hadolint: %w", err)
			}
			defer linter.Close()

			var (
				linted   int
				failures int
			)
			for _, img := range project.ImagesByIdentifier {
				failures += lintEntrypoint(linter, img.Name, "", img.BuildEntryPointPath, &linted)
				for _, variant := range img.Variants {
					failures += lintEntrypoint(linter, img.Name, variant.Name, variant.BuildEntryPointPath, &linted)
				}
			}

			if linted == 0 {
				slog.Info("No plain Dockerfiles to lint")
				return nil
			}

			if failures > 0 {
				return fmt.Errorf("hadolint reported findings in %d Dockerfile(s)", failures)
			}

			slog.Info("Lint passed", "files", linted)
			return nil
		},
	}
}

// resolveLintConfig folds the CLI failure-threshold override into the project
// lint config. Returns a non-nil config when either source supplies a value,
// so hadolint's default config discovery is short-circuited.
func resolveLintConfig(projectCfg *model.LintConfig, cliThreshold string) *model.LintConfig {
	if projectCfg == nil && cliThreshold == "" {
		// Neither configured — apply our documented default threshold of
		// "error" so failures stay predictable across hadolint versions.
		return &model.LintConfig{FailureThreshold: "error"}
	}

	out := &model.LintConfig{}
	if projectCfg != nil {
		*out = *projectCfg
	}
	if cliThreshold != "" {
		out.FailureThreshold = cliThreshold
	}
	if out.FailureThreshold == "" {
		out.FailureThreshold = "error"
	}
	return out
}

// lintEntrypoint runs hadolint against a single image or variant entrypoint.
// Returns 1 if the file failed lint, 0 otherwise (including when skipped).
func lintEntrypoint(linter *hadolint.Linter, imageName, variantName, path string, linted *int) int {
	logger := slog.With("image", imageName)
	if variantName != "" {
		logger = logger.With("variant", variantName)
	}
	logger = logger.With("path", path)

	if file_resolver.RemoveTemplateExt(path) != path {
		logger.Warn("Skipping templated Dockerfile (hadolint does not support templates)")
		return 0
	}

	if filepath.Base(path) != "Dockerfile" {
		logger.Warn("Skipping non-Dockerfile build entrypoint")
		return 0
	}

	res, err := linter.Lint(path)
	if err != nil {
		// AnalyzeFile returns an error only on infrastructure failures (binary
		// missing, JSON parse errors). Surface them so the user can act.
		logger.Error("hadolint invocation failed", "error", err)
		return 1
	}

	*linted++

	for _, f := range res.Findings {
		fmt.Printf("%s:%d:%d %s %s: %s\n", f.File, f.Line, f.Column, f.Level, f.Code, f.Message)
	}

	if res.ExitCode != 0 {
		return 1
	}
	return 0
}
