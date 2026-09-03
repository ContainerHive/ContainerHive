package dependency

import (
	"errors"
	"fmt"
	"slices"
)

// Graph represents a dependency graph of container images.
// Edges encode "from depends on to", meaning "to" must be built before "from".
type Graph struct {
	nodes map[string]bool
	edges map[string][]string
}

// NewGraph creates an empty dependency graph.
func NewGraph() *Graph {
	return &Graph{
		nodes: make(map[string]bool),
		edges: make(map[string][]string),
	}
}

// AddImage registers an image name as a node in the graph.
func (g *Graph) AddImage(name string) {
	g.nodes[name] = true
}

// AddDependency records that "from" depends on "to",
// meaning "to" must be built before "from".
// Deduped: multi-variant images can call this with the same pair more than once.
func (g *Graph) AddDependency(from, to string) {
	if slices.Contains(g.edges[from], to) {
		return
	}
	g.edges[from] = append(g.edges[from], to)
}

// Dependencies returns the list of images that the given image depends on.
func (g *Graph) Dependencies(name string) []string {
	return g.edges[name]
}

// Dependents returns the list of images that depend on the given image.
func (g *Graph) Dependents(name string) []string {
	var result []string
	for from, deps := range g.edges {
		for _, to := range deps {
			if to == name {
				result = append(result, from)
				break
			}
		}
	}
	return result
}

// HasDependencies returns true if any dependency edges exist in the graph.
func (g *Graph) HasDependencies() bool {
	for _, deps := range g.edges {
		if len(deps) > 0 {
			return true
		}
	}
	return false
}

// TopologicalSort returns a build order where dependencies come first.
// It uses Kahn's algorithm and returns an error if a cycle is detected.
func (g *Graph) TopologicalSort() ([]string, error) {
	// Build a reverse adjacency list: for each node, which nodes depend on it.
	// Also compute in-degree (number of dependencies each node has).
	dependents := make(map[string][]string) // dependents[to] = list of nodes that depend on "to"
	inDegree := make(map[string]int)

	for node := range g.nodes {
		inDegree[node] = 0
	}

	for from, deps := range g.edges {
		inDegree[from] += len(deps)
		for _, to := range deps {
			dependents[to] = append(dependents[to], from)
		}
	}

	// Initialize queue with nodes that have no dependencies (in-degree 0).
	var queue []string
	for node := range g.nodes {
		if inDegree[node] == 0 {
			queue = append(queue, node)
		}
	}
	slices.Sort(queue)

	var order []string
	for len(queue) > 0 {
		// Take the first element (alphabetically smallest for determinism).
		node := queue[0]
		queue = queue[1:]
		order = append(order, node)

		// For each node that depends on the current node, reduce its in-degree.
		for _, dependent := range dependents[node] {
			inDegree[dependent]--
			if inDegree[dependent] == 0 {
				queue = append(queue, dependent)
				slices.Sort(queue)
			}
		}
	}

	if len(order) != len(g.nodes) {
		return nil, errors.Join(
			errors.New("dependency cycle detected"),
			fmt.Errorf("resolved %d of %d images", len(order), len(g.nodes)),
		)
	}

	return order, nil
}
