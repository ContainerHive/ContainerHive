package dependency

import (
	"fmt"
	"strings"

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
			for _, parent := range nixParentRefs(img) {
				if err := addNixParentDependency(graph, project, name, parent); err != nil {
					return nil, err
				}
			}
		}
	}

	return graph, nil
}

// addNixParentDependency records the implicit build order edge from a
// nix.parent hive reference, so a flake build with a parent depends on it
// without an explicit depends_on. Plain image references and the parent
// placeholder carry no build order edge.
func addNixParentDependency(graph *Graph, project *model.ContainerHiveProject, from, parent string) error {
	if parent == "" || parent == model.HiveParentPlaceholder {
		return nil
	}
	if !strings.HasPrefix(parent, HivePrefix) {
		return nil
	}
	name, tag := parent, ""
	if rest := strings.TrimPrefix(parent, HivePrefix); rest != "" {
		if idx := strings.LastIndex(rest, ":"); idx > -1 {
			name, tag = rest[:idx], rest[idx+1:]
		}
	}
	if name == "" || tag == "" {
		return fmt.Errorf("image %q sets nix.parent %q, want %s<name>:<tag>", from, parent, HivePrefix)
	}
	if _, exists := project.ImagesByName[name]; !exists {
		return fmt.Errorf("image %q sets nix.parent %q, but no image with that name exists in the project", from, parent)
	}
	if name != from {
		graph.AddDependency(from, name)
	}
	return nil
}
