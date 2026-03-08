package ci

import (
	"strings"
	"testing"
	"testing/fstest"
)

func TestProviderRegistry(t *testing.T) {
	t.Run("register and get", func(t *testing.T) {
		RegisterProvider(&Provider{
			Name:       "test-provider",
			TemplateFS: fstest.MapFS{},
			Entrypoint: "main.gotpl",
		})

		p, err := GetProvider("test-provider")
		if err != nil {
			t.Fatal(err)
		}
		if p.Name != "test-provider" {
			t.Errorf("expected name 'test-provider', got %q", p.Name)
		}
	})

	t.Run("unknown provider", func(t *testing.T) {
		_, err := GetProvider("nonexistent")
		if err == nil {
			t.Fatal("expected error for unknown provider")
		}
		if !strings.Contains(err.Error(), "unknown CI provider") {
			t.Errorf("error should mention 'unknown CI provider', got: %v", err)
		}
	})
}

func TestGenerate(t *testing.T) {
	tplFS := fstest.MapFS{
		"main.gotpl": {Data: []byte("# Generated: {{ .GeneratedAt }}\n{{ range .Images }}Image: {{ .Name }}\n{{ end }}")},
	}

	RegisterProvider(&Provider{
		Name:       "test-gen",
		TemplateFS: tplFS,
		Entrypoint: "main.gotpl",
	})

	ctx := &CIContext{
		GeneratedAt: "2024-01-01T00:00:00Z",
		Images: []CIImage{
			{Name: "myapp", Depth: 0, Platforms: []string{"amd64"}},
		},
	}

	result, err := Generate("test-gen", ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	got := string(result)
	if !strings.Contains(got, "Generated: 2024-01-01T00:00:00Z") {
		t.Errorf("result should contain generated timestamp, got: %s", got)
	}
	if !strings.Contains(got, "Image: myapp") {
		t.Errorf("result should contain image name, got: %s", got)
	}
}
