package dependency

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/ContainerHive/ContainerHive/pkg/model"
)

func writeDockerfile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write %s: %v", name, err)
	}
	return path
}

func TestScanProjectVariants(t *testing.T) {
	dir := t.TempDir()
	baseDF := writeDockerfile(t, dir, "Dockerfile.base", "FROM ubuntu:22.04")
	plainDF := writeDockerfile(t, dir, "Dockerfile.plain", "FROM __hive__/base:1.0")
	slimDF := writeDockerfile(t, dir, "Dockerfile.slim", "FROM __hive__/base:1.0-slim")
	selfDF := writeDockerfile(t, dir, "Dockerfile.self", "FROM __hive__/app:1.0\nFROM __hive__/missing:1.0")

	project := &model.ContainerHiveProject{
		ImagesByName: map[string][]*model.Image{
			"base": {{
				Name:                "base",
				BuildEntryPointPath: baseDF,
				Variants: map[string]*model.ImageVariant{
					"slim": {TagSuffix: "-slim"},
				},
			}},
			"app": {{
				Name:                "app",
				BuildEntryPointPath: plainDF,
				Variants: map[string]*model.ImageVariant{
					"slim": {BuildEntryPointPath: slimDF, TagSuffix: "-slim"},
					"self": {BuildEntryPointPath: selfDF},
				},
			}},
		},
	}

	graph := ScanProjectVariants(project)

	wantNodes := []string{"app", "app/self", "app/slim", "base", "base/slim"}
	if got := graph.Nodes(); !reflect.DeepEqual(got, wantNodes) {
		t.Errorf("Nodes() = %v, want %v", got, wantNodes)
	}

	wantEdges := map[string][]string{
		"app":      {"base"},
		"app/slim": {"base/slim"},
	}
	if got := graph.Edges(); !reflect.DeepEqual(got, wantEdges) {
		t.Errorf("Edges() = %v, want %v", got, wantEdges)
	}
}

func TestScanProjectVariantsDependsOn(t *testing.T) {
	project := &model.ContainerHiveProject{
		ImagesByName: map[string][]*model.Image{
			"base": {{Name: "base"}},
			"app": {{
				Name:      "app",
				DependsOn: []string{"base", "unknown"},
				Variants: map[string]*model.ImageVariant{
					"slim": {},
				},
			}},
		},
	}

	graph := ScanProjectVariants(project)

	wantEdges := map[string][]string{
		"app":      {"base"},
		"app/slim": {"base"},
	}
	if got := graph.Edges(); !reflect.DeepEqual(got, wantEdges) {
		t.Errorf("Edges() = %v, want %v", got, wantEdges)
	}
}

func TestScanProjectVariantsBackendParent(t *testing.T) {
	tests := []struct {
		name  string
		image *model.Image
		want  map[string][]string
	}{
		{
			name: "variant parent selects variant by tag suffix",
			image: &model.Image{
				Name:         "app",
				BuildBackend: model.BuildBackendNix,
				Nix:          model.NixConfig{Parent: "__hive__/base:1.0"},
				Variants: map[string]*model.ImageVariant{
					"slim":    {Nix: &model.NixConfig{Parent: "__hive__/base:1.0-slim"}},
					"inherit": {},
				},
			},
			want: map[string][]string{
				"app":         {"base"},
				"app/inherit": {"base"},
				"app/slim":    {"base/slim"},
			},
		},
		{
			name: "placeholder and plain image parents add no edge",
			image: &model.Image{
				Name:         "app",
				BuildBackend: model.BuildBackendNix,
				Nix:          model.NixConfig{Parent: model.HiveParentPlaceholder},
				Variants: map[string]*model.ImageVariant{
					"slim": {Nix: &model.NixConfig{Parent: "ubuntu:22.04"}},
				},
			},
			want: map[string][]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			project := &model.ContainerHiveProject{
				ImagesByName: map[string][]*model.Image{
					"base": {{
						Name: "base",
						Variants: map[string]*model.ImageVariant{
							"slim": {TagSuffix: "-slim"},
						},
					}},
					"app": {tt.image},
				},
			}

			got := ScanProjectVariants(project).Edges()
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Edges() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResolveVariantTarget(t *testing.T) {
	images := []*model.Image{{
		Variants: map[string]*model.ImageVariant{
			"slim":     {TagSuffix: "-slim"},
			"fat-slim": {TagSuffix: "-fat-slim"},
			"plain":    {},
		},
	}}

	tests := []struct {
		tag  string
		want string
	}{
		{tag: "1.0", want: "base"},
		{tag: "1.0-slim", want: "base/slim"},
		{tag: "1.0-fat-slim", want: "base/fat-slim"},
	}

	for _, tt := range tests {
		t.Run(tt.tag, func(t *testing.T) {
			if got := resolveVariantTarget("base", images, tt.tag); got != tt.want {
				t.Errorf("resolveVariantTarget(%q) = %q, want %q", tt.tag, got, tt.want)
			}
		})
	}
}
