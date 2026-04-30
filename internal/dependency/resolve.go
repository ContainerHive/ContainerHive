package dependency

import (
	"fmt"

	"github.com/ContainerHive/ContainerHive/pkg/model"
)

// BuildDependencyGraph merges a scanned dependency graph (from Dockerfile analysis)
// with explicit depends_on declarations from image configs.
func BuildDependencyGraph(scannedGraph *Graph, project *model.ContainerHiveProject) (*Graph, error) {
	graph := NewGraph()
	for node := range scannedGraph.nodes {
		graph.AddImage(node)
	}
	for from, deps := range scannedGraph.edges {
		for _, dep := range deps {
			graph.AddDependency(from, dep)
		}
	}

	for name, images := range project.ImagesByName {
		graph.AddImage(name)
		for _, img := range images {
			for _, dep := range img.DependsOn {
				if _, exists := project.ImagesByName[dep]; !exists {
					return nil, fmt.Errorf("image %q declares depends_on %q, but no image with that name exists in the project", name, dep)
				}
				graph.AddDependency(name, dep)
			}
		}
	}

	return graph, nil
}
