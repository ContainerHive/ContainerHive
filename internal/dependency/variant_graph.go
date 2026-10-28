package dependency

import (
	"os"
	"regexp"
	"strings"

	"github.com/ContainerHive/ContainerHive/pkg/model"
)

var hiveParentFromPattern = regexp.MustCompile(`(?im)^\s*FROM\s+(?:--\S+\s+)*__hive_parent__(\s|$)`)

// VariantNodeID returns the graph node name of an image variant. The base
// image keeps its plain name.
func VariantNodeID(imageName, variantName string) string {
	return imageName + "/" + variantName
}

// ScanProjectVariants builds a dependency graph on image and variant level
// from the project source. The base image of every image name is a node, and
// so is each of its variants as "<image>/<variant>". Tags are not nodes: a
// __hive__/<name>:<tag> reference points at the variant whose tag suffix the
// tag ends with, or at the base image when none matches.
func ScanProjectVariants(project *model.ContainerHiveProject) *Graph {
	graph := NewGraph()

	for name, images := range project.ImagesByName {
		graph.AddImage(name)
		for _, img := range images {
			for variantName := range img.Variants {
				graph.AddImage(VariantNodeID(name, variantName))
			}
		}
	}

	for name, images := range project.ImagesByName {
		for _, img := range images {
			scanVariantNode(graph, project, name, name, "", img.BuildEntryPointPath, imageParentRef(img))
			for _, dep := range img.DependsOn {
				addVariantEdge(graph, name, dep)
			}

			for variantName, variant := range img.Variants {
				node := VariantNodeID(name, variantName)
				scanVariantNode(graph, project, name, node, name, variant.BuildEntryPointPath, variantParentRef(img, variant))
				// depends_on is image level config and applies to every variant.
				for _, dep := range img.DependsOn {
					addVariantEdge(graph, node, dep)
				}
			}
		}
	}

	return graph
}

// scanVariantNode adds the edges of one node: its Dockerfile hive references
// and its backend parent. baseNode is the base image of a variant node and
// empty for a base image: only a variant can build on __hive_parent__, which
// stands for its base image.
func scanVariantNode(graph *Graph, project *model.ContainerHiveProject, imageName, node, baseNode, entryPointPath, parent string) {
	if entryPointPath != "" {
		refs, err := ScanDockerfileForHiveRefs(entryPointPath)
		if err == nil {
			for _, ref := range refs {
				addTaggedEdge(graph, project, imageName, node, ref.ImageName, ref.Tag)
			}
		}
		if baseNode != "" && usesHiveParent(entryPointPath) {
			addVariantEdge(graph, node, baseNode)
		}
	}

	if baseNode != "" && parent == model.HiveParentPlaceholder {
		addVariantEdge(graph, node, baseNode)
	}

	if parent == "" || parent == model.HiveParentPlaceholder || !strings.HasPrefix(parent, HivePrefix) {
		return
	}
	rest := strings.TrimPrefix(parent, HivePrefix)
	idx := strings.LastIndex(rest, ":")
	if idx < 1 {
		return
	}
	addTaggedEdge(graph, project, imageName, node, rest[:idx], rest[idx+1:])
}

// addTaggedEdge records that node depends on the variant of targetImage the
// tag selects. References to unknown images and to the image itself are
// skipped; the build reports missing parents.
func addTaggedEdge(graph *Graph, project *model.ContainerHiveProject, imageName, node, targetImage, tag string) {
	if targetImage == imageName {
		return
	}
	images, ok := project.ImagesByName[targetImage]
	if !ok {
		return
	}
	addVariantEdge(graph, node, resolveVariantTarget(targetImage, images, tag))
}

// addVariantEdge records an edge when both ends are known nodes and differ.
func addVariantEdge(graph *Graph, from, to string) {
	if from == to || !graph.nodes[to] {
		return
	}
	graph.AddDependency(from, to)
}

// resolveVariantTarget maps a tag of an image to the node it is built by:
// the variant with the longest non-empty tag suffix the tag ends with, or the
// base image.
func resolveVariantTarget(imageName string, images []*model.Image, tag string) string {
	target, longest := imageName, 0
	for _, img := range images {
		for variantName, variant := range img.Variants {
			suffix := variant.TagSuffix
			if len(suffix) <= longest || !strings.HasSuffix(tag, suffix) {
				continue
			}
			target, longest = VariantNodeID(imageName, variantName), len(suffix)
		}
	}
	return target
}

// usesHiveParent reports whether the Dockerfile builds FROM __hive_parent__.
func usesHiveParent(dockerfilePath string) bool {
	content, err := os.ReadFile(dockerfilePath)
	if err != nil {
		return false
	}
	return hiveParentFromPattern.Match(content)
}
