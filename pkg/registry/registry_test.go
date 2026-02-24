package registry

import (
	"context"
	"sort"
	"testing"

	"github.com/timo-reymann/ContainerHive/pkg/build"
	"github.com/timo-reymann/ContainerHive/pkg/model"
)

func TestCollectAllTags_NoVariants(t *testing.T) {
	img := &model.Image{
		Name: "app",
		Tags: map[string]*model.Tag{
			"1.0": {},
			"2.0": {},
		},
		Variants: map[string]*model.ImageVariant{},
	}

	tags := collectAllTags(img)
	sort.Strings(tags)
	if len(tags) != 2 {
		t.Fatalf("expected 2 tags, got %d: %v", len(tags), tags)
	}
	if tags[0] != "1.0" || tags[1] != "2.0" {
		t.Errorf("unexpected tags: %v", tags)
	}
}

func TestCollectAllTags_WithVariants(t *testing.T) {
	img := &model.Image{
		Name: "app",
		Tags: map[string]*model.Tag{
			"1.0": {},
		},
		Variants: map[string]*model.ImageVariant{
			"slim":   {TagSuffix: "-slim"},
			"alpine": {TagSuffix: "-alpine"},
		},
	}

	tags := collectAllTags(img)
	sort.Strings(tags)
	if len(tags) != 3 {
		t.Fatalf("expected 3 tags, got %d: %v", len(tags), tags)
	}

	expected := []string{"1.0", "1.0-alpine", "1.0-slim"}
	for i, want := range expected {
		if tags[i] != want {
			t.Errorf("tag[%d] = %q, want %q", i, tags[i], want)
		}
	}
}

func TestRetagAllAliases_FilterMatch(t *testing.T) {
	project := &model.ContainerHiveProject{
		ImagesByIdentifier: map[string]*model.Image{
			"app": {
				Name:     "app",
				Tags:     map[string]*model.Tag{"latest": {}},
				Variants: map[string]*model.ImageVariant{},
			},
			"lib": {
				Name:     "lib",
				Tags:     map[string]*model.Tag{"1.0": {}},
				Variants: map[string]*model.ImageVariant{},
			},
		},
	}

	reg := &Registry{inner: &noopRegistry{}}

	// Filter to only "app" — should not error (no actual retag since tags aren't semver)
	if err := reg.RetagAllAliases(project, []build.Filter{{ImageName: "app"}}, ""); err != nil {
		t.Fatal(err)
	}

	// Empty filter — processes all
	if err := reg.RetagAllAliases(project, nil, ""); err != nil {
		t.Fatal(err)
	}
}

// noopRegistry implements internal/registry.Registry for testing.
type noopRegistry struct{}

func (n *noopRegistry) Start(_ context.Context) error                        { return nil }
func (n *noopRegistry) Stop(_ context.Context) error                         { return nil }
func (n *noopRegistry) Address() string                                      { return "127.0.0.1:0" }
func (n *noopRegistry) IsLocal() bool                                        { return true }
func (n *noopRegistry) Push(_ context.Context, _, _, _ string) error         { return nil }
