package cli

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/ContainerHive/ContainerHive/internal/file_resolver"
	"github.com/ContainerHive/ContainerHive/internal/hadolint"
	"github.com/ContainerHive/ContainerHive/pkg/model"
	"github.com/Ladicle/tabwriter"
	gohadolint "github.com/timo-reymann/go-hadolint"
	"github.com/urfave/cli/v3"
	"golang.org/x/term"
)

// ANSI codes matching the palette already used by pkg/logging and
// pkg/progress so lint findings blend in with the rest of ch's output.
const (
	ansiReset        = "\x1b[0m"
	ansiBold         = "\x1b[1m"
	ansiFaint        = "\x1b[2m"
	ansiBrightRed    = "\x1b[91m"
	ansiBrightYellow = "\x1b[93m"
	ansiBrightCyan   = "\x1b[96m"
)

// hadolintWikiBase is the documentation root for hadolint rule codes. Each
// finding's check_name (e.g. DL3006) appends as-is.
const hadolintWikiBase = "https://github.com/hadolint/hadolint/wiki/"

// tableFinding bundles a hadolint finding with both the repo-relative path
// (for the CodeClimate report) and the absolute filesystem path (for the
// console output, which favours full paths to support editor click-through).
type tableFinding struct {
	finding  gohadolint.Finding
	path     string
	fullPath string
}

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

			cfg := resolveLintConfig(project.Config.Lint, cmd.String("failure-threshold"))

			linter, err := hadolint.NewLinter(cfg)
			if err != nil {
				return fmt.Errorf("failed to initialize hadolint: %w", err)
			}
			defer linter.Close()

			// Use the discovered project root (already absolute) and resolve any
			// symlinks so that paths in the report match the paths hadolint sees,
			// which discovery resolves via filepath.EvalSymlinks.
			projectRoot := project.RootDir
			if resolved, evalErr := filepath.EvalSymlinks(projectRoot); evalErr == nil {
				projectRoot = resolved
			}
			reportPath := cmd.String("codeclimate-report")

			var (
				linted   int
				failures int
				report   []hadolint.CodeQualityEntry
				rows     []tableFinding
			)
			for _, img := range project.ImagesByIdentifier {
				failures += lintEntrypoint(linter, img.Name, "", img.BuildEntryPointPath, projectRoot, &linted, &report, &rows)
				for _, variant := range img.Variants {
					failures += lintEntrypoint(linter, img.Name, variant.Name, variant.BuildEntryPointPath, projectRoot, &linted, &report, &rows)
				}
			}

			if len(rows) > 0 {
				if err := renderFindingsTable(os.Stdout, rows, stdoutSupportsColor()); err != nil {
					return fmt.Errorf("failed to render findings table: %w", err)
				}
			}

			if reportPath != "" {
				if err := writeCodeQualityReport(reportPath, report); err != nil {
					return fmt.Errorf("failed to write code quality report: %w", err)
				}
				slog.Info("Wrote code quality report", "path", reportPath, "entries", len(report))
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
// Returns 1 if the file failed lint, 0 otherwise (including when skipped). It
// appends every parsed finding (regardless of failure) to *report and *rows so
// the resulting artifact and table reflect what hadolint actually saw, even
// for findings below the failure threshold.
func lintEntrypoint(linter *hadolint.Linter, imageName, variantName, path, projectRoot string, linted *int, report *[]hadolint.CodeQualityEntry, rows *[]tableFinding) int {
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

	displayPath := relativeReportPath(projectRoot, path)
	fullPath, fpErr := filepath.Abs(path)
	if fpErr != nil {
		fullPath = path
	}
	for _, f := range res.Findings {
		*report = append(*report, hadolint.ToCodeQuality(f, displayPath))
		*rows = append(*rows, tableFinding{finding: f, path: displayPath, fullPath: fullPath})
	}

	if res.ExitCode != 0 {
		return 1
	}
	return 0
}

// renderFindingsTable writes findings to w in the text/key-value layout used
// by gitlab-ci-verify (each finding is a small block with bold labels and a
// blank line between entries).
func renderFindingsTable(w io.Writer, rows []tableFinding, color bool) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', tabwriter.TabIndent)
	for i, r := range rows {
		f := r.finding
		entries := [][2]string{
			{"Code", f.Code},
			{"Severity", formatLevel(f.Level, color)},
			{"Location", fmt.Sprintf("%s:%d:%d", r.fullPath, f.Line, f.Column)},
			{"Link", hadolintWikiBase + f.Code},
			{"Description", f.Message},
		}
		for _, e := range entries {
			if _, err := fmt.Fprintf(tw, "%s\t%s\n", boldLabel(e[0], color), e[1]); err != nil {
				return err
			}
		}
		if err := tw.Flush(); err != nil {
			return err
		}
		if i < len(rows)-1 {
			if _, err := fmt.Fprintln(w); err != nil {
				return err
			}
		}
	}
	return nil
}

// boldLabel wraps a label in the ANSI bold escape when color is enabled.
func boldLabel(label string, color bool) string {
	if !color {
		return label
	}
	return ansiBold + label + ansiReset
}

// relativeReportPath returns path relative to projectRoot so the report entry
// matches what a GitLab runner expects (repo-relative paths). Falls back to
// the absolute path if it can't be made relative.
func relativeReportPath(projectRoot, path string) string {
	abs, err := filepath.Abs(projectRoot)
	if err != nil {
		return path
	}
	rel, err := filepath.Rel(abs, path)
	if err != nil {
		return path
	}
	return filepath.ToSlash(rel)
}

// stdoutSupportsColor reports whether ANSI color escapes are appropriate on
// the current stdout: stdout must be a terminal and the NO_COLOR convention
// (https://no-color.org) must not be set.
func stdoutSupportsColor() bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	return term.IsTerminal(int(os.Stdout.Fd()))
}

// formatLevel renders an uppercased hadolint severity, optionally wrapped in
// ANSI color escapes that match the rest of ch's output palette.
func formatLevel(level string, color bool) string {
	upper := strings.ToUpper(level)
	if !color {
		return upper
	}
	var code string
	switch level {
	case "error":
		code = ansiBrightRed
	case "warning":
		code = ansiBrightYellow
	case "info":
		code = ansiBrightCyan
	case "style":
		code = ansiFaint
	default:
		return upper
	}
	return code + upper + ansiReset
}

func writeCodeQualityReport(path string, entries []hadolint.CodeQualityEntry) error {
	data, err := hadolint.MarshalCodeQuality(entries)
	if err != nil {
		return err
	}
	if dir := filepath.Dir(path); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return os.WriteFile(path, data, 0o644)
}
