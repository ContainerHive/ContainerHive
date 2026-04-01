package templating

import (
	"strings"
	"testing"
	"testing/fstest"
)

func TestRenderString(t *testing.T) {
	t.Run("simple template", func(t *testing.T) {
		result, err := RenderString("test", "Hello {{ .Name }}!", map[string]string{"Name": "World"})
		if err != nil {
			t.Fatal(err)
		}
		if got := string(result); got != "Hello World!" {
			t.Errorf("got %q, want %q", got, "Hello World!")
		}
	})

	t.Run("sprig functions available", func(t *testing.T) {
		result, err := RenderString("test", `{{ "hello" | upper }}`, nil)
		if err != nil {
			t.Fatal(err)
		}
		if got := string(result); got != "HELLO" {
			t.Errorf("got %q, want %q", got, "HELLO")
		}
	})

	t.Run("resolve_base helper", func(t *testing.T) {
		result, err := RenderString("test", `{{ resolve_base "myimage" "latest" }}`, nil)
		if err != nil {
			t.Fatal(err)
		}
		if got := string(result); got != "__hive__/myimage:latest" {
			t.Errorf("got %q, want %q", got, "__hive__/myimage:latest")
		}
	})

	t.Run("invalid template", func(t *testing.T) {
		_, err := RenderString("test", "{{ .Invalid", nil)
		if err == nil {
			t.Fatal("expected error for invalid template")
		}
	})
}

func TestRenderStringWithOptions(t *testing.T) {
	t.Run("option function returns value", func(t *testing.T) {
		opts := map[string]string{"foo": "bar"}
		result, err := RenderStringWithOptions("test", `{{ option "foo" }}`, nil, opts)
		if err != nil {
			t.Fatal(err)
		}
		if got := string(result); got != "bar" {
			t.Errorf("got %q, want %q", got, "bar")
		}
	})

	t.Run("option function returns empty for unknown key", func(t *testing.T) {
		opts := map[string]string{"foo": "bar"}
		result, err := RenderStringWithOptions("test", `{{ option "missing" }}`, nil, opts)
		if err != nil {
			t.Fatal(err)
		}
		if got := string(result); got != "" {
			t.Errorf("got %q, want %q", got, "")
		}
	})

	t.Run("option function works with if", func(t *testing.T) {
		opts := map[string]string{"present": "yes"}
		result, err := RenderStringWithOptions("test", `{{ if option "present" }}found{{ end }}`, nil, opts)
		if err != nil {
			t.Fatal(err)
		}
		if got := string(result); got != "found" {
			t.Errorf("got %q, want %q", got, "found")
		}
	})

	t.Run("nil options map", func(t *testing.T) {
		result, err := RenderStringWithOptions("test", `{{ option "any" }}`, nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		if got := string(result); got != "" {
			t.Errorf("got %q, want %q", got, "")
		}
	})

	t.Run("sprig and option coexist", func(t *testing.T) {
		opts := map[string]string{"name": "world"}
		result, err := RenderStringWithOptions("test", `{{ option "name" | upper }}`, nil, opts)
		if err != nil {
			t.Fatal(err)
		}
		if got := string(result); got != "WORLD" {
			t.Errorf("got %q, want %q", got, "WORLD")
		}
	})
}

func TestRenderWithOptions(t *testing.T) {
	t.Run("option function in partials", func(t *testing.T) {
		fsys := fstest.MapFS{
			"main.gotpl":    {Data: []byte(`{{ template "partial.gotpl" . }}`)},
			"partial.gotpl": {Data: []byte(`image: {{ option "ci_buildkit_image" }}`)},
		}
		opts := map[string]string{"ci_buildkit_image": "moby/buildkit"}
		result, err := RenderWithOptions(fsys, "main.gotpl", nil, opts)
		if err != nil {
			t.Fatal(err)
		}
		if got := string(result); got != "image: moby/buildkit" {
			t.Errorf("got %q, want %q", got, "image: moby/buildkit")
		}
	})
}

func TestRender(t *testing.T) {
	t.Run("with partials", func(t *testing.T) {
		fsys := fstest.MapFS{
			"main.gotpl":    {Data: []byte("Header\n{{ template \"partial.gotpl\" . }}\nFooter")},
			"partial.gotpl": {Data: []byte("Middle: {{ .Name }}")},
		}

		result, err := Render(fsys, "main.gotpl", map[string]string{"Name": "Test"})
		if err != nil {
			t.Fatal(err)
		}
		got := string(result)
		for _, want := range []string{"Header", "Middle: Test", "Footer"} {
			if !strings.Contains(got, want) {
				t.Errorf("result %q does not contain %q", got, want)
			}
		}
	})

	t.Run("missing entrypoint", func(t *testing.T) {
		fsys := fstest.MapFS{
			"other.gotpl": {Data: []byte("test")},
		}

		_, err := Render(fsys, "missing.gotpl", nil)
		if err == nil {
			t.Fatal("expected error for missing entrypoint")
		}
	})
}
