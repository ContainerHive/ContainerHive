package ci

import (
	"reflect"
	"testing"

	"github.com/ContainerHive/ContainerHive/pkg/model"
)

func TestBuildCIContext_NoDeps(t *testing.T) {
	project := &model.ContainerHiveProject{
		Config: model.HiveProjectConfig{
			Platforms: []string{"linux/amd64", "linux/arm64"},
		},
		ImagesByName: map[string][]*model.Image{
			"app": {{
				Name:      "app",
				Tags:      map[string]*model.Tag{"latest": {Name: "latest"}},
				DependsOn: nil,
			}},
		},
	}

	ctx, err := BuildCIContext(project, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(ctx.Images) != 1 {
		t.Fatalf("expected 1 image, got %d", len(ctx.Images))
	}
	if ctx.Images[0].Name != "app" {
		t.Errorf("expected image name 'app', got %q", ctx.Images[0].Name)
	}
	if ctx.Images[0].Depth != 0 {
		t.Errorf("expected depth 0, got %d", ctx.Images[0].Depth)
	}
	if !reflect.DeepEqual(ctx.Images[0].Platforms, []string{"amd64", "arm64"}) {
		t.Errorf("unexpected platforms: %v", ctx.Images[0].Platforms)
	}
	if len(ctx.Images[0].Dependencies) != 0 {
		t.Errorf("expected no dependencies, got %v", ctx.Images[0].Dependencies)
	}
}

func TestBuildCIContext_WithDeps(t *testing.T) {
	project := &model.ContainerHiveProject{
		Config: model.HiveProjectConfig{
			Platforms: []string{"linux/amd64"},
		},
		ImagesByName: map[string][]*model.Image{
			"base": {{
				Name: "base",
				Tags: map[string]*model.Tag{"latest": {Name: "latest"}},
			}},
			"app": {{
				Name:      "app",
				Tags:      map[string]*model.Tag{"v1": {Name: "v1"}},
				DependsOn: []string{"base"},
			}},
		},
	}

	ctx, err := BuildCIContext(project, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(ctx.Images) != 2 {
		t.Fatalf("expected 2 images, got %d", len(ctx.Images))
	}

	// base should come first (depth 0)
	if ctx.Images[0].Name != "base" || ctx.Images[0].Depth != 0 {
		t.Errorf("expected base at depth 0, got %q at depth %d", ctx.Images[0].Name, ctx.Images[0].Depth)
	}

	// app should be second (depth 1)
	if ctx.Images[1].Name != "app" || ctx.Images[1].Depth != 1 {
		t.Errorf("expected app at depth 1, got %q at depth %d", ctx.Images[1].Name, ctx.Images[1].Depth)
	}
	if !reflect.DeepEqual(ctx.Images[1].Dependencies, []string{"base"}) {
		t.Errorf("expected dependencies [base], got %v", ctx.Images[1].Dependencies)
	}
}

func TestBuildCIContext_ImagePlatformOverride(t *testing.T) {
	project := &model.ContainerHiveProject{
		Config: model.HiveProjectConfig{
			Platforms: []string{"linux/amd64", "linux/arm64"},
		},
		ImagesByName: map[string][]*model.Image{
			"special": {{
				Name:      "special",
				Tags:      map[string]*model.Tag{"latest": {Name: "latest"}},
				Platforms: []string{"linux/amd64"},
			}},
		},
	}

	ctx, err := BuildCIContext(project, false)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(ctx.Images[0].Platforms, []string{"amd64"}) {
		t.Errorf("expected [amd64], got %v", ctx.Images[0].Platforms)
	}
}

func TestBuildCIContext_Stages(t *testing.T) {
	project := &model.ContainerHiveProject{
		Config: model.HiveProjectConfig{
			Platforms: []string{"linux/amd64"},
		},
		ImagesByName: map[string][]*model.Image{
			"alpha": {{Name: "alpha", Tags: map[string]*model.Tag{"v1": {Name: "v1"}}}},
			"beta":  {{Name: "beta", Tags: map[string]*model.Tag{"v1": {Name: "v1"}}, DependsOn: []string{"alpha"}}},
		},
	}

	ctx, err := BuildCIContext(project, false)
	if err != nil {
		t.Fatal(err)
	}
	expected := []string{"build-alpha", "test-alpha", "manifest-alpha", "build-beta", "test-beta", "manifest-beta"}
	if !reflect.DeepEqual(ctx.Stages, expected) {
		t.Errorf("expected stages %v, got %v", expected, ctx.Stages)
	}
}

func TestResolvePlatforms(t *testing.T) {
	t.Run("strips linux prefix", func(t *testing.T) {
		img := &model.Image{Platforms: []string{"linux/amd64", "linux/arm64"}}
		result := resolvePlatforms(img, nil)
		if !reflect.DeepEqual(result, []string{"amd64", "arm64"}) {
			t.Errorf("expected [amd64 arm64], got %v", result)
		}
	})

	t.Run("falls back to project defaults", func(t *testing.T) {
		img := &model.Image{}
		result := resolvePlatforms(img, []string{"linux/amd64"})
		if !reflect.DeepEqual(result, []string{"amd64"}) {
			t.Errorf("expected [amd64], got %v", result)
		}
	})

	t.Run("platform without prefix", func(t *testing.T) {
		img := &model.Image{Platforms: []string{"amd64"}}
		result := resolvePlatforms(img, nil)
		if !reflect.DeepEqual(result, []string{"amd64"}) {
			t.Errorf("expected [amd64], got %v", result)
		}
	})
}

func TestCalculateDepths_Circular(t *testing.T) {
	imageNames := map[string]bool{"a": true, "b": true}
	deps := map[string][]string{
		"a": {"b"},
		"b": {"a"},
	}
	depths := calculateDepths(imageNames, deps)
	if depths["a"] != depths["b"] {
		t.Errorf("circular deps should have same depth, got a=%d b=%d", depths["a"], depths["b"])
	}
}

func TestBuildCIContext_DefaultTemplateOptions(t *testing.T) {
	project := &model.ContainerHiveProject{
		Config: model.HiveProjectConfig{
			Platforms: []string{"linux/amd64"},
		},
		ImagesByName: map[string][]*model.Image{
			"app": {{Name: "app", Tags: map[string]*model.Tag{"latest": {Name: "latest"}}}},
		},
	}

	ctx, err := BuildCIContext(project, false)
	if err != nil {
		t.Fatal(err)
	}

	if ctx.TemplateOptions["ci_buildkit_image"] != "moby/buildkit" {
		t.Errorf("expected default ci_buildkit_image 'moby/buildkit', got %q", ctx.TemplateOptions["ci_buildkit_image"])
	}
	if ctx.TemplateOptions["ci_buildkit_version"] == "" {
		t.Error("expected ci_buildkit_version to have a default value")
	}
	if ctx.TemplateOptions["ci_lint"] != "true" {
		t.Errorf("expected default ci_lint 'true', got %q", ctx.TemplateOptions["ci_lint"])
	}
	if ctx.TemplateOptions["ci_report"] != "true" {
		t.Errorf("expected default ci_report 'true', got %q", ctx.TemplateOptions["ci_report"])
	}
}

func TestBuildCIContext_UserOverridesTemplateOptions(t *testing.T) {
	project := &model.ContainerHiveProject{
		Config: model.HiveProjectConfig{
			Platforms: []string{"linux/amd64"},
			TemplateOptions: map[string]string{
				"ci_buildkit_image": "registry.io/buildkit",
				"ci_report":         "false",
				"custom_var":        "custom_value",
			},
		},
		ImagesByName: map[string][]*model.Image{
			"app": {{Name: "app", Tags: map[string]*model.Tag{"latest": {Name: "latest"}}}},
		},
	}

	ctx, err := BuildCIContext(project, false)
	if err != nil {
		t.Fatal(err)
	}

	if ctx.TemplateOptions["ci_buildkit_image"] != "registry.io/buildkit" {
		t.Errorf("expected user override 'registry.io/buildkit', got %q", ctx.TemplateOptions["ci_buildkit_image"])
	}
	if ctx.TemplateOptions["ci_buildkit_version"] == "" {
		t.Error("expected ci_buildkit_version default to still be present")
	}
	if ctx.TemplateOptions["custom_var"] != "custom_value" {
		t.Errorf("expected custom_var 'custom_value', got %q", ctx.TemplateOptions["custom_var"])
	}
	if ctx.TemplateOptions["ci_report"] != "false" {
		t.Errorf("expected user override ci_report 'false', got %q", ctx.TemplateOptions["ci_report"])
	}
}
