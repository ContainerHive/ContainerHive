package file_resolver

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ContainerHive/ContainerHive/internal/file_resolver/templating"
	"github.com/ContainerHive/ContainerHive/pkg/model"
)

func TestCopyAndRenderFile(t *testing.T) {
	t.Run("copies plain file without template extension", func(t *testing.T) {
		dir := t.TempDir()
		src := filepath.Join(dir, "Dockerfile")
		target := filepath.Join(dir, "out", "Dockerfile")
		os.Mkdir(filepath.Join(dir, "out"), 0755)

		os.WriteFile(src, []byte("FROM alpine\nRUN echo hello"), 0644)

		err := CopyAndRenderFile(nil, src, target)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got, err := os.ReadFile(target)
		if err != nil {
			t.Fatalf("failed to read target: %v", err)
		}
		if string(got) != "FROM alpine\nRUN echo hello" {
			t.Errorf("expected plain copy, got %q", string(got))
		}
	})

	t.Run("renders gotpl template with context", func(t *testing.T) {
		dir := t.TempDir()
		src := filepath.Join(dir, "test.yml.gotpl")
		target := filepath.Join(dir, "test.yml")

		os.WriteFile(src, []byte("version: {{.Versions.python}}"), 0644)

		ctx := &templating.TemplateContext{
			Versions: model.Versions{"python": "3.11"},
		}

		err := CopyAndRenderFile(ctx, src, target)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got, err := os.ReadFile(target)
		if err != nil {
			t.Fatalf("failed to read target: %v", err)
		}
		if string(got) != "version: 3.11" {
			t.Errorf("expected rendered template, got %q", string(got))
		}
	})

	t.Run("copies file with unknown extension as-is", func(t *testing.T) {
		dir := t.TempDir()
		src := filepath.Join(dir, "config.toml")
		target := filepath.Join(dir, "out.toml")

		content := "[server]\nport = 8080"
		os.WriteFile(src, []byte(content), 0644)

		err := CopyAndRenderFile(nil, src, target)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got, err := os.ReadFile(target)
		if err != nil {
			t.Fatalf("failed to read target: %v", err)
		}
		if string(got) != content {
			t.Errorf("expected plain copy, got %q", string(got))
		}
	})

	t.Run("copies file with single-char extension as-is", func(t *testing.T) {
		dir := t.TempDir()
		src := filepath.Join(dir, "file.a")
		target := filepath.Join(dir, "out.a")

		content := "short ext"
		os.WriteFile(src, []byte(content), 0644)

		err := CopyAndRenderFile(nil, src, target)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got, err := os.ReadFile(target)
		if err != nil {
			t.Fatalf("failed to read target: %v", err)
		}
		if string(got) != content {
			t.Errorf("expected plain copy, got %q", string(got))
		}
	})

	t.Run("returns error when source file does not exist", func(t *testing.T) {
		dir := t.TempDir()
		src := filepath.Join(dir, "nonexistent.gotpl")
		target := filepath.Join(dir, "out")

		ctx := &templating.TemplateContext{}

		err := CopyAndRenderFile(ctx, src, target)
		if err == nil {
			t.Fatal("expected error for missing source file, got nil")
		}
	})

	t.Run("returns error for invalid template syntax", func(t *testing.T) {
		dir := t.TempDir()
		src := filepath.Join(dir, "bad.gotpl")
		target := filepath.Join(dir, "out")

		os.WriteFile(src, []byte("{{.Unclosed"), 0644)

		ctx := &templating.TemplateContext{}

		err := CopyAndRenderFile(ctx, src, target)
		if err == nil {
			t.Fatal("expected error for invalid template, got nil")
		}
	})

	t.Run("renders template with all context fields", func(t *testing.T) {
		dir := t.TempDir()
		src := filepath.Join(dir, "full.gotpl")
		target := filepath.Join(dir, "full.out")

		tmpl := "image={{.ImageName}} python={{.Versions.python}} base={{.BuildArgs.BASE_IMAGE}}"
		os.WriteFile(src, []byte(tmpl), 0644)

		ctx := &templating.TemplateContext{
			ImageName: "myapp",
			Versions:  model.Versions{"python": "3.12"},
			BuildArgs: model.BuildArgs{"BASE_IMAGE": "alpine:3.19"},
		}

		err := CopyAndRenderFile(ctx, src, target)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got, err := os.ReadFile(target)
		if err != nil {
			t.Fatalf("failed to read target: %v", err)
		}
		expected := "image=myapp python=3.12 base=alpine:3.19"
		if string(got) != expected {
			t.Errorf("expected %q, got %q", expected, string(got))
		}
	})
}
