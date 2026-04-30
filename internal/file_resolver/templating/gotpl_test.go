package templating

import (
	"testing"

	"github.com/ContainerHive/ContainerHive/pkg/model"
)

func TestGoTemplateTemplatingProcessor_Process(t *testing.T) {
	processor := &GoTemplateTemplatingProcessor{}

	t.Run("renders simple version variable", func(t *testing.T) {
		ctx := &TemplateContext{
			Versions: model.Versions{"python": "3.11"},
		}

		got, err := processor.Process(ctx, "test.gotpl", []byte("Python {{.Versions.python}}"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(got) != "Python 3.11" {
			t.Errorf("expected %q, got %q", "Python 3.11", string(got))
		}
	})

	t.Run("renders build args", func(t *testing.T) {
		ctx := &TemplateContext{
			BuildArgs: model.BuildArgs{"BASE_IMAGE": "alpine:3.19"},
		}

		got, err := processor.Process(ctx, "test.gotpl", []byte("FROM {{.BuildArgs.BASE_IMAGE}}"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(got) != "FROM alpine:3.19" {
			t.Errorf("expected %q, got %q", "FROM alpine:3.19", string(got))
		}
	})

	t.Run("renders image name", func(t *testing.T) {
		ctx := &TemplateContext{
			ImageName: "myapp",
		}

		got, err := processor.Process(ctx, "test.gotpl", []byte("name={{.ImageName}}"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(got) != "name=myapp" {
			t.Errorf("expected %q, got %q", "name=myapp", string(got))
		}
	})

	t.Run("renders all context fields together", func(t *testing.T) {
		ctx := &TemplateContext{
			ImageName: "webapp",
			Versions:  model.Versions{"node": "20.0.0"},
			BuildArgs: model.BuildArgs{"ENV": "prod"},
		}
		tmpl := "{{.ImageName}}-{{.Versions.node}}-{{.BuildArgs.ENV}}"

		got, err := processor.Process(ctx, "test.gotpl", []byte(tmpl))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(got) != "webapp-20.0.0-prod" {
			t.Errorf("expected %q, got %q", "webapp-20.0.0-prod", string(got))
		}
	})

	t.Run("returns error for invalid template syntax", func(t *testing.T) {
		ctx := &TemplateContext{}

		_, err := processor.Process(ctx, "bad.gotpl", []byte("{{.Unclosed"))
		if err == nil {
			t.Fatal("expected parse error, got nil")
		}
	})

	t.Run("returns error for missing field in strict template", func(t *testing.T) {
		ctx := &TemplateContext{}

		_, err := processor.Process(ctx, "test.gotpl", []byte("{{.NonExistent.Field}}"))
		if err == nil {
			t.Fatal("expected execution error, got nil")
		}
	})

	t.Run("renders empty template", func(t *testing.T) {
		ctx := &TemplateContext{}

		got, err := processor.Process(ctx, "empty.gotpl", []byte(""))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(got) != "" {
			t.Errorf("expected empty string, got %q", string(got))
		}
	})

	t.Run("renders template with no variables", func(t *testing.T) {
		ctx := &TemplateContext{}

		got, err := processor.Process(ctx, "static.gotpl", []byte("static content"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(got) != "static content" {
			t.Errorf("expected %q, got %q", "static content", string(got))
		}
	})

	t.Run("renders resolve_base function", func(t *testing.T) {
		ctx := &TemplateContext{
			Versions: model.Versions{"go": "1.22"},
		}

		got, err := processor.Process(ctx, "test.gotpl", []byte(`FROM {{ resolve_base "ubuntu" "22.04" }}`))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(got) != "FROM __hive__/ubuntu:22.04" {
			t.Errorf("expected %q, got %q", "FROM __hive__/ubuntu:22.04", string(got))
		}
	})

	t.Run("renders resolve_base with version variable", func(t *testing.T) {
		ctx := &TemplateContext{
			Versions: model.Versions{"ubuntu_tag": "22.04"},
		}

		got, err := processor.Process(ctx, "test.gotpl", []byte(`FROM {{ resolve_base "ubuntu" .Versions.ubuntu_tag }}`))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(got) != "FROM __hive__/ubuntu:22.04" {
			t.Errorf("expected %q, got %q", "FROM __hive__/ubuntu:22.04", string(got))
		}
	})
}
