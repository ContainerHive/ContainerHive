package deps

import (
	"github.com/timo-reymann/ContainerHive/internal/dependency"
	"github.com/timo-reymann/ContainerHive/pkg/model"
)

// BuildOrder wraps a resolved dependency graph, providing build-order
// iteration and dependency queries without exposing internal types.
type BuildOrder struct {
	graph *dependency.Graph
	order []string
}

// HasDependencies returns true if any inter-image dependencies exist.
func (b *BuildOrder) HasDependencies() bool {
	return b.graph.HasDependencies()
}

// Dependents returns the list of images that depend on the given image.
func (b *BuildOrder) Dependents(name string) []string {
	return b.graph.Dependents(name)
}

// Order returns the topologically sorted build order (dependencies first).
func (b *BuildOrder) Order() []string {
	return b.order
}

// ResolveOrder scans a rendered dist directory for __hive__/ references,
// merges with explicit depends_on declarations, and returns a topologically
// sorted build order.
func ResolveOrder(distPath string, project *model.ContainerHiveProject) (*BuildOrder, error) {
	scannedGraph, err := dependency.ScanRenderedProject(distPath)
	if err != nil {
		return nil, err
	}

	graph, err := dependency.BuildDependencyGraph(scannedGraph, project)
	if err != nil {
		return nil, err
	}

	order, err := graph.TopologicalSort()
	if err != nil {
		return nil, err
	}

	return &BuildOrder{
		graph: graph,
		order: order,
	}, nil
}
