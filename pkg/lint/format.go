package lint

import (
	"fmt"
	"io"
	"strings"
)

type Format interface {
	Name() string
	Render(w io.Writer, findings []Finding) error
	HasPath() bool
}

type FormatOption struct {
	Name string
	Path string
}

var formatPrototypes = map[string]Format{
	"terminal":       &TerminalFormat{},
	"github-actions": &GitHubActionsFormat{},
	"codeclimate":    &CodeClimateFormat{},
}

func ParseFormats(flags []string) ([]FormatOption, error) {
	if len(flags) == 0 {
		return []FormatOption{{Name: "terminal"}}, nil
	}

	seen := make(map[string]bool)
	out := make([]FormatOption, 0, len(flags))
	for _, raw := range flags {
		name, path, _ := strings.Cut(raw, "=")
		if name == "" {
			return nil, fmt.Errorf("empty format name in %q", raw)
		}
		if seen[name] {
			return nil, fmt.Errorf("duplicate format %q", name)
		}
		seen[name] = true

		proto, ok := formatPrototypes[name]
		if !ok {
			return nil, fmt.Errorf("unknown format %q (supported: terminal, codeclimate, github-actions)", name)
		}

		if proto.HasPath() && path == "" {
			return nil, fmt.Errorf("format %q requires a path suffix (e.g. --format codeclimate=report.json)", name)
		}
		if !proto.HasPath() && path != "" {
			return nil, fmt.Errorf("format %q does not support a path suffix", name)
		}

		out = append(out, FormatOption{Name: name, Path: path})
	}
	return out, nil
}
