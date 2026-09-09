package registry

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"testing"

	"github.com/ContainerHive/ContainerHive/pkg/build"
	"github.com/ContainerHive/ContainerHive/pkg/model"
	"github.com/ContainerHive/ContainerHive/pkg/rendering"
)

// candidateNames extracts the Name field from a slice of AliasCandidate, for
// tests that only care about tag names.
func candidateNames(candidates []rendering.AliasCandidate) []string {
	names := make([]string, len(candidates))
	for i, c := range candidates {
		names[i] = c.Name
	}
	return names
}

func TestCollectAllTags_NoVariants(t *testing.T) {
	img := &model.Image{
		Name: "app",
		Tags: map[string]*model.Tag{
			"1.0": {},
			"2.0": {},
		},
		Variants: map[string]*model.ImageVariant{},
	}

	tags := candidateNames(collectAllTags(img))
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

	tags := candidateNames(collectAllTags(img))
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

func TestCollectAllTags_DeterministicAcrossRuns(t *testing.T) {
	// img.Tags and img.Variants are maps; without sorting, collectAllTags
	// would return the tags in Go's randomized iteration order, and any
	// alias-resolution tie (e.g. a release vs. its prerelease sharing the
	// same alias) would pick a different winner on every run. Rebuild the
	// map fresh each iteration to actually exercise the randomization.
	names := []string{"1.0.0", "1.0.0-rc1", "2.0.0", "2.0.0-rc1", "3.0.0"}
	var first []string
	for i := 0; i < 20; i++ {
		tags := make(map[string]*model.Tag, len(names))
		for _, n := range names {
			tags[n] = &model.Tag{}
		}
		img := &model.Image{Name: "app", Tags: tags, Variants: map[string]*model.ImageVariant{}}

		got := candidateNames(collectAllTags(img))
		if first == nil {
			first = got
			continue
		}
		if !slices.Equal(got, first) {
			t.Fatalf("collectAllTags order is not deterministic: run 0 = %v, run %d = %v", first, i, got)
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

func TestRetagAllAliases_VariantLatestAlias(t *testing.T) {
	img := &model.Image{
		Name: "app",
		Tags: map[string]*model.Tag{
			"8.0.100": {},
			"8.0.300": {},
		},
		Variants: map[string]*model.ImageVariant{
			"browsers": {TagSuffix: "-browsers"},
		},
		LatestAlias: &model.LatestAliasConfig{Tag: "latest", OnMissing: "error"},
	}

	project := &model.ContainerHiveProject{
		ImagesByIdentifier: map[string]*model.Image{"app": img},
	}

	reg := &Registry{inner: &noopRegistry{}}

	// Should not error: aliases are computed and retag errors are logged as warnings
	if err := reg.RetagAllAliases(project, nil, ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// noopRegistry implements internal/registry.Registry for testing.
type noopRegistry struct{}

func (n *noopRegistry) Start(_ context.Context) error                { return nil }
func (n *noopRegistry) Stop(_ context.Context) error                 { return nil }
func (n *noopRegistry) Address() string                              { return "127.0.0.1:0" }
func (n *noopRegistry) IsLocal() bool                                { return true }
func (n *noopRegistry) Push(_ context.Context, _, _, _ string) error { return nil }
func (n *noopRegistry) UseDockerMediaTypes() bool                    { return false }

// ---------------------------------------------------------------------------
// pushTag
// ---------------------------------------------------------------------------

func TestPushTag_WithoutBuildID(t *testing.T) {
	got := pushTag("1.0", "linux/amd64", "")
	want := "1.0.linux-amd64"
	if got != want {
		t.Errorf("pushTag(%q, %q, %q) = %q, want %q", "1.0", "linux/amd64", "", got, want)
	}
}

func TestPushTag_WithBuildID(t *testing.T) {
	got := pushTag("1.0", "linux/amd64", "abc123")
	want := "1.0.linux-amd64-build.abc123"
	if got != want {
		t.Errorf("pushTag(%q, %q, %q) = %q, want %q", "1.0", "linux/amd64", "abc123", got, want)
	}
}

// ---------------------------------------------------------------------------
// matchesImageFilter
// ---------------------------------------------------------------------------

func TestMatchesImageFilter_EmptyFiltersMatchesEverything(t *testing.T) {
	if !matchesImageFilter(nil, "any-image") {
		t.Error("empty filters should match everything")
	}
}

func TestMatchesImageFilter_MatchingImageName(t *testing.T) {
	filters := []build.Filter{{ImageName: "app"}}
	if !matchesImageFilter(filters, "app") {
		t.Error("filter with matching image name should return true")
	}
}

func TestMatchesImageFilter_NonMatchingImageName(t *testing.T) {
	filters := []build.Filter{{ImageName: "app"}}
	if matchesImageFilter(filters, "lib") {
		t.Error("filter with non-matching image name should return false")
	}
}

func TestMatchesImageFilter_EmptyImageNameMatchesAll(t *testing.T) {
	filters := []build.Filter{{ImageName: ""}}
	if !matchesImageFilter(filters, "anything") {
		t.Error("filter with empty ImageName should match all images")
	}
}

// ---------------------------------------------------------------------------
// matchesTagFilter
// ---------------------------------------------------------------------------

func TestMatchesTagFilter_EmptyFiltersMatchesEverything(t *testing.T) {
	if !matchesTagFilter(nil, "app", "1.0") {
		t.Error("empty filters should match everything")
	}
}

func TestMatchesTagFilter_MatchingImageAndTag(t *testing.T) {
	filters := []build.Filter{{ImageName: "app", TagName: "1.0"}}
	if !matchesTagFilter(filters, "app", "1.0") {
		t.Error("filter matching both image and tag should return true")
	}
}

func TestMatchesTagFilter_NonMatchingImage(t *testing.T) {
	filters := []build.Filter{{ImageName: "app", TagName: "1.0"}}
	if matchesTagFilter(filters, "lib", "1.0") {
		t.Error("filter with non-matching image should return false")
	}
}

func TestMatchesTagFilter_NonMatchingTag(t *testing.T) {
	filters := []build.Filter{{ImageName: "app", TagName: "1.0"}}
	if matchesTagFilter(filters, "app", "2.0") {
		t.Error("filter with non-matching tag should return false")
	}
}

func TestMatchesTagFilter_EmptyTagNameMatchesAllTagsForImage(t *testing.T) {
	filters := []build.Filter{{ImageName: "app", TagName: ""}}
	if !matchesTagFilter(filters, "app", "any-tag") {
		t.Error("filter with empty TagName should match all tags for that image")
	}
}

func TestMatchesTagFilter_EmptyImageNameMatchesAllImages(t *testing.T) {
	filters := []build.Filter{{ImageName: "", TagName: "1.0"}}
	if !matchesTagFilter(filters, "any-image", "1.0") {
		t.Error("filter with empty ImageName should match all images for the given tag")
	}
}

// ---------------------------------------------------------------------------
// ImageRef
// ---------------------------------------------------------------------------

func TestImageRef_FormatWithoutBuildID(t *testing.T) {
	reg := &Registry{inner: &noopRegistry{}}
	got := reg.ImageRef("myapp", "1.0", "linux/amd64", "")
	want := "127.0.0.1:0/myapp:1.0.linux-amd64"
	if got != want {
		t.Errorf("ImageRef() = %q, want %q", got, want)
	}
}

func TestImageRef_FormatWithBuildID(t *testing.T) {
	reg := &Registry{inner: &noopRegistry{}}
	got := reg.ImageRef("myapp", "1.0", "linux/amd64", "abc123")
	want := "127.0.0.1:0/myapp:1.0.linux-amd64-build.abc123"
	if got != want {
		t.Errorf("ImageRef() = %q, want %q", got, want)
	}
}

// ---------------------------------------------------------------------------
// collectBaseTags
// ---------------------------------------------------------------------------

func TestCollectBaseTags(t *testing.T) {
	img := &model.Image{
		Name: "app",
		Tags: map[string]*model.Tag{
			"1.0": {},
			"2.0": {},
		},
		Variants: map[string]*model.ImageVariant{
			"slim": {TagSuffix: "-slim"},
		},
	}

	tags := candidateNames(collectBaseTags(img))
	sort.Strings(tags)
	if len(tags) != 2 {
		t.Fatalf("expected 2 base tags, got %d: %v", len(tags), tags)
	}
	if tags[0] != "1.0" || tags[1] != "2.0" {
		t.Errorf("unexpected base tags: %v", tags)
	}
}

// ---------------------------------------------------------------------------
// collectAllTags – edge case: image with no tags
// ---------------------------------------------------------------------------

func TestCollectAllTags_NoTags(t *testing.T) {
	img := &model.Image{
		Name:     "app",
		Tags:     map[string]*model.Tag{},
		Variants: map[string]*model.ImageVariant{},
	}

	tags := collectAllTags(img)
	if len(tags) != 0 {
		t.Fatalf("expected 0 tags for image with no tags, got %d: %v", len(tags), tags)
	}
}

// ---------------------------------------------------------------------------
// loadImageFromTar helpers (mirror of internal/docker/main_test.go pattern)
// ---------------------------------------------------------------------------

// buildOCITar creates a minimal valid OCI image tar for testing.
func buildOCITarForRegistry(t *testing.T) string {
	t.Helper()

	config := []byte(`{"architecture":"amd64","os":"linux","rootfs":{"type":"layers","diff_ids":[]}}`)
	configDigest := fmt.Sprintf("sha256:%x", sha256.Sum256(config))

	manifest, _ := json.Marshal(map[string]any{
		"schemaVersion": 2,
		"mediaType":     "application/vnd.oci.image.manifest.v1+json",
		"config": map[string]any{
			"mediaType": "application/vnd.oci.image.config.v1+json",
			"digest":    configDigest,
			"size":      len(config),
		},
		"layers": []any{},
	})
	manifestDigest := fmt.Sprintf("sha256:%x", sha256.Sum256(manifest))

	index, _ := json.Marshal(map[string]any{
		"schemaVersion": 2,
		"mediaType":     "application/vnd.oci.image.index.v1+json",
		"manifests": []map[string]any{
			{
				"mediaType": "application/vnd.oci.image.manifest.v1+json",
				"digest":    manifestDigest,
				"size":      len(manifest),
			},
		},
	})

	ociLayout := []byte(`{"imageLayoutVersion":"1.0.0"}`)

	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	writeEntry := func(name string, data []byte) {
		tw.WriteHeader(&tar.Header{Name: name, Size: int64(len(data)), Mode: 0644})
		tw.Write(data)
	}
	writeEntry("oci-layout", ociLayout)
	writeEntry("index.json", index)
	writeEntry(fmt.Sprintf("blobs/sha256/%x", sha256.Sum256(manifest)), manifest)
	writeEntry(fmt.Sprintf("blobs/sha256/%x", sha256.Sum256(config)), config)
	tw.Close()

	p := filepath.Join(t.TempDir(), "image.tar")
	if err := os.WriteFile(p, buf.Bytes(), 0644); err != nil {
		t.Fatal(err)
	}
	return p
}

// buildOCITarNoManifestsForRegistry creates an OCI tar with an empty manifests array.
func buildOCITarNoManifestsForRegistry(t *testing.T) string {
	t.Helper()

	index, _ := json.Marshal(map[string]any{
		"schemaVersion": 2,
		"manifests":     []any{},
	})
	ociLayout := []byte(`{"imageLayoutVersion":"1.0.0"}`)

	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	writeEntry := func(name string, data []byte) {
		tw.WriteHeader(&tar.Header{Name: name, Size: int64(len(data)), Mode: 0644})
		tw.Write(data)
	}
	writeEntry("oci-layout", ociLayout)
	writeEntry("index.json", index)
	tw.Close()

	p := filepath.Join(t.TempDir(), "no-manifests.tar")
	if err := os.WriteFile(p, buf.Bytes(), 0644); err != nil {
		t.Fatal(err)
	}
	return p
}

// ---------------------------------------------------------------------------
// loadImageFromTar
// ---------------------------------------------------------------------------

func TestLoadImageFromTar_ValidOCITar(t *testing.T) {
	tarPath := buildOCITarForRegistry(t)
	ociImage, err := loadImageFromTar(tarPath)
	if err != nil {
		t.Fatalf("expected no error for valid OCI tar, got: %v", err)
	}
	if ociImage == nil {
		t.Fatal("expected non-nil OCIImage for valid OCI tar")
	}
	defer ociImage.Cleanup()

	if ociImage.Image == nil {
		t.Error("expected non-nil v1.Image inside OCIImage")
	}
}

func TestLoadImageFromTar_NonexistentPath(t *testing.T) {
	_, err := loadImageFromTar("/nonexistent/path/image.tar")
	if err == nil {
		t.Fatal("expected error for nonexistent tar path, got nil")
	}
	t.Logf("got expected error: %v", err)
}

func TestLoadImageFromTar_EmptyManifests(t *testing.T) {
	tarPath := buildOCITarNoManifestsForRegistry(t)
	_, err := loadImageFromTar(tarPath)
	if err == nil {
		t.Fatal("expected error for tar with empty manifests, got nil")
	}
	if err.Error() != "no manifests in OCI layout" {
		t.Fatalf("unexpected error message: %v", err)
	}
	t.Logf("got expected error: %v", err)
}
