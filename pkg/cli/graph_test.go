package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/ContainerHive/ContainerHive/pkg/deps"
	"github.com/ContainerHive/ContainerHive/pkg/model"
)

func TestValidateGraphFormat(t *testing.T) {
	tests := []struct {
		format  string
		wantErr bool
	}{
		{format: "mermaid"},
		{format: "dot"},
		{format: "svg", wantErr: true},
		{format: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			err := validateGraphFormat(tt.format)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateGraphFormat(%q) error = %v, wantErr %v", tt.format, err, tt.wantErr)
			}
		})
	}
}

func TestWriteGraph(t *testing.T) {
	project := &model.ContainerHiveProject{
		ImagesByName: map[string][]*model.Image{
			"base": {{Name: "base"}},
			"app":  {{Name: "app", DependsOn: []string{"base"}}},
		},
	}
	graph := deps.ProjectGraph(project)

	tests := []struct {
		format string
		want   string
	}{
		{format: "mermaid", want: "graph LR"},
		{format: "dot", want: "digraph containerhive"},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			var out bytes.Buffer
			if err := writeGraph(&out, graph, tt.format); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !strings.Contains(out.String(), tt.want) {
				t.Errorf("output %q does not contain %q", out.String(), tt.want)
			}
		})
	}

	t.Run("unknown format", func(t *testing.T) {
		var out bytes.Buffer
		if err := writeGraph(&out, graph, "svg"); err == nil {
			t.Error("expected error for unknown format")
		}
	})
}
