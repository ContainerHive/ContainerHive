package dependency

import (
	"reflect"
	"testing"

	"github.com/ContainerHive/ContainerHive/pkg/model"
)

func TestScanProjectVariantsHiveParent(t *testing.T) {
	dir := t.TempDir()
	parentDF := writeDockerfile(t, dir, "Dockerfile.parent", "FROM __hive_parent__\nRUN true")
	baseDF := writeDockerfile(t, dir, "Dockerfile.base", "FROM __hive_parent__")
	otherDF := writeDockerfile(t, dir, "Dockerfile.other", "FROM ubuntu:22.04\n# FROM __hive_parent__")
	platformDF := writeDockerfile(t, dir, "Dockerfile.platform", "FROM --platform=$BUILDPLATFORM __hive_parent__ AS build")

	project := &model.ContainerHiveProject{
		ImagesByName: map[string][]*model.Image{
			"app": {{
				Name:                "app",
				BuildEntryPointPath: baseDF,
				Variants: map[string]*model.ImageVariant{
					"node":     {BuildEntryPointPath: parentDF},
					"other":    {BuildEntryPointPath: otherDF},
					"platform": {BuildEntryPointPath: platformDF},
					"nix": {
						BuildBackend: model.BuildBackendNix,
						Nix:          &model.NixConfig{Parent: model.HiveParentPlaceholder},
					},
				},
			}},
		},
	}

	// The base image ignores the placeholder, a commented one does not count.
	wantEdges := map[string][]string{
		"app/node":     {"app"},
		"app/nix":      {"app"},
		"app/platform": {"app"},
	}
	if got := ScanProjectVariants(project).Edges(); !reflect.DeepEqual(got, wantEdges) {
		t.Errorf("Edges() = %v, want %v", got, wantEdges)
	}
}
