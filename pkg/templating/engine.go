package templating

import (
	"bytes"
	"fmt"
	"io/fs"
	"text/template"

	sprig "github.com/Masterminds/sprig/v3"
)

func funcMap() template.FuncMap {
	funcs := sprig.TxtFuncMap()
	funcs["resolve_base"] = func(name, tag string) string {
		return fmt.Sprintf("__hive__/%s:%s", name, tag)
	}
	return funcs
}

// Render renders a template from an fs.FS, supporting partials via ParseFS.
func Render(templateFS fs.FS, entrypoint string, data any) ([]byte, error) {
	tpl, err := template.New(entrypoint).Funcs(funcMap()).ParseFS(templateFS, "*.gotpl")
	if err != nil {
		return nil, fmt.Errorf("failed to parse templates: %w", err)
	}

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("failed to execute template: %w", err)
	}
	return buf.Bytes(), nil
}

// RenderString renders a single template string.
func RenderString(name string, content string, data any) ([]byte, error) {
	tpl, err := template.New(name).Funcs(funcMap()).Parse(content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("failed to execute template: %w", err)
	}
	return buf.Bytes(), nil
}
