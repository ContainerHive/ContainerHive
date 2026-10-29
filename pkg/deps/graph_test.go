package deps

import (
	"testing"

	"github.com/ContainerHive/ContainerHive/pkg/model"
)

func testGraph() *Graph {
	return &Graph{
		nodes: []string{"app", "app/slim", "base"},
		edges: map[string][]string{
			"app":      {"base"},
			"app/slim": {"base", "app"},
		},
	}
}

func TestGraphMermaid(t *testing.T) {
	want := `graph LR
  subgraph n0_group["app"]
    n0["app"]
    n1["app/slim"]
  end
  n2["base"]
  n0 --> n2
  n1 --> n2
  n1 --> n0
`
	if got := testGraph().Mermaid(); got != want {
		t.Errorf("Mermaid() =\n%s\nwant\n%s", got, want)
	}
}

func TestGraphDot(t *testing.T) {
	want := `digraph containerhive {
  rankdir=LR;
  subgraph "cluster_app" {
    label="app";
    "app";
    "app/slim";
  }
  "base";
  "app" -> "base";
  "app/slim" -> "base";
  "app/slim" -> "app";
}
`
	if got := testGraph().Dot(); got != want {
		t.Errorf("Dot() =\n%s\nwant\n%s", got, want)
	}
}

func TestProjectGraph(t *testing.T) {
	project := &model.ContainerHiveProject{
		ImagesByName: map[string][]*model.Image{
			"base": {{Name: "base"}},
			"app": {{
				Name:      "app",
				DependsOn: []string{"base"},
				Variants:  map[string]*model.ImageVariant{"slim": {}},
			}},
		},
	}

	graph := ProjectGraph(project)

	if got := len(graph.Nodes()); got != 3 {
		t.Errorf("len(Nodes()) = %d, want 3", got)
	}
	if got := graph.Edges()["app/slim"]; len(got) != 1 || got[0] != "base" {
		t.Errorf("Edges()[app/slim] = %v, want [base]", got)
	}
}
