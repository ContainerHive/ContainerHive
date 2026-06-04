package lint

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	gohadolint "github.com/timo-reymann/go-hadolint"
)

func TestCodeClimateFormat_Render(t *testing.T) {
	t.Run("writes valid JSON", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "gl-code-quality-report.json")
		f := NewCodeClimateFormat(path)

		findings := []Finding{
			{
				Path: "images/test/Dockerfile",
				Finding: gohadolint.Finding{
					File:    "/abs/Dockerfile",
					Line:    7,
					Column:  3,
					Level:   "warning",
					Code:    "DL3006",
					Message: "Always tag the version of an image explicitly",
				},
			},
		}

		if err := f.Render(nil, findings); err != nil {
			t.Fatalf("render: %v", err)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read report: %v", err)
		}

		var entries []map[string]any
		if err := json.Unmarshal(data, &entries); err != nil {
			t.Fatalf("invalid JSON: %v\n%s", err, data)
		}

		if len(entries) != 1 {
			t.Fatalf("expected 1 entry, got %d", len(entries))
		}

		got := entries[0]
		if got["check_name"] != "DL3006" {
			t.Errorf("check_name = %v, want DL3006", got["check_name"])
		}
		if got["severity"] != "major" {
			t.Errorf("severity = %v, want major", got["severity"])
		}
	})

	t.Run("creates parent directory", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "sub", "report.json")
		f := NewCodeClimateFormat(path)

		findings := []Finding{
			{
				Path: "Dockerfile",
				Finding: gohadolint.Finding{
					Code:    "DL4000",
					Level:   "error",
					Line:    1,
					Column:  1,
					Message: "bad",
				},
			},
		}

		if err := f.Render(nil, findings); err != nil {
			t.Fatalf("render: %v", err)
		}

		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("report not created at %s", path)
		}
	})

	t.Run("empty findings", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "empty.json")
		f := NewCodeClimateFormat(path)

		if err := f.Render(nil, nil); err != nil {
			t.Fatalf("render: %v", err)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read report: %v", err)
		}

		var entries []map[string]any
		if err := json.Unmarshal(data, &entries); err != nil {
			t.Fatalf("invalid JSON for empty: %v\n%s", err, data)
		}
	})
}
