package lint

import (
	"os"
	"path/filepath"

	"github.com/ContainerHive/ContainerHive/internal/hadolint"
)

// WriteCodeQualityReport serialises entries to GitLab Code Quality JSON and
// writes them to path, creating any missing parent directories.
func WriteCodeQualityReport(path string, entries []hadolint.CodeQualityEntry) error {
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

// readDockerfile reads a Dockerfile from disk. Split out so tests can swap in
// fixtures without touching the filesystem.
func readDockerfile(path string) ([]byte, error) {
	return os.ReadFile(path)
}
