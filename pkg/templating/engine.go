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

// FuncMapWithOptions returns the base function map plus an "option" function
// that looks up keys in the provided options map.
func FuncMapWithOptions(options map[string]string) template.FuncMap {
	funcs := funcMap()
	funcs["option"] = func(key string) string {
		if options == nil {
			return ""
		}
		return options[key]
	}
	return funcs
}

// RenderWithOptions renders a template from an fs.FS with template options available via the option function.
func RenderWithOptions(templateFS fs.FS, entrypoint string, data any, options map[string]string) ([]byte, error) {
	tpl, err := template.New(entrypoint).Funcs(FuncMapWithOptions(options)).ParseFS(templateFS, "*.gotpl")
	if err != nil {
		return nil, fmt.Errorf("failed to parse templates: %w", err)
	}

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("failed to execute template: %w", err)
	}
	return buf.Bytes(), nil
}

// RenderStringWithOptions renders a single template string with template options available via the option function.
func RenderStringWithOptions(name string, content string, data any, options map[string]string) ([]byte, error) {
	tpl, err := template.New(name).Funcs(FuncMapWithOptions(options)).Parse(content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("failed to execute template: %w", err)
	}
	return buf.Bytes(), nil
}
