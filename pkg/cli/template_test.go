package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ContainerHive/ContainerHive/pkg/ci"
)

func TestOutputPathIsPerImage(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		perImage bool
		wantErr  bool
	}{
		{name: "plain path", output: "out/result.yml", perImage: false},
		{name: "image field", output: "foo/{{ .Image.Name }}.yml", perImage: true},
		{name: "image field chained", output: "{{ .Image.Name }}/{{ .Image.Depth }}.yml", perImage: true},
		{name: "dollar image", output: "foo/{{ $.Image.Name }}.yml", perImage: true},
		{name: "assigned variable", output: "foo/{{ $i := .Image }}{{ $i.Name }}.yml", perImage: true},
		{name: "images field is not image", output: "out/{{ len .Images }}.yml", perImage: false},
		{name: "literal text", output: "out/report.Image.yml", perImage: false},
		{name: "other object field", output: "out/{{ .Foo.Image }}.yml", perImage: false},
		{name: "image in range body", output: "out/{{ range .Stages }}{{ end }}.yml", perImage: false},
		{name: "invalid template", output: "out/{{", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := outputPathIsPerImage(tt.output, nil)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.perImage {
				t.Errorf("outputPathIsPerImage(%q) = %v, want %v", tt.output, got, tt.perImage)
			}
		})
	}
}

func TestTemplateCustom_OutputPerImage(t *testing.T) {
	tmpDir := t.TempDir()
	copyDir(t, "../testdata/minimal-project", tmpDir)

	apiDir := filepath.Join(tmpDir, "images", "api")
	if err := os.MkdirAll(apiDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(apiDir, "image.yml"), []byte("tags:\n  - name: \"1.0\"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(apiDir, "Dockerfile"), []byte("FROM alpine\n"), 0644); err != nil {
		t.Fatal(err)
	}

	templatePath := filepath.Join(tmpDir, "image.gotpl")
	content := "image: {{ .Image.Name }}\ntotal: {{ len .Images }}\n"
	if err := os.WriteFile(templatePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	output := filepath.Join(tmpDir, "out", "{{ .Image.Name }}.yml")
	app := NewApp()
	args := []string{"ch", "--project", tmpDir, "template", "custom", "--template", templatePath, "--output", output}
	if err := app.Run(t.Context(), args); err != nil {
		t.Fatalf("template custom failed: %v", err)
	}

	for _, name := range []string{"nginx", "api"} {
		path := filepath.Join(tmpDir, "out", name+".yml")
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("expected output file %s: %v", path, err)
		}
		got := string(data)
		if !strings.Contains(got, "image: "+name) {
			t.Errorf("file %s does not contain image name, got: %q", path, got)
		}
		if !strings.Contains(got, "total: 2") {
			t.Errorf("file %s does not contain promoted CIContext field, got: %q", path, got)
		}
	}
}

func TestTemplateCustom_OutputSingleRender(t *testing.T) {
	tmpDir := t.TempDir()
	copyDir(t, "../testdata/minimal-project", tmpDir)

	templatePath := filepath.Join(tmpDir, "all.gotpl")
	if err := os.WriteFile(templatePath, []byte("{{ range .Images }}{{ .Name }} {{ end }}"), 0644); err != nil {
		t.Fatal(err)
	}

	output := filepath.Join(tmpDir, "out", "all-{{ len .Images }}.yml")
	app := NewApp()
	args := []string{"ch", "--project", tmpDir, "template", "custom", "--template", templatePath, "--output", output}
	if err := app.Run(t.Context(), args); err != nil {
		t.Fatalf("template custom failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(tmpDir, "out", "all-1.yml"))
	if err != nil {
		t.Fatalf("expected single rendered output file: %v", err)
	}
	if !strings.Contains(string(data), "nginx") {
		t.Errorf("unexpected content: %q", string(data))
	}
}

func TestWritePerImageOutputs_NoImages(t *testing.T) {
	ciCtx := &ci.CIContext{}
	err := writePerImageOutputs("out/{{ .Image.Name }}.yml", "t.gotpl", "x", ciCtx)
	if err == nil {
		t.Fatal("expected error for project without images, got nil")
	}
	if !strings.Contains(err.Error(), "no images") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestWriteOutput_CreatesParentDirs(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "nested", "deeper", "result.yml")
	if err := writeOutput(path, []byte("data")); err != nil {
		t.Fatalf("writeOutput failed: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("expected file to be created: %v", err)
	}
	if string(got) != "data" {
		t.Errorf("got %q, want %q", string(got), "data")
	}
}
