// Package hadolint wraps github.com/timo-reymann/go-hadolint, translating
// the project-level model.LintConfig into hadolint's typed Config and exposing
// a small API used by the ch lint command.
package hadolint

import (
	"github.com/ContainerHive/ContainerHive/pkg/model"
	gohadolint "github.com/timo-reymann/go-hadolint"
)

// Linter lints Dockerfiles using the embedded hadolint binary (or a $PATH
// fallback when no embedded binary is available for the current platform).
type Linter struct {
	h *gohadolint.Hadolinter
}

// NewLinter constructs a Linter from an optional project lint config. A nil
// cfg means hadolint runs with its built-in defaults (no config discovery is
// short-circuited unless cfg is non-nil).
//
// The embedded hadolint binary is preferred. If the embedded binary fails to
// produce a version banner (e.g. when go-hadolint ships a binary for the
// wrong architecture on the current host), we fall back to a system hadolint
// on $PATH so the user still gets working lint output.
func NewLinter(cfg *model.LintConfig) (*Linter, error) {
	h, err := gohadolint.NewHadolinter()
	if err != nil {
		return nil, err
	}
	if v := h.Version(); v == "" {
		_ = h.Close()
		path, pathErr := gohadolint.NewHadolinterFromPATH()
		if pathErr != nil {
			return nil, pathErr
		}
		h = path
	}
	if cfg != nil {
		h.Config = toHadolintConfig(cfg)
	}
	return &Linter{h: h}, nil
}

// Close releases the embedded binary handle.
func (l *Linter) Close() error {
	if l.h == nil {
		return nil
	}
	return l.h.Close()
}

// Lint analyses a single Dockerfile on disk and returns the parsed result. The
// returned ExitCode reflects hadolint's own threshold evaluation, so callers
// can treat ExitCode != 0 as "this file failed lint" without re-implementing
// the severity ladder.
func (l *Linter) Lint(path string) (*gohadolint.Result, error) {
	return l.h.AnalyzeFile(path, "")
}

func toHadolintConfig(cfg *model.LintConfig) *gohadolint.Config {
	return &gohadolint.Config{
		Ignored:           cfg.Ignored,
		TrustedRegistries: cfg.TrustedRegistries,
		LabelSchema:       cfg.LabelSchema,
		StrictLabels:      cfg.StrictLabels,
		FailureThreshold:  cfg.FailureThreshold,
	}
}
