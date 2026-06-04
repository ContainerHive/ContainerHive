package lint

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/ContainerHive/ContainerHive/internal/hadolint"
)

type CodeClimateFormat struct {
	path string
}

func NewCodeClimateFormat(path string) *CodeClimateFormat {
	return &CodeClimateFormat{path: path}
}

func (f *CodeClimateFormat) HasPath() bool { return true }

func (f *CodeClimateFormat) Name() string {
	return "codeclimate"
}

func (f *CodeClimateFormat) Render(_ io.Writer, findings []Finding) error {
	entries := make([]hadolint.CodeQualityEntry, len(findings))
	for i, r := range findings {
		entries[i] = hadolint.ToCodeQuality(r.Finding, r.Path)
	}

	data, err := hadolint.MarshalCodeQuality(entries)
	if err != nil {
		return fmt.Errorf("marshal code quality: %w", err)
	}

	if dir := filepath.Dir(f.path); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create directory: %w", err)
		}
	}

	if err := os.WriteFile(f.path, data, 0o644); err != nil {
		return fmt.Errorf("write code quality report: %w", err)
	}

	return nil
}
