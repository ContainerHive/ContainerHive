package deps

import (
	"strings"
	"testing"

	"github.com/ContainerHive/ContainerHive/pkg/model"
)

func TestGraphAccessorsReturnCopies(t *testing.T) {
	graph := testGraph()

	nodes := graph.Nodes()
	nodes[0] = "changed"
	edges := graph.Edges()
	edges["app"][0] = "changed"
	delete(edges, "app/slim")

	if got := graph.Nodes()[0]; got != "app" {
		t.Errorf("Nodes()[0] = %q after mutating a copy, want app", got)
	}
	if got := graph.Edges()["app"][0]; got != "base" {
		t.Errorf("Edges()[app][0] = %q after mutating a copy, want base", got)
	}
	if _, ok := graph.Edges()["app/slim"]; !ok {
		t.Error("Edges() lost app/slim after deleting from a copy")
	}
}

func TestProjectGraphGroupsNestedImageNames(t *testing.T) {
	project := &model.ContainerHiveProject{
		ImagesByName: map[string][]*model.Image{
			"a/b": {{
				Name:     "a/b",
				Variants: map[string]*model.ImageVariant{"slim": {}},
			}},
			"a": {{Name: "a"}},
		},
	}

	tests := []struct {
		name   string
		render func(*Graph) string
		want   string
	}{
		{name: "mermaid", render: (*Graph).Mermaid, want: `subgraph n1_group["a/b"]`},
		{name: "dot", render: (*Graph).Dot, want: `subgraph "cluster_a/b"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.render(ProjectGraph(project))
			if !strings.Contains(got, tt.want) {
				t.Errorf("output does not contain %q:\n%s", tt.want, got)
			}
		})
	}
}
