package tagrange

import (
	"math/rand/v2"
	"reflect"
	"strconv"
	"testing"

	"github.com/ContainerHive/ContainerHive/pkg/model"
)

func ptr[T any](v T) *T { return &v }

func nodeVersions() []RawVersion {
	return []RawVersion{
		{Raw: "24.11.1", Extra: map[string]string{"npm": "11.6.2", "openssl": "3.5.4"}},
		{Raw: "24.10.0", Extra: map[string]string{"npm": "11.5.0", "openssl": "3.5.3"}},
		{Raw: "24.9.0", Extra: map[string]string{"npm": "11.4.0", "openssl": "3.5.2"}},
		{Raw: "22.22.0", Extra: map[string]string{"npm": "10.9.2", "openssl": "3.4.1"}},
		{Raw: "22.21.1", Extra: map[string]string{"npm": "10.9.0", "openssl": "3.4.0"}},
		{Raw: "20.20.0", Extra: map[string]string{"npm": "10.8.2", "openssl": "3.0.17"}},
		{Raw: "20.19.5", Extra: map[string]string{"npm": "10.8.1", "openssl": "3.0.16"}},
		{Raw: "18.20.8", Extra: map[string]string{"npm": "10.8.2", "openssl": "3.0.16"}},
	}
}

// TestGenerateTags_NodeExample walks the exact worked example from the
// plan/issue: 3 majors, 3 minors each, latest patch, guarded by
// max_major_increment.
func TestGenerateTags_NodeExample(t *testing.T) {
	tr := &model.TagRange{
		TagName: "{{.major}}.{{.minor}}",
		Select: &model.SelectConfig{
			Major: &model.LevelSelect{Last: 3},
			Minor: &model.LevelSelect{Last: 3},
			Patch: &model.LevelSelect{Last: 1},
		},
		Filter: &model.FilterConfig{
			ExcludePrerelease: ptr(true),
			MaxMajorIncrement: 5,
		},
		Versions: model.Versions{
			"nodejs":  "{{.full}}",
			"npm":     "{{.extra.npm}}",
			"openssl": "{{.extra.openssl}}",
		},
	}

	tags, err := GenerateTags("node", tr, nodeVersions())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wantNames := []string{"24.11", "24.10", "24.9", "22.22", "22.21", "20.20", "20.19"}
	if got := names(tags); !reflect.DeepEqual(got, wantNames) {
		t.Fatalf("tag names = %v, want %v", got, wantNames)
	}

	byName := indexByName(tags)
	if v := byName["24.11"].Versions["nodejs"]; v != "24.11.1" {
		t.Errorf("24.11 nodejs version = %q, want 24.11.1", v)
	}
	if v := byName["24.11"].Versions["npm"]; v != "11.6.2" {
		t.Errorf("24.11 npm version = %q, want 11.6.2", v)
	}
	if v := byName["24.11"].Versions["openssl"]; v != "3.5.4" {
		t.Errorf("24.11 openssl version = %q, want 3.5.4", v)
	}
}

// TestGenerateTags_NodeExample_OrderIndependent proves selectVersions'
// output doesn't depend on the order the source returned versions in —
// upstream APIs are not guaranteed to return a sorted list.
func TestGenerateTags_NodeExample_OrderIndependent(t *testing.T) {
	tr := &model.TagRange{
		TagName: "{{.major}}.{{.minor}}",
		Select: &model.SelectConfig{
			Major: &model.LevelSelect{Last: 3},
			Minor: &model.LevelSelect{Last: 3},
			Patch: &model.LevelSelect{Last: 1},
		},
	}
	want := []string{"24.11", "24.10", "24.9", "22.22", "22.21", "20.20", "20.19"}

	for seed := uint64(0); seed < 50; seed++ {
		shuffled := shuffleRaw(nodeVersions(), seed)
		tags, err := GenerateTags("node", tr, shuffled)
		if err != nil {
			t.Fatalf("seed %d: unexpected error: %v", seed, err)
		}
		if got := names(tags); !reflect.DeepEqual(got, want) {
			t.Fatalf("seed %d: tag names = %v, want %v", seed, got, want)
		}
	}
}

func shuffleRaw(versions []RawVersion, seed uint64) []RawVersion {
	out := append([]RawVersion(nil), versions...)
	r := rand.New(rand.NewPCG(seed, seed))
	r.Shuffle(len(out), func(i, j int) { out[i], out[j] = out[j], out[i] })
	return out
}

func TestGenerateTags_MaxMajorIncrement_DropsOutlier(t *testing.T) {
	raw := []RawVersion{{Raw: "18.0.0"}, {Raw: "20.0.0"}, {Raw: "22.0.0"}, {Raw: "24.0.0"}, {Raw: "99.0.0"}}
	tr := &model.TagRange{
		TagName: "{{.major}}",
		Select:  &model.SelectConfig{Major: &model.LevelSelect{All: true}},
		Filter:  &model.FilterConfig{MaxMajorIncrement: 5},
	}
	tags, err := GenerateTags("t", tr, raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := names(tags); reflect.DeepEqual(got, []string{}) || contains(got, "99") {
		t.Errorf("expected 99 to be dropped by max_major_increment, got %v", got)
	}
	if !contains(names(tags), "24") {
		t.Errorf("expected 24 to survive, got %v", names(tags))
	}
}

func TestGenerateTags_MaxMajorIncrement_ZeroDisables(t *testing.T) {
	raw := []RawVersion{{Raw: "18.0.0"}, {Raw: "99.0.0"}}
	tr := &model.TagRange{
		TagName: "{{.major}}",
		Select:  &model.SelectConfig{Major: &model.LevelSelect{All: true}},
		Filter:  &model.FilterConfig{MaxMajorIncrement: 0},
	}
	tags, err := GenerateTags("t", tr, raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !contains(names(tags), "99") {
		t.Errorf("expected 99 to survive with max_major_increment disabled, got %v", names(tags))
	}
}

func TestGenerateTags_ExcludePrereleaseDefaultTrue(t *testing.T) {
	raw := []RawVersion{{Raw: "1.0.0-rc1"}, {Raw: "1.0.0"}}
	tr := &model.TagRange{TagName: "{{.full}}"}
	tags, err := GenerateTags("t", tr, raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tags) != 1 || tags[0].Name != "1.0.0" {
		t.Errorf("expected only 1.0.0 to survive (prerelease excluded by default), got %v", names(tags))
	}
}

func TestGenerateTags_ExcludeSuffixes(t *testing.T) {
	raw := []RawVersion{{Raw: "1.0.0-alpine"}, {Raw: "1.0.0"}}
	tr := &model.TagRange{
		TagName: "{{.full}}",
		Filter:  &model.FilterConfig{ExcludePrerelease: ptr(false), ExcludeSuffixes: []string{"-alpine"}},
	}
	tags, err := GenerateTags("t", tr, raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tags) != 1 || tags[0].Name != "1.0.0" {
		t.Errorf("expected only 1.0.0 to survive, got %v", names(tags))
	}
}

func TestGenerateTags_CollisionHighestWins(t *testing.T) {
	raw := []RawVersion{{Raw: "1.2.3"}, {Raw: "1.2.4"}}
	tr := &model.TagRange{
		TagName: "{{.major}}.{{.minor}}",
		Select:  &model.SelectConfig{Patch: &model.LevelSelect{Last: 2}},
		Versions: model.Versions{
			"full": "{{.full}}",
		},
	}
	tags, err := GenerateTags("t", tr, raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tags) != 1 {
		t.Fatalf("expected the two patches to collide into 1 tag, got %d: %v", len(tags), names(tags))
	}
	if tags[0].Versions["full"] != "1.2.4" {
		t.Errorf("expected the highest version (1.2.4) to win the collision, got %q", tags[0].Versions["full"])
	}
}

func TestGenerateTags_MaxTagsExceeded(t *testing.T) {
	var raw []RawVersion
	for i := 0; i < 60; i++ {
		raw = append(raw, RawVersion{Raw: "1." + strconv.Itoa(i) + ".0"})
	}
	tr := &model.TagRange{
		TagName: "{{.major}}.{{.minor}}",
		Select:  &model.SelectConfig{Minor: &model.LevelSelect{All: true}},
	}
	if _, err := GenerateTags("t", tr, raw); err == nil {
		t.Error("expected an error when generated tags exceed the default max_tags")
	}
}

func TestGenerateTags_InvalidOCITagName(t *testing.T) {
	raw := []RawVersion{{Raw: "1.2.3+build.metadata"}}
	tr := &model.TagRange{TagName: "{{.raw}}"} // "+" is not a valid OCI tag character
	if _, err := GenerateTags("t", tr, raw); err == nil {
		t.Error("expected an error for a tag name containing '+'")
	}
}

func TestGenerateTags_MissingExtraKeyErrors(t *testing.T) {
	raw := []RawVersion{{Raw: "1.0.0", Extra: map[string]string{"npm": "1.0.0"}}}
	tr := &model.TagRange{
		TagName:  "{{.full}}",
		Versions: model.Versions{"npm": "{{.extra.npmm}}"}, // typo
	}
	if _, err := GenerateTags("t", tr, raw); err == nil {
		t.Error("expected an error for a typo'd .extra key rendering empty")
	}
}

func TestGenerateTags_AllUnparsedIsError(t *testing.T) {
	raw := []RawVersion{{Raw: "latest"}, {Raw: "alpine"}}
	tr := &model.TagRange{TagName: "{{.full}}"}
	if _, err := GenerateTags("t", tr, raw); err == nil {
		t.Error("expected an error when nothing parses as semver")
	}
}

func TestGenerateTags_SkipsUnparsedEntries(t *testing.T) {
	raw := []RawVersion{{Raw: "latest"}, {Raw: "1.0.0"}}
	tr := &model.TagRange{TagName: "{{.full}}"}
	tags, err := GenerateTags("t", tr, raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tags) != 1 || tags[0].Name != "1.0.0" {
		t.Errorf("expected \"latest\" to be skipped, got %v", names(tags))
	}
}

func names(tags []GeneratedTag) []string {
	out := make([]string, len(tags))
	for i, tag := range tags {
		out[i] = tag.Name
	}
	return out
}

func indexByName(tags []GeneratedTag) map[string]GeneratedTag {
	m := make(map[string]GeneratedTag, len(tags))
	for _, tag := range tags {
		m[tag.Name] = tag
	}
	return m
}

func contains(s []string, v string) bool {
	for _, item := range s {
		if item == v {
			return true
		}
	}
	return false
}
