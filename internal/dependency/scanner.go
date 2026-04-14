package dependency

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/timo-reymann/ContainerHive/pkg/model"
)

// HivePrefix is the Dockerfile image reference prefix used to denote project-local hive dependencies.
const HivePrefix = "__hive__/"

var hiveFromPattern = regexp.MustCompile(`(?i)^FROM\s+__hive__/([^:\s]+):([^\s]+)`)

// HiveRef represents a reference to a project-local image via the __hive__/ prefix.
type HiveRef struct {
	ImageName string
	Tag       string
}

// ScanDockerfileForHiveRefs scans a Dockerfile for FROM __hive__/<name>:<tag> references.
func ScanDockerfileForHiveRefs(dockerfilePath string) ([]HiveRef, error) {
	content, err := os.ReadFile(dockerfilePath)
	if err != nil {
		return nil, err
	}

	var refs []HiveRef
	for _, line := range strings.Split(string(content), "\n") {
		line = strings.TrimSpace(line)
		matches := hiveFromPattern.FindStringSubmatch(line)
		if matches != nil {
			refs = append(refs, HiveRef{
				ImageName: matches[1],
				Tag:       matches[2],
			})
		}
	}
	return refs, nil
}

// ScanProjectSource scans source Dockerfiles from the project model for
// __hive__/ references and builds a dependency graph. This is used by CI
// template generation where the rendered dist directory may not exist yet.
func ScanProjectSource(project *model.ContainerHiveProject) *Graph {
	graph := NewGraph()

	for name := range project.ImagesByName {
		graph.AddImage(name)
	}

	for name, images := range project.ImagesByName {
		for _, img := range images {
			scanEntryPoint(graph, name, img.BuildEntryPointPath)
			for _, variant := range img.Variants {
				scanEntryPoint(graph, name, variant.BuildEntryPointPath)
			}
		}
	}

	return graph
}

func scanEntryPoint(graph *Graph, imageName, entryPointPath string) {
	if entryPointPath == "" {
		return
	}
	refs, err := ScanDockerfileForHiveRefs(entryPointPath)
	if err != nil {
		return
	}
	for _, ref := range refs {
		if ref.ImageName != imageName {
			graph.AddDependency(imageName, ref.ImageName)
		}
	}
}

// ScanRenderedProject scans all Dockerfiles in a rendered dist directory
// and builds a dependency graph based on __hive__/ references.
func ScanRenderedProject(distPath string) (*Graph, error) {
	graph := NewGraph()

	entries, err := os.ReadDir(distPath)
	if err != nil {
		return nil, errors.Join(errors.New("failed to read dist directory"), err)
	}

	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		graph.AddImage(entry.Name())
	}

	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		imageName := entry.Name()
		imageDir := filepath.Join(distPath, imageName)

		tagEntries, err := os.ReadDir(imageDir)
		if err != nil {
			return nil, errors.Join(errors.New("failed to read image directory "+imageName), err)
		}

		for _, tagEntry := range tagEntries {
			if !tagEntry.IsDir() {
				continue
			}
			tagDir := filepath.Join(imageDir, tagEntry.Name())

			for _, dfName := range []string{"Dockerfile", "Dockerfile.gotpl"} {
				dfPath := filepath.Join(tagDir, dfName)
				if _, statErr := os.Stat(dfPath); statErr != nil {
					continue
				}

				refs, err := ScanDockerfileForHiveRefs(dfPath)
				if err != nil {
					return nil, errors.Join(errors.New("failed to scan "+dfPath), err)
				}

				for _, ref := range refs {
					if ref.ImageName != imageName {
						graph.AddDependency(imageName, ref.ImageName)
					}
				}
			}
		}
	}

	return graph, nil
}
