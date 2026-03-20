package cst

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCollectTestDefinitions_FilesFound(t *testing.T) {
	dir := t.TempDir()
	testsDir := filepath.Join(dir, "tests")
	if err := os.MkdirAll(testsDir, 0755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"test1.yaml", "test2.yaml"} {
		if err := os.WriteFile(filepath.Join(testsDir, name), []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	defs := CollectTestDefinitions(dir)
	if len(defs) != 2 {
		t.Errorf("expected 2 test definitions, got %d", len(defs))
	}
}

func TestCollectTestDefinitions_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	testsDir := filepath.Join(dir, "tests")
	if err := os.MkdirAll(testsDir, 0755); err != nil {
		t.Fatal(err)
	}

	defs := CollectTestDefinitions(dir)
	if len(defs) != 0 {
		t.Errorf("expected 0 test definitions, got %d", len(defs))
	}
}

func TestCollectTestDefinitions_MissingDir(t *testing.T) {
	defs := CollectTestDefinitions("/nonexistent/path")
	if defs != nil {
		t.Errorf("expected nil for missing dir, got %v", defs)
	}
}

func TestCollectTestDefinitions_SubdirsExcluded(t *testing.T) {
	dir := t.TempDir()
	testsDir := filepath.Join(dir, "tests")
	if err := os.MkdirAll(filepath.Join(testsDir, "subdir"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(testsDir, "test.yaml"), []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	defs := CollectTestDefinitions(dir)
	if len(defs) != 1 {
		t.Errorf("expected 1 test definition (subdir excluded), got %d", len(defs))
	}
}

func TestReportFileName(t *testing.T) {
	tests := []struct {
		reportDir string
		imageTag  string
		want      string
	}{
		{"reports", "app:1.0", filepath.Join("reports", "app-1.0-cst-report.xml")},
		{"/abs/reports", "my-img:latest", filepath.Join("/abs/reports", "my-img-latest-cst-report.xml")},
		{"reports", "app:1.0:extra", filepath.Join("reports", "app-1.0-extra-cst-report.xml")},
		{"reports", "no-colon", filepath.Join("reports", "no-colon-cst-report.xml")},
	}

	for _, tt := range tests {
		t.Run(tt.imageTag, func(t *testing.T) {
			got := ReportFileName(tt.reportDir, tt.imageTag)
			if got != tt.want {
				t.Errorf("ReportFileName(%q, %q) = %q, want %q", tt.reportDir, tt.imageTag, got, tt.want)
			}
		})
	}
}

func TestCollectTestDefinitions_OnlyFiles(t *testing.T) {
	dir := t.TempDir()
	testsDir := filepath.Join(dir, "tests")
	if err := os.MkdirAll(testsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Mix of files and subdirs
	for _, name := range []string{"a.yaml", "b.yml", "c.json"} {
		if err := os.WriteFile(filepath.Join(testsDir, name), []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(testsDir, "nested"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(testsDir, "nested", "deep.yaml"), []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	defs := CollectTestDefinitions(dir)
	if len(defs) != 3 {
		t.Errorf("expected 3 test definitions (files only), got %d: %v", len(defs), defs)
	}

	// Verify all paths are absolute and in the tests dir
	for _, d := range defs {
		if !filepath.IsAbs(d) && !strings.HasPrefix(d, dir) {
			t.Errorf("expected path under %s, got %s", dir, d)
		}
	}
}

func TestCollectTestDefinitions_NoTestsSubdir(t *testing.T) {
	// Dir exists but has no "tests" subdirectory
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "somefile.yaml"), []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	defs := CollectTestDefinitions(dir)
	if defs != nil {
		t.Errorf("expected nil when tests/ subdir doesn't exist, got %v", defs)
	}
}
