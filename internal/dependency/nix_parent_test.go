package dependency

import (
	"slices"
	"testing"

	"github.com/ContainerHive/ContainerHive/pkg/model"
)

func TestNixParentEdges_effectiveBackend(t *testing.T) {
	const ref = "__hive__/base:1.0"
	nix := model.BuildBackendNix
	dockerfile := model.BuildBackendDockerfile

	tests := []struct {
		name     string
		child    *model.Image
		wantEdge bool
		wantErr  bool
	}{
		{
			name:  "dockerfile image ignores a stray nix parent",
			child: &model.Image{Name: "child", BuildBackend: dockerfile, Nix: model.NixConfig{Parent: "__hive__/missing:1.0"}},
		},
		{
			name:     "nix image uses its parent",
			child:    &model.Image{Name: "child", BuildBackend: nix, Nix: model.NixConfig{Parent: ref}},
			wantEdge: true,
		},
		{
			name: "dockerfile image with nix variant inheriting the image parent",
			child: &model.Image{Name: "child", BuildBackend: dockerfile, Nix: model.NixConfig{Parent: ref},
				Variants: map[string]*model.ImageVariant{"v": {Name: "v", BuildBackend: nix}}},
			wantEdge: true,
		},
		{
			name: "nix variant with own parent",
			child: &model.Image{Name: "child", BuildBackend: dockerfile,
				Variants: map[string]*model.ImageVariant{"v": {Name: "v", BuildBackend: nix, Nix: &model.NixConfig{Parent: ref}}}},
			wantEdge: true,
		},
		{
			name: "dockerfile variant of a nix image ignores its stray parent",
			child: &model.Image{Name: "child", BuildBackend: nix,
				Variants: map[string]*model.ImageVariant{"v": {Name: "v", BuildBackend: dockerfile, Nix: &model.NixConfig{Parent: "__hive__/missing:1.0"}}}},
		},
		{
			name: "variant without backend follows the nix image",
			child: &model.Image{Name: "child", BuildBackend: nix,
				Variants: map[string]*model.ImageVariant{"v": {Name: "v", Nix: &model.NixConfig{Parent: ref}}}},
			wantEdge: true,
		},
		{
			name:    "nix image with unknown parent fails",
			child:   &model.Image{Name: "child", BuildBackend: nix, Nix: model.NixConfig{Parent: "__hive__/missing:1.0"}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		project := &model.ContainerHiveProject{
			ImagesByName: map[string][]*model.Image{
				"base":  {{Name: "base"}},
				"child": {tt.child},
			},
		}

		t.Run("resolve/"+tt.name, func(t *testing.T) {
			graph, err := BuildDependencyGraph(NewGraph(), project)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if got := slices.Contains(graph.edges["child"], "base"); got != tt.wantEdge {
				t.Errorf("edge child->base = %v, want %v", got, tt.wantEdge)
			}
		})

		if tt.wantErr {
			continue
		}
		t.Run("scan/"+tt.name, func(t *testing.T) {
			graph := ScanProjectSource(project)
			if got := slices.Contains(graph.edges["child"], "base"); got != tt.wantEdge {
				t.Errorf("edge child->base = %v, want %v", got, tt.wantEdge)
			}
		})
	}
}
