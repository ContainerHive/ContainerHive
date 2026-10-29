package deps

import (
	"fmt"
	"slices"
	"strings"

	"github.com/ContainerHive/ContainerHive/internal/dependency"
	"github.com/ContainerHive/ContainerHive/pkg/model"
)

// Graph is the dependency graph of a project on image and variant level.
// Nodes are named "<image>" for a base image and "<image>/<variant>" for a
// variant. Tags are not part of the graph.
type Graph struct {
	nodes []string
	edges map[string][]string
	// nodeImages maps a node to its image name. Image names can contain a
	// slash, so the name cannot always be cut from the node ID; nodes missing
	// here fall back to the part before the first slash.
	nodeImages map[string]string
}

// ProjectGraph scans the project source and returns its image and variant
// dependency graph. It does not need a rendered dist directory.
func ProjectGraph(project *model.ContainerHiveProject) *Graph {
	graph := dependency.ScanProjectVariants(project)

	nodeImages := make(map[string]string)
	for imageName, images := range project.ImagesByName {
		nodeImages[imageName] = imageName
		for _, img := range images {
			for variantName := range img.Variants {
				nodeImages[dependency.VariantNodeID(imageName, variantName)] = imageName
			}
		}
	}

	return &Graph{
		nodes:      graph.Nodes(),
		edges:      graph.Edges(),
		nodeImages: nodeImages,
	}
}

// Nodes returns all node names in sorted order.
func (g *Graph) Nodes() []string {
	return slices.Clone(g.nodes)
}

// Edges returns the dependency edges: each key depends on every listed
// value. The result is a copy.
func (g *Graph) Edges() map[string][]string {
	edges := make(map[string][]string, len(g.edges))
	for from, deps := range g.edges {
		edges[from] = slices.Clone(deps)
	}
	return edges
}

// imageOf returns the image name a node belongs to.
func (g *Graph) imageOf(node string) string {
	if image, ok := g.nodeImages[node]; ok {
		return image
	}
	image, _, _ := strings.Cut(node, "/")
	return image
}

// groups maps each image name to its node names, base image first.
func (g *Graph) groups() (images []string, members map[string][]string) {
	members = make(map[string][]string)
	for _, node := range g.nodes {
		image := g.imageOf(node)
		if _, ok := members[image]; !ok {
			images = append(images, image)
		}
		members[image] = append(members[image], node)
	}
	return images, members
}

// sortedSources returns the edge sources in sorted order.
func (g *Graph) sortedSources() []string {
	sources := make([]string, 0, len(g.edges))
	for from := range g.edges {
		sources = append(sources, from)
	}
	slices.Sort(sources)
	return sources
}

// Mermaid renders the graph as a Mermaid flowchart. An arrow points from a
// node to a node it depends on, and the variants of an image share a
// subgraph.
func (g *Graph) Mermaid() string {
	var b strings.Builder
	b.WriteString("graph LR\n")

	ids := make(map[string]string, len(g.nodes))
	for i, node := range g.nodes {
		ids[node] = fmt.Sprintf("n%d", i)
	}

	images, members := g.groups()
	for _, image := range images {
		nodes := members[image]
		if len(nodes) == 1 {
			fmt.Fprintf(&b, "  %s[%q]\n", ids[nodes[0]], nodes[0])
			continue
		}
		fmt.Fprintf(&b, "  subgraph %s_group[%q]\n", ids[nodes[0]], image)
		for _, node := range nodes {
			fmt.Fprintf(&b, "    %s[%q]\n", ids[node], node)
		}
		b.WriteString("  end\n")
	}

	for _, from := range g.sortedSources() {
		for _, to := range g.edges[from] {
			fmt.Fprintf(&b, "  %s --> %s\n", ids[from], ids[to])
		}
	}
	return b.String()
}

// Dot renders the graph in Graphviz DOT format. An edge points from a node to
// a node it depends on, and the variants of an image share a cluster.
func (g *Graph) Dot() string {
	var b strings.Builder
	b.WriteString("digraph containerhive {\n")
	b.WriteString("  rankdir=LR;\n")

	images, members := g.groups()
	for _, image := range images {
		nodes := members[image]
		if len(nodes) == 1 {
			fmt.Fprintf(&b, "  %q;\n", nodes[0])
			continue
		}
		fmt.Fprintf(&b, "  subgraph %q {\n", "cluster_"+image)
		fmt.Fprintf(&b, "    label=%q;\n", image)
		for _, node := range nodes {
			fmt.Fprintf(&b, "    %q;\n", node)
		}
		b.WriteString("  }\n")
	}

	for _, from := range g.sortedSources() {
		for _, to := range g.edges[from] {
			fmt.Fprintf(&b, "  %q -> %q;\n", from, to)
		}
	}
	b.WriteString("}\n")
	return b.String()
}
