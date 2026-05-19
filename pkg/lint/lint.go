// Package lint runs hadolint over every plain Dockerfile in a ContainerHive
// project and renders the findings either as a CodeClimate/GitLab Code
// Quality report or as a colored text report for terminal output.
//
// The package owns the project-level lint workflow (config resolution,
// __hive_parent__ substitution, finding aggregation, report writing) so the
// CLI layer can stay a thin glue.
package lint

import (
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/ContainerHive/ContainerHive/internal/file_resolver"
	"github.com/ContainerHive/ContainerHive/internal/hadolint"
	"github.com/ContainerHive/ContainerHive/pkg/model"
	gohadolint "github.com/timo-reymann/go-hadolint"
)

// Finding bundles a hadolint finding with both the repo-relative path (used in
// the Code Quality report) and the absolute filesystem path (used in console
// output, which favours full paths so editors can click-through).
type Finding struct {
	Finding  gohadolint.Finding
	Path     string
	FullPath string
}

// Result is the outcome of linting a project: every parsed finding (regardless
// of severity), the matching Code Quality entries, the number of files that
// hadolint actually analysed, and how many of those failed at or above the
// configured failure threshold.
type Result struct {
	Findings       []Finding
	CodeQuality    []hadolint.CodeQualityEntry
	LintedCount    int
	FailedCount    int
}

// Linter wraps the embedded hadolint binary with the project's resolved
// LintConfig. Close releases the binary handle.
type Linter struct {
	cfg *model.LintConfig
	h   *hadolint.Linter
}

// NewLinter constructs a Linter from a resolved project lint config. Pass nil
// to fall back to hadolint's built-in defaults.
func NewLinter(cfg *model.LintConfig) (*Linter, error) {
	h, err := hadolint.NewLinter(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize hadolint: %w", err)
	}
	return &Linter{cfg: cfg, h: h}, nil
}

// Close releases the underlying hadolint binary handle.
func (l *Linter) Close() error {
	if l.h == nil {
		return nil
	}
	return l.h.Close()
}

// Lint runs hadolint over every plain Dockerfile entrypoint in the project
// (skipping templated entrypoints) and returns a Result describing all
// findings encountered.
//
// projectRoot must be the discovered, symlink-resolved root that hadolint also
// resolves to, so paths in the report match between sources.
func (l *Linter) Lint(project *model.ContainerHiveProject, projectRoot string) (*Result, error) {
	res := &Result{}
	for _, img := range project.ImagesByIdentifier {
		parentRef := BuildHiveParentRef(img)
		if err := l.lintEntrypoint(img.Name, "", img.BuildEntryPointPath, projectRoot, parentRef, res); err != nil {
			return nil, err
		}
		for _, variant := range img.Variants {
			if err := l.lintEntrypoint(img.Name, variant.Name, variant.BuildEntryPointPath, projectRoot, parentRef, res); err != nil {
				return nil, err
			}
		}
	}
	return res, nil
}

// lintEntrypoint runs hadolint against a single image or variant entrypoint
// and appends any findings to res. Returns an error only for infrastructure
// failures (binary missing, JSON parse errors); finding-level violations are
// recorded in res.FailedCount.
func (l *Linter) lintEntrypoint(imageName, variantName, path, projectRoot, parentRef string, res *Result) error {
	logger := slog.With("image", imageName)
	if variantName != "" {
		logger = logger.With("variant", variantName)
	}
	logger = logger.With("path", path)

	if file_resolver.RemoveTemplateExt(path) != path {
		logger.Warn("Skipping templated Dockerfile (hadolint does not support templates)")
		return nil
	}

	if filepath.Base(path) != "Dockerfile" {
		logger.Warn("Skipping non-Dockerfile build entrypoint")
		return nil
	}

	content, err := readDockerfile(path)
	if err != nil {
		logger.Error("failed to read Dockerfile", "error", err)
		res.FailedCount++
		return nil
	}
	content = SubstituteHiveParent(content, parentRef)

	out, err := l.h.LintSnippet(content)
	if err != nil {
		logger.Error("hadolint invocation failed", "error", err)
		res.FailedCount++
		return nil
	}

	res.LintedCount++

	displayPath := relativeReportPath(projectRoot, path)
	fullPath, fpErr := filepath.Abs(path)
	if fpErr != nil {
		fullPath = path
	}
	for _, f := range out.Findings {
		res.CodeQuality = append(res.CodeQuality, hadolint.ToCodeQuality(f, displayPath))
		res.Findings = append(res.Findings, Finding{Finding: f, Path: displayPath, FullPath: fullPath})
	}

	if out.ExitCode != 0 {
		res.FailedCount++
	}
	return nil
}

// ResolveConfig folds the CLI failure-threshold override into the project
// lint config. Returns a non-nil config when either source supplies a value
// so hadolint's default config discovery is short-circuited. The caller's
// projectCfg is never mutated.
func ResolveConfig(projectCfg *model.LintConfig, cliThreshold string) *model.LintConfig {
	if projectCfg == nil && cliThreshold == "" {
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
