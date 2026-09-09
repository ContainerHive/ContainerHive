package discovery

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/ContainerHive/ContainerHive/internal/tagrange"
	"github.com/ContainerHive/ContainerHive/internal/tagrange/source"
	"github.com/ContainerHive/ContainerHive/pkg/model"
)

// fixtureSource is a VersionSource entirely under test control, used so
// discovery-level tag_ranges tests never touch the network.
type fixtureSource struct {
	versions []source.Version
}

func (fixtureSource) Name() string                                          { return "fixture" }
func (fixtureSource) Validate(*model.SourceConfig) error                     { return nil }
func (fixtureSource) Descriptor(*model.SourceConfig) string                  { return "fixture" }
func (fixtureSource) CacheKeyParts(*model.SourceConfig) []string             { return []string{"fixture"} }
func (f fixtureSource) Fetch(context.Context, *model.SourceConfig) ([]source.Version, error) {
	return f.versions, nil
}

func fixtureResolver(versions ...string) *tagrange.Resolver {
	vs := make([]source.Version, len(versions))
	for i, v := range versions {
		vs[i] = source.Version{Raw: v}
	}
	reg := source.NewRegistry(fixtureSource{versions: vs})
	return tagrange.NewResolver(tagrange.NewCache("/dev/null/unused"), reg)
}

func writeProjectWithTagRanges(t *testing.T, imageYML string) string {
	t.Helper()
	root := t.TempDir()
	imagesDir := filepath.Join(root, "images", "app")
	if err := os.MkdirAll(imagesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "hive.yml"), []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(imagesDir, "image.yml"), []byte(imageYML), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(imagesDir, "Dockerfile"), []byte("FROM scratch\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return root
}

const tagRangeYML = `tags:
  - name: static
tag_ranges:
  - tag_name: "{{.major}}.{{.minor}}"
    source:
      type: fixture
    select:
      minor: { all: true }
`

func TestDiscoverProject_ResolvesTagRanges(t *testing.T) {
	root := writeProjectWithTagRanges(t, tagRangeYML)
	project, err := DiscoverProject(context.Background(), root, WithTagRangeResolver(fixtureResolver("1.2.0", "1.3.0")))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	img := project.ImagesByIdentifier["app"]
	if img == nil {
		t.Fatal("expected image \"app\" to be discovered")
	}
	if _, ok := img.Tags["static"]; !ok {
		t.Error("expected the static tag to survive")
	}
	if _, ok := img.Tags["1.2"]; !ok {
		t.Errorf("expected generated tag 1.2, got tags: %v", tagNames(img.Tags))
	}
	if _, ok := img.Tags["1.3"]; !ok {
		t.Errorf("expected generated tag 1.3, got tags: %v", tagNames(img.Tags))
	}
	if img.Tags["1.3"].GeneratedFrom == "" {
		t.Error("expected the generated tag to record GeneratedFrom")
	}
}

func TestDiscoverProject_WithoutTagRanges_NeverConstructsResolver(t *testing.T) {
	root := writeProjectWithTagRanges(t, "tags:\n  - name: static\n")
	// No resolver option given, and no tag_ranges in the config: default
	// DiscoverProject must not attempt to build a network-capable resolver
	// (which would try to resolve an XDG cache directory) — it must be a
	// pure no-op.
	project, err := DiscoverProject(context.Background(), root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := project.ImagesByIdentifier["app"].Tags["static"]; !ok {
		t.Error("expected the static tag to survive")
	}
}

func TestDiscoverProject_StaticTagShadowsGenerated(t *testing.T) {
	imageYML := `tags:
  - name: "1.2"
tag_ranges:
  - tag_name: "{{.major}}.{{.minor}}"
    source:
      type: fixture
    select:
      minor: { all: true }
`
	root := writeProjectWithTagRanges(t, imageYML)
	project, err := DiscoverProject(context.Background(), root, WithTagRangeResolver(fixtureResolver("1.2.0")))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	tag := project.ImagesByIdentifier["app"].Tags["1.2"]
	if tag.GeneratedFrom != "" {
		t.Error("expected the static tag to win and keep GeneratedFrom empty")
	}
}

func TestDiscoverProject_CrossRangeCollisionIsError(t *testing.T) {
	imageYML := `tags: []
tag_ranges:
  - tag_name: "{{.major}}.{{.minor}}"
    source:
      type: fixture
    select:
      minor: { all: true }
  - tag_name: "{{.major}}.{{.minor}}"
    source:
      type: fixture
    select:
      minor: { all: true }
`
	root := writeProjectWithTagRanges(t, imageYML)
	_, err := DiscoverProject(context.Background(), root, WithTagRangeResolver(fixtureResolver("1.2.0")))
	if err == nil {
		t.Error("expected an error: two ranges emitting the same tag name have no defensible winner")
	}
}

func tagNames(tags map[string]*model.Tag) []string {
	names := make([]string, 0, len(tags))
	for n := range tags {
		names = append(names, n)
	}
	return names
}
