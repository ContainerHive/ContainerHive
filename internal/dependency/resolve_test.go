package dependency

import (
	"slices"
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

func TestBuildDependencyGraph_NixParent(t *testing.T) {
	t.Run("nix parent hive ref adds the edge without depends_on", func(t *testing.T) {
		scannedGraph := NewGraph()
		scannedGraph.AddImage("base")
		scannedGraph.AddImage("child")

		project := &model.ContainerHiveProject{
			ImagesByName: map[string][]*model.Image{
				"base":  {{Name: "base"}},
				"child": {{Name: "child", Nix: model.NixConfig{Parent: "__hive__/base:1.0"}}},
			},
		}

		order, err := BuildDependencyGraph(scannedGraph, project)
		if err != nil {
			t.Fatalf("BuildDependencyGraph failed: %v", err)
		}
		sorted, err := order.TopologicalSort()
		if err != nil {
			t.Fatalf("TopologicalSort failed: %v", err)
		}
		if slices.Index(sorted, "base") > slices.Index(sorted, "child") {
			t.Errorf("order = %v, base must build before child", sorted)
		}
	})

	t.Run("unknown nix parent fails", func(t *testing.T) {
		scannedGraph := NewGraph()
		project := &model.ContainerHiveProject{
			ImagesByName: map[string][]*model.Image{
				"child": {{Name: "child", Nix: model.NixConfig{Parent: "__hive__/missing:1.0"}}},
			},
		}
		_, err := BuildDependencyGraph(scannedGraph, project)
		if err == nil {
			t.Fatal("expected error for unknown nix parent, got nil")
		}
	})

	t.Run("plain parent refs and the placeholder carry no edge", func(t *testing.T) {
		scannedGraph := NewGraph()
		project := &model.ContainerHiveProject{
			ImagesByName: map[string][]*model.Image{
				"child": {{Name: "child", Nix: model.NixConfig{Parent: "ubuntu:22.04"}}},
			},
		}
		if _, err := BuildDependencyGraph(scannedGraph, project); err != nil {
			t.Fatalf("BuildDependencyGraph failed: %v", err)
		}

		scannedGraph2 := NewGraph()
		project2 := &model.ContainerHiveProject{
			ImagesByName: map[string][]*model.Image{
				"app": {{Name: "app", Variants: map[string]*model.ImageVariant{
					"dev": {Name: "dev", Nix: &model.NixConfig{Parent: model.HiveParentPlaceholder}},
				}}},
			},
		}
		if _, err := BuildDependencyGraph(scannedGraph2, project2); err != nil {
			t.Fatalf("BuildDependencyGraph failed: %v", err)
		}
	})
}
