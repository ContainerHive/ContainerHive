package ci

import (
	"strings"
	"testing"
	"testing/fstest"
)

func TestGenerate_WithTemplateOptions(t *testing.T) {
	tplFS := fstest.MapFS{
		"main.gotpl": {Data: []byte(`image: {{ option "my_image" }}`)},
	}

	RegisterProvider(&Provider{
		Name:       "test-opts",
		TemplateFS: tplFS,
		Entrypoint: "main.gotpl",
	})

	ctx := &CIContext{
		TemplateOptions: map[string]string{"my_image": "custom/image"},
	}

	result, err := Generate("test-opts", ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	got := string(result)
	if got != "image: custom/image" {
		t.Errorf("got %q, want %q", got, "image: custom/image")
	}
}

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
		"main.gotpl": {Data: []byte("# Command: {{ .Command }}\n{{ range .Images }}Image: {{ .Name }}\n{{ end }}")},
	}

	RegisterProvider(&Provider{
		Name:       "test-gen",
		TemplateFS: tplFS,
		Entrypoint: "main.gotpl",
	})

	ctx := &CIContext{
		Command: "ch template ci --provider test-gen",
		Images: []CIImage{
			{Name: "myapp", Depth: 0, Platforms: []string{"amd64"}},
		},
	}

	result, err := Generate("test-gen", ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	got := string(result)
	if !strings.Contains(got, "Command: ch template ci --provider test-gen") {
		t.Errorf("result should contain command, got: %s", got)
	}
	if !strings.Contains(got, "Image: myapp") {
		t.Errorf("result should contain image name, got: %s", got)
	}
}

func TestResolveDefaultImageName(t *testing.T) {
	tests := []struct {
		provider string
		want     string
	}{
		{provider: "github", want: "ghcr.io/containerhive/containerhive"},
		{provider: "gitlab", want: "registry.gitlab.com/containerhive/containerhive"},
		{provider: "does-not-exist", want: "containerhive/containerhive"},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			if got := ResolveDefaultImageName(tt.provider); got != tt.want {
				t.Errorf("ResolveDefaultImageName(%q) = %q, want %q", tt.provider, got, tt.want)
			}
		})
	}
}
