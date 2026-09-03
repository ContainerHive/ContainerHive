package dependency

import (
	"testing"
)

func TestGraph_TopologicalSort(t *testing.T) {
	t.Run("linear chain A->B->C", func(t *testing.T) {
		g := NewGraph()
		g.AddImage("A")
		g.AddImage("B")
		g.AddImage("C")
		g.AddDependency("C", "B")
		g.AddDependency("B", "A")

		order, err := g.TopologicalSort()
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

		if indexOf("A") > indexOf("B") {
			t.Error("A must come before B")
		}
		if indexOf("B") > indexOf("C") {
			t.Error("B must come before C")
		}
	})

	t.Run("diamond dependency", func(t *testing.T) {
		g := NewGraph()
		g.AddImage("base")
		g.AddImage("left")
		g.AddImage("right")
		g.AddImage("top")
		g.AddDependency("left", "base")
		g.AddDependency("right", "base")
		g.AddDependency("top", "left")
		g.AddDependency("top", "right")

		order, err := g.TopologicalSort()
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

		if indexOf("base") > indexOf("left") || indexOf("base") > indexOf("right") {
			t.Error("base must come before left and right")
		}
		if indexOf("left") > indexOf("top") || indexOf("right") > indexOf("top") {
			t.Error("left and right must come before top")
		}
	})

	t.Run("no dependencies returns all images", func(t *testing.T) {
		g := NewGraph()
		g.AddImage("A")
		g.AddImage("B")

		order, err := g.TopologicalSort()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(order) != 2 {
			t.Errorf("expected 2 images, got %d", len(order))
		}
	})

	t.Run("detects simple cycle", func(t *testing.T) {
		g := NewGraph()
		g.AddImage("A")
		g.AddImage("B")
		g.AddDependency("A", "B")
		g.AddDependency("B", "A")

		_, err := g.TopologicalSort()
		if err == nil {
			t.Fatal("expected cycle error, got nil")
		}
	})

	t.Run("detects transitive cycle", func(t *testing.T) {
		g := NewGraph()
		g.AddImage("A")
		g.AddImage("B")
		g.AddImage("C")
		g.AddDependency("A", "B")
		g.AddDependency("B", "C")
		g.AddDependency("C", "A")

		_, err := g.TopologicalSort()
		if err == nil {
			t.Fatal("expected cycle error, got nil")
		}
	})
}

func TestGraph_Dependents(t *testing.T) {
	t.Run("returns images that depend on given image", func(t *testing.T) {
		g := NewGraph()
		g.AddImage("ubuntu")
		g.AddImage("dotnet")
		g.AddImage("python")
		g.AddDependency("dotnet", "ubuntu")
		g.AddDependency("python", "ubuntu")

		deps := g.Dependents("ubuntu")
		if len(deps) != 2 {
			t.Fatalf("expected 2 dependents, got %d", len(deps))
		}
	})

	t.Run("returns empty for leaf image", func(t *testing.T) {
		g := NewGraph()
		g.AddImage("ubuntu")
		g.AddImage("dotnet")
		g.AddDependency("dotnet", "ubuntu")

		deps := g.Dependents("dotnet")
		if len(deps) != 0 {
			t.Fatalf("expected 0 dependents, got %d", len(deps))
		}
	})

	t.Run("returns empty for image with no edges", func(t *testing.T) {
		g := NewGraph()
		g.AddImage("standalone")

		deps := g.Dependents("standalone")
		if len(deps) != 0 {
			t.Fatalf("expected 0 dependents, got %d", len(deps))
		}
	})
}

func TestGraph_HasDependencies(t *testing.T) {
	t.Run("returns false when no edges", func(t *testing.T) {
		g := NewGraph()
		g.AddImage("A")
		if g.HasDependencies() {
			t.Error("expected false")
		}
	})

	t.Run("returns true when edges exist", func(t *testing.T) {
		g := NewGraph()
		g.AddImage("A")
		g.AddImage("B")
		g.AddDependency("B", "A")
		if !g.HasDependencies() {
			t.Error("expected true")
		}
	})
}

func TestGraph_AddDependency_Dedup(t *testing.T) {
	g := NewGraph()
	g.AddImage("A")
	g.AddImage("B")

	g.AddDependency("B", "A")
	g.AddDependency("B", "A")

	deps := g.Dependencies("B")
	if len(deps) != 1 {
		t.Fatalf("expected 1 dependency, got %d: %v", len(deps), deps)
	}
}
