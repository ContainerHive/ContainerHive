package dependency

import (
	"path/filepath"
	"testing"

	"github.com/timo-reymann/ContainerHive/pkg/discovery"
	"github.com/timo-reymann/ContainerHive/pkg/rendering"
)

func mustAbs(t *testing.T, path string) string {
	t.Helper()
	abs, err := filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}
	return abs
}

func TestIntegration_DependencyProject(t *testing.T) {
	projectPath := mustAbs(t, "../../pkg/testdata/dependency-project")

	project, err := discovery.DiscoverProject(t.Context(), projectPath)
	if err != nil {
		t.Fatalf("discovery failed: %v", err)
	}

	distPath := filepath.Join(t.TempDir(), "dist")
	if err := rendering.RenderProject(t.Context(), project, distPath); err != nil {
		t.Fatalf("rendering failed: %v", err)
	}

	scannedGraph, err := ScanRenderedProject(distPath)
	if err != nil {
		t.Fatalf("scanning failed: %v", err)
	}

	graph, err := BuildDependencyGraph(scannedGraph, project)
	if err != nil {
		t.Fatalf("graph building failed: %v", err)
	}

	if !graph.HasDependencies() {
		t.Error("expected dependencies")
	}

	order, err := graph.TopologicalSort()
	if err != nil {
		t.Fatalf("topological sort failed: %v", err)
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
		t.Errorf("ubuntu (idx=%d) must come before python (idx=%d)", indexOf("ubuntu"), indexOf("python"))
	}
}
