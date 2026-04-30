package dependency

import (
	"testing"

	"github.com/ContainerHive/ContainerHive/pkg/model"
)

func TestBuildDependencyGraph(t *testing.T) {
	t.Run("merges auto-detected and explicit dependencies", func(t *testing.T) {
		scannedGraph := NewGraph()
		scannedGraph.AddImage("ubuntu")
		scannedGraph.AddImage("python")
		scannedGraph.AddImage("app")
		scannedGraph.AddDependency("python", "ubuntu")

		project := &model.ContainerHiveProject{
			ImagesByName: map[string][]*model.Image{
				"ubuntu": {{Name: "ubuntu"}},
				"python": {{Name: "python", DependsOn: []string{"ubuntu"}}},
				"app":    {{Name: "app", DependsOn: []string{"python"}}},
			},
		}

		graph, err := BuildDependencyGraph(scannedGraph, project)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		order, err := graph.TopologicalSort()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		indexOf := func(name string) int {
			for i, v := range order {
				if v == name {
					return i
				}
			}
			return -1
		}

		if indexOf("ubuntu") > indexOf("python") {
			t.Error("ubuntu must come before python")
		}
		if indexOf("python") > indexOf("app") {
			t.Error("python must come before app")
		}
	})

	t.Run("errors on unknown depends_on target", func(t *testing.T) {
		scannedGraph := NewGraph()
		scannedGraph.AddImage("app")

		project := &model.ContainerHiveProject{
			ImagesByName: map[string][]*model.Image{
				"app": {{Name: "app", DependsOn: []string{"nonexistent"}}},
			},
		}

		_, err := BuildDependencyGraph(scannedGraph, project)
		if err == nil {
			t.Fatal("expected error for unknown dependency, got nil")
		}
	})
}
