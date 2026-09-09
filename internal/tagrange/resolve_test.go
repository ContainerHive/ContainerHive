package tagrange

import (
	"context"
	"errors"
	"testing"

	"github.com/ContainerHive/ContainerHive/internal/tagrange/source"
	"github.com/ContainerHive/ContainerHive/pkg/model"
)

// fixtureSource is a VersionSource entirely under test control, so
// Resolver tests never touch the network.
type fixtureSource struct {
	name     string
	versions []source.Version
	fetchErr error
}

func (f fixtureSource) Name() string                              { return f.name }
func (f fixtureSource) Validate(*model.SourceConfig) error         { return nil }
func (f fixtureSource) Descriptor(cfg *model.SourceConfig) string  { return f.name + " fixture" }
func (f fixtureSource) CacheKeyParts(cfg *model.SourceConfig) []string {
	return []string{f.name, cfg.Type}
}
func (f fixtureSource) Fetch(context.Context, *model.SourceConfig) ([]source.Version, error) {
	if f.fetchErr != nil {
		return nil, f.fetchErr
	}
	return f.versions, nil
}

func TestResolver_ResolveImage_ExplicitSource(t *testing.T) {
	fixture := fixtureSource{name: "fixture", versions: []source.Version{
		{Raw: "1.2.0"}, {Raw: "1.3.0"},
	}}
	reg := source.NewRegistry(fixture)
	resolver := NewResolver(NewCache(t.TempDir()), reg)

	ranges := []*model.TagRange{{
		TagName: "{{.full}}",
		Source:  &model.SourceConfig{Type: "fixture"},
		Select:  &model.SelectConfig{Minor: &model.LevelSelect{All: true}, Patch: &model.LevelSelect{All: true}},
	}}

	tags, err := resolver.ResolveImage(context.Background(), "myimage", ranges)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tags) != 2 {
		t.Fatalf("expected 2 tags, got %d: %v", len(tags), names(tags))
	}
}

func TestResolver_ResolveImage_MissingSelectIsError(t *testing.T) {
	fixture := fixtureSource{name: "fixture", versions: []source.Version{{Raw: "1.0.0"}}}
	reg := source.NewRegistry(fixture)
	resolver := NewResolver(NewCache(t.TempDir()), reg)

	ranges := []*model.TagRange{{
		TagName: "{{.full}}",
		Source:  &model.SourceConfig{Type: "fixture"},
		// Select intentionally omitted.
	}}
	if _, err := resolver.ResolveImage(context.Background(), "myimage", ranges); err == nil {
		t.Error("expected an error when select is omitted for an explicit source")
	}
}

func TestResolver_ResolveImage_SourceAndGeneratorMutuallyExclusive(t *testing.T) {
	reg := source.NewRegistry(fixtureSource{name: "fixture"})
	resolver := NewResolver(NewCache(t.TempDir()), reg)

	ranges := []*model.TagRange{{
		TagName:   "{{.full}}",
		Source:    &model.SourceConfig{Type: "fixture"},
		Generator: "semver-matrix",
	}}
	if _, err := resolver.ResolveImage(context.Background(), "myimage", ranges); err == nil {
		t.Error("expected an error when both source and generator are set")
	}
}

func TestResolver_ResolveImage_NeitherSourceNorGeneratorIsError(t *testing.T) {
	reg := source.NewRegistry(fixtureSource{name: "fixture"})
	resolver := NewResolver(NewCache(t.TempDir()), reg)

	ranges := []*model.TagRange{{TagName: "{{.full}}"}}
	if _, err := resolver.ResolveImage(context.Background(), "myimage", ranges); err == nil {
		t.Error("expected an error when neither source nor generator is set")
	}
}

func TestResolver_ResolveImage_UnknownSourceType(t *testing.T) {
	reg := source.NewRegistry() // empty registry
	resolver := NewResolver(NewCache(t.TempDir()), reg)

	ranges := []*model.TagRange{{
		TagName: "{{.full}}",
		Source:  &model.SourceConfig{Type: "nonexistent"},
		Select:  &model.SelectConfig{},
	}}
	if _, err := resolver.ResolveImage(context.Background(), "myimage", ranges); err == nil {
		t.Error("expected an error for an unknown source type")
	}
}

func TestResolver_ResolveImage_GeneratorExpandsToSource(t *testing.T) {
	fixture := fixtureSource{name: "registry", versions: []source.Version{
		{Raw: "20.1.0"}, {Raw: "20.2.0"}, {Raw: "22.1.0"},
	}}
	reg := source.NewRegistry(fixture)
	resolver := NewResolver(NewCache(t.TempDir()), reg)

	ranges := []*model.TagRange{{
		TagName:   "{{.major}}",
		Generator: "semver-matrix",
		Params:    map[string]string{"source": "registry", "image": "library/node", "majors": "2"},
	}}
	tags, err := resolver.ResolveImage(context.Background(), "node", ranges)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tags) != 2 {
		t.Fatalf("expected 2 tags (majors: 2), got %d: %v", len(tags), names(tags))
	}
}

func TestResolver_ResolveImage_FetchFailurePropagates(t *testing.T) {
	fixture := fixtureSource{name: "fixture", fetchErr: errors.New("boom")}
	reg := source.NewRegistry(fixture)
	resolver := NewResolver(NewCache(t.TempDir()), reg)

	ranges := []*model.TagRange{{
		TagName: "{{.full}}",
		Source:  &model.SourceConfig{Type: "fixture"},
		Select:  &model.SelectConfig{},
	}}
	if _, err := resolver.ResolveImage(context.Background(), "myimage", ranges); err == nil {
		t.Error("expected the fetch error to propagate")
	}
}

func TestSemverMatrixGenerator_Defaults(t *testing.T) {
	cfg, sel, filter, err := expandGenerator("semver-matrix", map[string]string{"source": "dockerhub", "image": "library/node"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Type != "dockerhub" || cfg.Image != "library/node" {
		t.Errorf("cfg = %+v, want type=dockerhub image=library/node", cfg)
	}
	if sel.Major.Last != 1 || sel.Minor.Last != 1 || sel.Patch.Last != 1 {
		t.Errorf("sel = %+v, want all defaults to 1", sel)
	}
	if filter.ExcludePrerelease == nil || !*filter.ExcludePrerelease {
		t.Errorf("filter = %+v, want ExcludePrerelease=true", filter)
	}
}

func TestSemverMatrixGenerator_ExplicitWindow(t *testing.T) {
	_, sel, _, err := expandGenerator("semver-matrix", map[string]string{
		"source": "dockerhub", "image": "library/node", "majors": "3", "minors": "3",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sel.Major.Last != 3 || sel.Minor.Last != 3 {
		t.Errorf("sel = %+v, want majors=3 minors=3", sel)
	}
}

func TestSemverMatrixGenerator_MissingSourceIsError(t *testing.T) {
	if _, _, _, err := expandGenerator("semver-matrix", map[string]string{"image": "x"}); err == nil {
		t.Error("expected an error when source param is missing")
	}
}

func TestSemverMatrixGenerator_UnknownParamIsError(t *testing.T) {
	if _, _, _, err := expandGenerator("semver-matrix", map[string]string{"source": "dockerhub", "bogus": "1"}); err == nil {
		t.Error("expected an error for an unknown param")
	}
}

func TestSemverMatrixGenerator_NonNumericParamIsError(t *testing.T) {
	if _, _, _, err := expandGenerator("semver-matrix", map[string]string{"source": "dockerhub", "majors": "abc"}); err == nil {
		t.Error("expected an error for a non-numeric majors param")
	}
}

func TestExpandGenerator_UnknownGenerator(t *testing.T) {
	if _, _, _, err := expandGenerator("nonexistent-generator", nil); err == nil {
		t.Error("expected an error for an unknown generator name")
	}
}
