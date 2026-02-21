package deps

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/timo-reymann/ContainerHive/pkg/model"
)

func setupDistDir(t *testing.T, images map[string]map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for imgName, tags := range images {
		for tagName, dockerfile := range tags {
			tagDir := filepath.Join(dir, imgName, tagName)
			if err := os.MkdirAll(tagDir, 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(tagDir, "Dockerfile"), []byte(dockerfile), 0644); err != nil {
				t.Fatal(err)
			}
		}
	}
	return dir
}

func TestResolveOrder_NoDeps(t *testing.T) {
	dist := setupDistDir(t, map[string]map[string]string{
		"app": {"latest": "FROM ubuntu:22.04\n"},
	})

	project := &model.ContainerHiveProject{
		ImagesByName: map[string][]*model.Image{
			"app": {{Name: "app"}},
		},
		ImagesByIdentifier: map[string]*model.Image{
			"app": {Name: "app"},
		},
	}

	bo, err := ResolveOrder(dist, project)
	if err != nil {
		t.Fatal(err)
	}

	if bo.HasDependencies() {
		t.Error("expected no dependencies")
	}

	order := bo.Order()
	if len(order) != 1 || order[0] != "app" {
		t.Errorf("unexpected order: %v", order)
	}
}

func TestResolveOrder_WithHiveRefs(t *testing.T) {
	dist := setupDistDir(t, map[string]map[string]string{
		"base": {"1.0": "FROM ubuntu:22.04\n"},
		"app":  {"latest": "FROM __hive__/base:1.0\nRUN echo hello\n"},
	})

	project := &model.ContainerHiveProject{
		ImagesByName: map[string][]*model.Image{
			"base": {{Name: "base"}},
			"app":  {{Name: "app"}},
		},
		ImagesByIdentifier: map[string]*model.Image{
			"base": {Name: "base"},
			"app":  {Name: "app"},
		},
	}

	bo, err := ResolveOrder(dist, project)
	if err != nil {
		t.Fatal(err)
	}

	if !bo.HasDependencies() {
		t.Error("expected dependencies")
	}

	order := bo.Order()
	if len(order) != 2 {
		t.Fatalf("expected 2 images in order, got %d: %v", len(order), order)
	}
	if order[0] != "base" || order[1] != "app" {
		t.Errorf("expected [base app], got %v", order)
	}

	deps := bo.Dependents("base")
	if len(deps) != 1 || deps[0] != "app" {
		t.Errorf("expected [app] as dependents of base, got %v", deps)
	}
}

func TestResolveOrder_ExplicitDependsOn(t *testing.T) {
	dist := setupDistDir(t, map[string]map[string]string{
		"lib": {"1.0": "FROM scratch\n"},
		"svc": {"latest": "FROM ubuntu:22.04\n"},
	})

	project := &model.ContainerHiveProject{
		ImagesByName: map[string][]*model.Image{
			"lib": {{Name: "lib"}},
			"svc": {{Name: "svc", DependsOn: []string{"lib"}}},
		},
		ImagesByIdentifier: map[string]*model.Image{
			"lib": {Name: "lib"},
			"svc": {Name: "svc", DependsOn: []string{"lib"}},
		},
	}

	bo, err := ResolveOrder(dist, project)
	if err != nil {
		t.Fatal(err)
	}

	if !bo.HasDependencies() {
		t.Error("expected dependencies from depends_on")
	}

	order := bo.Order()
	if len(order) != 2 || order[0] != "lib" {
		t.Errorf("expected lib before svc, got %v", order)
	}
}

func TestResolveOrder_InvalidDistPath(t *testing.T) {
	project := &model.ContainerHiveProject{
		ImagesByName:       map[string][]*model.Image{},
		ImagesByIdentifier: map[string]*model.Image{},
	}

	_, err := ResolveOrder("/nonexistent/path", project)
	if err == nil {
		t.Error("expected error for nonexistent dist path")
	}
}
