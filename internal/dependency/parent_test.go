package dependency

import (
	"slices"
	"testing"

	"github.com/ContainerHive/ContainerHive/pkg/model"
)

func TestBackendParentEdges_nix(t *testing.T) {
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

func TestBackendParentEdges_chisel(t *testing.T) {
	const ref = "__hive__/base:1.0"
	chisel := model.BuildBackendChisel
	dockerfile := model.BuildBackendDockerfile

	tests := []struct {
		name     string
		child    *model.Image
		wantEdge bool
		wantErr  bool
	}{
		{
			name:  "dockerfile image ignores a stray chisel parent",
			child: &model.Image{Name: "child", BuildBackend: dockerfile, Chisel: model.ChiselConfig{Parent: "__hive__/missing:1.0"}},
		},
		{
			name:     "chisel image uses its parent",
			child:    &model.Image{Name: "child", BuildBackend: chisel, Chisel: model.ChiselConfig{Parent: ref}},
			wantEdge: true,
		},
		{
			name: "chisel variant with own parent",
			child: &model.Image{Name: "child", BuildBackend: dockerfile,
				Variants: map[string]*model.ImageVariant{"v": {Name: "v", BuildBackend: chisel, Chisel: &model.ChiselConfig{Parent: ref}}}},
			wantEdge: true,
		},
		{
			name: "chisel variant without block inherits the image parent",
			child: &model.Image{Name: "child", BuildBackend: chisel, Chisel: model.ChiselConfig{Parent: ref},
				Variants: map[string]*model.ImageVariant{"v": {Name: "v"}}},
			wantEdge: true,
		},
		{
			name: "dockerfile variant of a chisel image ignores its stray parent",
			child: &model.Image{Name: "child", BuildBackend: chisel,
				Variants: map[string]*model.ImageVariant{"v": {Name: "v", BuildBackend: dockerfile, Chisel: &model.ChiselConfig{Parent: "__hive__/missing:1.0"}}}},
		},
		{
			name: "chisel parent and variant parent both add edges",
			child: &model.Image{Name: "child", BuildBackend: chisel, Chisel: model.ChiselConfig{Parent: "__hive__/other:1.0"},
				Variants: map[string]*model.ImageVariant{"v": {Name: "v", BuildBackend: chisel, Chisel: &model.ChiselConfig{Parent: "__hive__/base:1.0"}}}},
			wantEdge: true,
		},
	}

	for _, tt := range tests {
		project := &model.ContainerHiveProject{
			ImagesByName: map[string][]*model.Image{
				"base":  {{Name: "base"}},
				"other": {{Name: "other"}},
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
				t.Errorf("edge child->base = %v, want %v (edges: %v)", got, tt.wantEdge, graph.edges["child"])
			}
		})

		if tt.wantErr {
			continue
		}
		t.Run("scan/"+tt.name, func(t *testing.T) {
			graph := ScanProjectSource(project)
			if tt.child.Nix.Parent == "" {
				// Chisel cases assert the base edge through the shared loop.
			}
			if got := slices.Contains(graph.edges["child"], "base"); got != tt.wantEdge {
				t.Errorf("edge child->base = %v, want %v (edges: %v)", got, tt.wantEdge, graph.edges["child"])
			}
		})
	}
}

func TestBackendParentEdges_chiselPlaceholder(t *testing.T) {
	// The parent placeholder carries no build order edge.
	project := &model.ContainerHiveProject{
		ImagesByName: map[string][]*model.Image{
			"child": {{Name: "child", BuildBackend: model.BuildBackendChisel,
				Chisel: model.ChiselConfig{Parent: model.HiveParentPlaceholder}}},
		},
	}
	graph, err := BuildDependencyGraph(NewGraph(), project)
	if err != nil {
		t.Fatalf("BuildDependencyGraph failed: %v", err)
	}
	if len(graph.edges["child"]) != 0 {
		t.Errorf("placeholder edges = %v, want none", graph.edges["child"])
	}
}
