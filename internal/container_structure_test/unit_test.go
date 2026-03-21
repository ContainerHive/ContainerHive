package container_structure_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GoogleContainerTools/container-structure-test/pkg/types/unversioned"
)

func TestIsTar(t *testing.T) {
	tests := []struct {
		name     string
		image    string
		expected bool
	}{
		{name: "tar extension", image: "image.tar", expected: true},
		{name: "yml extension", image: "config.yml", expected: false},
		{name: "no extension", image: "myimage", expected: false},
		{name: "empty string", image: "", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := &TestRunner{Image: tt.image}
			if got := runner.isTar(); got != tt.expected {
				t.Errorf("isTar() = %v, want %v for image %q", got, tt.expected, tt.image)
			}
		})
	}
}

func TestIsTar_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		image    string
		expected bool
	}{
		{name: "tar.gz extension", image: "image.tar.gz", expected: false},
		{name: "absolute path to tar", image: "/path/to/image.tar", expected: true},
		{name: "uppercase TAR", image: "image.TAR", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := &TestRunner{Image: tt.image}
			if got := runner.isTar(); got != tt.expected {
				t.Errorf("isTar() = %v, want %v for image %q", got, tt.expected, tt.image)
			}
		})
	}
}

func TestGetOptions(t *testing.T) {
	runner := &TestRunner{
		TestDefinitionPaths: []string{"/tmp/test1.yml", "/tmp/test2.yml"},
		Image:               "myapp:latest",
		Platform:            "linux/amd64",
		ReportFile:          "/tmp/report.xml",
	}

	opts := runner.getOptions(unversioned.Junit)

	if opts.ImagePath != runner.Image {
		t.Errorf("ImagePath = %q, want %q", opts.ImagePath, runner.Image)
	}

	if len(opts.ConfigFiles) != len(runner.TestDefinitionPaths) {
		t.Fatalf("ConfigFiles length = %d, want %d", len(opts.ConfigFiles), len(runner.TestDefinitionPaths))
	}
	for i, cf := range opts.ConfigFiles {
		if cf != runner.TestDefinitionPaths[i] {
			t.Errorf("ConfigFiles[%d] = %q, want %q", i, cf, runner.TestDefinitionPaths[i])
		}
	}

	if opts.Platform != runner.Platform {
		t.Errorf("Platform = %q, want %q", opts.Platform, runner.Platform)
	}

	if opts.Driver != "docker" {
		t.Errorf("Driver = %q, want %q", opts.Driver, "docker")
	}

	if !opts.Quiet {
		t.Error("Quiet = false, want true")
	}

	if !opts.JSON {
		t.Error("JSON = false, want true")
	}

	if opts.Output != unversioned.Junit {
		t.Errorf("Output = %v, want %v", opts.Output, unversioned.Junit)
	}
}

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

func TestCollectTestDefinitions_OnlyFiles(t *testing.T) {
	dir := t.TempDir()
	testsDir := filepath.Join(dir, "tests")
	if err := os.MkdirAll(testsDir, 0755); err != nil {
		t.Fatal(err)
	}

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

	for _, d := range defs {
		if !filepath.IsAbs(d) && !strings.HasPrefix(d, dir) {
			t.Errorf("expected path under %s, got %s", dir, d)
		}
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
