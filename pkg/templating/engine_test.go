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
