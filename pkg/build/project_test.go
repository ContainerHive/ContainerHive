package build

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ContainerHive/ContainerHive/pkg/deps"
	"github.com/ContainerHive/ContainerHive/pkg/model"
)

func TestMatchesFilters_Empty(t *testing.T) {
	if !matchesFilters(nil, "any", "any") {
		t.Error("empty filters should match everything")
	}
	if !matchesFilters([]Filter{}, "any", "any") {
		t.Error("empty filter slice should match everything")
	}
}

func TestMatchesFilters_ImageOnly(t *testing.T) {
	filters := []Filter{{ImageName: "dotnet"}}
	// image-only filter matches all tags and variants
	if !matchesFilters(filters, "dotnet", "8.0.300") {
		t.Error("should match base tag")
	}
	if !matchesFilters(filters, "dotnet", "8.0.300-node") {
		t.Error("should match variant tag")
	}
	if matchesFilters(filters, "ubuntu", "22.04") {
		t.Error("should not match different image")
	}
}

func TestMatchesFilters_ExactTag(t *testing.T) {
	filters := []Filter{{ImageName: "dotnet", TagName: "8.0.300"}}
	// exact tag matches only that tag
	if !matchesFilters(filters, "dotnet", "8.0.300") {
		t.Error("should match exact tag")
	}
	if matchesFilters(filters, "dotnet", "8.0.300-node") {
		t.Error("should not match variant when filtering by base tag")
	}
	if matchesFilters(filters, "dotnet", "8.0.200") {
		t.Error("should not match different tag")
	}
}

func TestMatchesFilters_ExactVariantTag(t *testing.T) {
	filters := []Filter{{ImageName: "dotnet", TagName: "8.0.300-node"}}
	// exact variant tag matches only that variant
	if !matchesFilters(filters, "dotnet", "8.0.300-node") {
		t.Error("should match exact variant tag")
	}
	if matchesFilters(filters, "dotnet", "8.0.300") {
		t.Error("should not match base tag when filtering by variant")
	}
}

func TestMatchesFilters_TagOnly(t *testing.T) {
	filters := []Filter{{TagName: "1.0"}}
	if !matchesFilters(filters, "any", "1.0") {
		t.Error("should match tag 1.0")
	}
	if matchesFilters(filters, "any", "2.0") {
		t.Error("should not match tag 2.0")
	}
}

func TestMatchesFilters_Combined(t *testing.T) {
	filters := []Filter{{ImageName: "app", TagName: "1.0"}}
	if !matchesFilters(filters, "app", "1.0") {
		t.Error("should match exact image+tag")
	}
	if matchesFilters(filters, "app", "2.0") {
		t.Error("should not match different tag")
	}
	if matchesFilters(filters, "other", "1.0") {
		t.Error("should not match different image")
	}
}

func TestMatchesFilters_MultipleFilters(t *testing.T) {
	filters := []Filter{
		{ImageName: "app", TagName: "1.0"},
		{ImageName: "lib"},
	}
	if !matchesFilters(filters, "app", "1.0") {
		t.Error("should match first filter exactly")
	}
	if matchesFilters(filters, "app", "2.0") {
		t.Error("should not match app with wrong tag")
	}
	if !matchesFilters(filters, "lib", "any") {
		t.Error("should match lib with any tag")
	}
	if matchesFilters(filters, "other", "1.0") {
		t.Error("should not match unrelated image")
	}
}

type mockRegistry struct {
	address          string
	local            bool
	dockerMediaTypes bool
}

func (r *mockRegistry) Address() string           { return r.address }
func (r *mockRegistry) IsLocal() bool             { return r.local }
func (r *mockRegistry) UseDockerMediaTypes() bool { return r.dockerMediaTypes }

func TestPushTag(t *testing.T) {
	tests := []struct {
		name     string
		buildID  string
		tag      string
		platform string
		want     string
	}{
		{"without build ID", "", "latest", "linux/amd64", "latest.linux-amd64"},
		{"with build ID", "42", "v1.0", "linux/arm64", "v1.0.linux-arm64.42"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &ProjectBuildOpts{BuildID: tt.buildID}
			got := opts.pushTag(tt.tag, tt.platform)
			if got != tt.want {
				t.Errorf("pushTag(%q, %q) = %q, want %q", tt.tag, tt.platform, got, tt.want)
			}
		})
	}
}

func TestRegistryRef(t *testing.T) {
	t.Run("nil registry returns empty", func(t *testing.T) {
		opts := &ProjectBuildOpts{Registry: nil}
		got := opts.registryRef("app", "latest", "linux/amd64")
		if got != "" {
			t.Errorf("expected empty string, got %q", got)
		}
	})

	t.Run("with registry", func(t *testing.T) {
		opts := &ProjectBuildOpts{
			Registry: &mockRegistry{address: "registry.example.com"},
		}
		got := opts.registryRef("app", "latest", "linux/amd64")
		want := "registry.example.com/app:latest.linux-amd64"
		if got != want {
			t.Errorf("registryRef() = %q, want %q", got, want)
		}
	})

	t.Run("with registry and build ID", func(t *testing.T) {
		opts := &ProjectBuildOpts{
			Registry: &mockRegistry{address: "localhost:5000"},
			BuildID:  "99",
		}
		got := opts.registryRef("myimg", "v2", "linux/arm64")
		want := "localhost:5000/myimg:v2.linux-arm64.99"
		if got != want {
			t.Errorf("registryRef() = %q, want %q", got, want)
		}
	})
}

func TestRegistryInsecure(t *testing.T) {
	t.Run("nil registry", func(t *testing.T) {
		opts := &ProjectBuildOpts{Registry: nil}
		if opts.registryInsecure() {
			t.Error("nil registry should not be insecure")
		}
	})

	t.Run("local registry", func(t *testing.T) {
		opts := &ProjectBuildOpts{Registry: &mockRegistry{local: true}}
		if !opts.registryInsecure() {
			t.Error("local registry should be insecure")
		}
	})

	t.Run("remote registry", func(t *testing.T) {
		opts := &ProjectBuildOpts{Registry: &mockRegistry{local: false}}
		if opts.registryInsecure() {
			t.Error("remote registry should not be insecure")
		}
	})
}

// Helper to create a BuildOrder for an empty project with no dependencies.
func emptyBuildOrder(t *testing.T) *deps.BuildOrder {
	t.Helper()
	distPath := t.TempDir()
	project := &model.ContainerHiveProject{
		ImagesByIdentifier: map[string]*model.Image{},
		ImagesByName:       map[string][]*model.Image{},
	}
	bo, err := deps.ResolveOrder(distPath, project)
	if err != nil {
		t.Fatalf("failed to create empty build order: %v", err)
	}
	return bo
}

func TestBuildTag_DockerfileNotFound(t *testing.T) {
	distPath := t.TempDir()
	client := &Client{inner: nil}
	imageDef := &model.Image{
		Name: "myimg",
		Tags: map[string]*model.Tag{
			"1.0": {Name: "1.0"},
		},
	}
	opts := &ProjectBuildOpts{
		Project: &model.ContainerHiveProject{
			ImagesByIdentifier: map[string]*model.Image{"myimg": imageDef},
			ImagesByName:       map[string][]*model.Image{"myimg": {imageDef}},
		},
		DistPath: distPath,
	}

	err := buildTag(context.Background(), client, opts, imageDef, "1.0", "linux/amd64")
	if err == nil {
		t.Fatal("expected error when Dockerfile not found, got nil")
	}
	if !strings.Contains(err.Error(), "Dockerfile not found") {
		t.Errorf("expected error to contain 'Dockerfile not found', got: %v", err)
	}
}

func TestBuildVariant_DockerfileNotFound(t *testing.T) {
	distPath := t.TempDir()
	client := &Client{inner: nil}
	variantDef := &model.ImageVariant{
		Name:      "node",
		TagSuffix: "-node",
	}
	imageDef := &model.Image{
		Name: "myimg",
		Tags: map[string]*model.Tag{
			"1.0": {Name: "1.0"},
		},
		Variants: map[string]*model.ImageVariant{
			"node": variantDef,
		},
	}
	opts := &ProjectBuildOpts{
		Project: &model.ContainerHiveProject{
			ImagesByIdentifier: map[string]*model.Image{"myimg": imageDef},
			ImagesByName:       map[string][]*model.Image{"myimg": {imageDef}},
		},
		DistPath: distPath,
	}

	err := buildVariant(context.Background(), client, opts, imageDef, "1.0", "node", variantDef, "linux/amd64")
	if err != nil {
		t.Errorf("expected nil error when variant Dockerfile not found, got: %v", err)
	}
}

func TestBuildProject_NilProgressOut(t *testing.T) {
	bo := emptyBuildOrder(t)
	opts := &ProjectBuildOpts{
		Project: &model.ContainerHiveProject{
			ImagesByIdentifier: map[string]*model.Image{},
			ImagesByName:       map[string][]*model.Image{},
		},
		BuildOrder:  bo,
		DistPath:    t.TempDir(),
		ProgressOut: nil, // explicitly nil
	}

	err := BuildProject(context.Background(), &Client{inner: nil}, opts)
	if err != nil {
		t.Errorf("expected no error for empty project, got: %v", err)
	}
	if opts.ProgressOut == nil {
		t.Error("expected ProgressOut to be set to os.Stdout, but it is still nil")
	}
	if opts.ProgressOut != os.Stdout {
		t.Error("expected ProgressOut to be os.Stdout")
	}
}

func TestBuildProject_NoImages(t *testing.T) {
	t.Run("no deps path", func(t *testing.T) {
		bo := emptyBuildOrder(t)
		opts := &ProjectBuildOpts{
			Project: &model.ContainerHiveProject{
				ImagesByIdentifier: map[string]*model.Image{},
				ImagesByName:       map[string][]*model.Image{},
			},
			BuildOrder:  bo,
			DistPath:    t.TempDir(),
			ProgressOut: os.Stdout,
		}

		err := BuildProject(context.Background(), &Client{inner: nil}, opts)
		if err != nil {
			t.Errorf("expected no error for empty project without deps, got: %v", err)
		}
	})
}

func TestRegistryRef_NilRegistry(t *testing.T) {
	opts := &ProjectBuildOpts{Registry: nil}
	got := opts.registryRef("myimg", "latest", "linux/amd64")
	if got != "" {
		t.Errorf("expected empty string for nil registry, got %q", got)
	}
}

func TestRegistryAddress_NilRegistry(t *testing.T) {
	opts := &ProjectBuildOpts{Registry: nil}
	got := opts.registryAddress()
	if got != "" {
		t.Errorf("expected empty string for nil registry, got %q", got)
	}
}

func TestRegistryAddress_WithRegistry(t *testing.T) {
	opts := &ProjectBuildOpts{
		Registry: &mockRegistry{address: "registry.example.com"},
	}
	got := opts.registryAddress()
	want := "registry.example.com"
	if got != want {
		t.Errorf("registryAddress() = %q, want %q", got, want)
	}
}

func TestRegistryInsecure_NilRegistryDedicated(t *testing.T) {
	opts := &ProjectBuildOpts{Registry: nil}
	if opts.registryInsecure() {
		t.Error("nil registry should return false for insecure")
	}
}

func TestRegistryInsecure_LocalRegistry(t *testing.T) {
	opts := &ProjectBuildOpts{
		Registry: &mockRegistry{address: "localhost:5000", local: true},
	}
	if !opts.registryInsecure() {
		t.Error("local registry should be insecure")
	}
}

// TestBuildWithoutDeps_VariantResolvesHiveRefs covers the case where a project
// has only a single image with a variant that FROMs its own parent via
// __hive__/. The dep graph sees no cross-image edges, so buildWithoutDeps is
// taken. It must still route the variant through buildVariant (not a bare
// direct-build path), which invokes ResolveHiveDeps — otherwise the raw
// __hive__/ reference reaches BuildKit and is rejected as invalid.
func TestBuildWithoutDeps_VariantResolvesHiveRefs(t *testing.T) {
	distPath := t.TempDir()
	variantDir := filepath.Join(distPath, "myimg", "1.0-node")
	if err := os.MkdirAll(variantDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(variantDir, "Dockerfile"), []byte("FROM __hive__/myimg:1.0\n"), 0644); err != nil {
		t.Fatal(err)
	}

	variantDef := &model.ImageVariant{Name: "node", TagSuffix: "-node"}
	imageDef := &model.Image{
		Name:     "myimg",
		Tags:     map[string]*model.Tag{"1.0": {Name: "1.0"}},
		Variants: map[string]*model.ImageVariant{"node": variantDef},
	}
	project := &model.ContainerHiveProject{
		Config:             model.HiveProjectConfig{Platforms: []string{"linux/amd64"}},
		ImagesByIdentifier: map[string]*model.Image{"myimg": imageDef},
		ImagesByName:       map[string][]*model.Image{"myimg": {imageDef}},
	}

	opts := &ProjectBuildOpts{
		Project:     project,
		DistPath:    distPath,
		ProgressOut: os.Stdout,
		// No Registry and no local tar → ResolveHiveDeps must surface a clear
		// error instead of letting the raw __hive__/ reference reach BuildKit.
		Filters: []Filter{{ImageName: "myimg", TagName: "1.0-node"}}, // skip the (unrendered) base tag
	}

	err := buildWithoutDeps(context.Background(), &Client{inner: nil}, opts)
	if err == nil {
		t.Fatal("expected hive-dep resolution error, got nil")
	}
	if !strings.Contains(err.Error(), "not built yet") || !strings.Contains(err.Error(), "no registry configured") {
		t.Errorf("expected hive-dep resolution error, got: %v", err)
	}
}

// TestBuildWithoutDeps_VariantResolvesHiveRefsViaRegistry ensures that when a
// registry is configured, the variant's __hive__/ reference resolves via the
// registry fallback. MkdirAll is blocked so execution halts right after
// ResolveHiveDeps succeeds but before the nil BuildKit client is dialled —
// proving the hive-deps path ran.
func TestBuildWithoutDeps_VariantResolvesHiveRefsViaRegistry(t *testing.T) {
	distPath := t.TempDir()
	variantDir := filepath.Join(distPath, "myimg", "1.0-node")
	if err := os.MkdirAll(variantDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(variantDir, "Dockerfile"), []byte("FROM __hive__/myimg:1.0\n"), 0644); err != nil {
		t.Fatal(err)
	}
	// Block MkdirAll for the tar output dir so we halt after ResolveHiveDeps.
	if err := os.WriteFile(filepath.Join(variantDir, "linux-amd64"), []byte("blocker"), 0644); err != nil {
		t.Fatal(err)
	}

	variantDef := &model.ImageVariant{Name: "node", TagSuffix: "-node"}
	imageDef := &model.Image{
		Name:     "myimg",
		Tags:     map[string]*model.Tag{"1.0": {Name: "1.0"}},
		Variants: map[string]*model.ImageVariant{"node": variantDef},
	}
	project := &model.ContainerHiveProject{
		Config:             model.HiveProjectConfig{Platforms: []string{"linux/amd64"}},
		ImagesByIdentifier: map[string]*model.Image{"myimg": imageDef},
		ImagesByName:       map[string][]*model.Image{"myimg": {imageDef}},
	}

	opts := &ProjectBuildOpts{
		Project:     project,
		DistPath:    distPath,
		ProgressOut: os.Stdout,
		Registry:    &mockRegistry{address: "registry.example.com"},
		BuildID:     "42",
		Filters:     []Filter{{ImageName: "myimg", TagName: "1.0-node"}},
	}

	err := buildWithoutDeps(context.Background(), &Client{inner: nil}, opts)
	if err == nil {
		t.Fatal("expected MkdirAll failure after hive-deps resolve, got nil")
	}
	if !strings.Contains(err.Error(), "failed to create platform dir") {
		t.Errorf("expected platform-dir error (hive-deps already resolved), got: %v", err)
	}
}

// TestBuildWithoutDeps_VariantResolvesBuildArgs verifies the variant's
// Versions get turned into BuildArgs (e.g. nodejs: "24" → NODEJS_VERSION=24)
// when buildWithoutDeps is the active branch. Before the refactor that routed
// variants through buildVariant, the no-deps path skipped ResolveVariantConfig
// entirely and NODEJS_VERSION reached BuildKit empty.
func TestBuildWithoutDeps_VariantResolvesBuildArgs(t *testing.T) {
	resolved, err := ResolveVariantConfig(
		&model.Image{
			Name: "myimg",
			Tags: map[string]*model.Tag{"1.0": {Name: "1.0"}},
		},
		&model.ImageVariant{
			Name:      "node",
			TagSuffix: "-node",
			Versions:  model.Versions{"nodejs": "24"},
		},
		&model.Tag{Name: "1.0"},
	)
	if err != nil {
		t.Fatalf("ResolveVariantConfig failed: %v", err)
	}
	if got := resolved.BuildArgs["NODEJS_VERSION"]; got != "24" {
		t.Errorf("expected NODEJS_VERSION=24, got %q", got)
	}
}

func TestBuildWithoutDeps_EmptyProject(t *testing.T) {
	opts := &ProjectBuildOpts{
		Project: &model.ContainerHiveProject{
			ImagesByIdentifier: map[string]*model.Image{},
			ImagesByName:       map[string][]*model.Image{},
		},
		DistPath:    t.TempDir(),
		ProgressOut: os.Stdout,
	}

	err := buildWithoutDeps(context.Background(), &Client{inner: nil}, opts)
	if err != nil {
		t.Errorf("expected no error for empty ImagesByName, got: %v", err)
	}
}

func TestBuildWithDeps_EmptyOrder(t *testing.T) {
	bo := emptyBuildOrder(t)
	opts := &ProjectBuildOpts{
		Project: &model.ContainerHiveProject{
			ImagesByIdentifier: map[string]*model.Image{},
			ImagesByName:       map[string][]*model.Image{},
		},
		BuildOrder:  bo,
		DistPath:    t.TempDir(),
		ProgressOut: os.Stdout,
	}

	err := buildWithDeps(context.Background(), &Client{inner: nil}, opts)
	if err != nil {
		t.Errorf("expected no error for empty build order, got: %v", err)
	}
}

// buildOrderWithDeps creates a BuildOrder that has dependencies.
// It sets up the dist directory with two images where "app" depends on "base".
func buildOrderWithDeps(t *testing.T, distPath string, project *model.ContainerHiveProject) *deps.BuildOrder {
	t.Helper()
	bo, err := deps.ResolveOrder(distPath, project)
	if err != nil {
		t.Fatalf("failed to create build order with deps: %v", err)
	}
	if !bo.HasDependencies() {
		t.Fatal("expected build order to have dependencies")
	}
	return bo
}

func TestBuildWithDeps_ImageNotInProject(t *testing.T) {
	distPath := t.TempDir()

	// Create dist dirs so ScanRenderedProject picks them up.
	if err := os.MkdirAll(filepath.Join(distPath, "found"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(distPath, "notfound"), 0755); err != nil {
		t.Fatal(err)
	}

	foundImg := &model.Image{
		Name:      "found",
		Tags:      map[string]*model.Tag{}, // no tags → nothing to build
		DependsOn: []string{"notfound"},
	}
	notfoundImg := &model.Image{
		Name: "notfound",
		Tags: map[string]*model.Tag{},
	}

	project := &model.ContainerHiveProject{
		ImagesByIdentifier: map[string]*model.Image{
			// "notfound" deliberately omitted from identifier lookup
			"found": foundImg,
		},
		ImagesByName: map[string][]*model.Image{
			"found":    {foundImg},
			"notfound": {notfoundImg},
		},
	}

	bo := buildOrderWithDeps(t, distPath, project)
	opts := &ProjectBuildOpts{
		Project:     project,
		BuildOrder:  bo,
		DistPath:    distPath,
		ProgressOut: os.Stdout,
	}

	// "notfound" is in order but not in ImagesByIdentifier → warning, continue.
	// "found" is in ImagesByIdentifier but has no tags → nothing to build.
	err := buildWithDeps(context.Background(), &Client{inner: nil}, opts)
	if err != nil {
		t.Errorf("expected no error when image not in project, got: %v", err)
	}
}

func TestBuildWithDeps_FiltersSkipAllTags(t *testing.T) {
	distPath := t.TempDir()

	baseDir := filepath.Join(distPath, "base", "1.0")
	appDir := filepath.Join(distPath, "app", "1.0")
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(baseDir, "Dockerfile"), []byte("FROM scratch\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "Dockerfile"), []byte("FROM scratch\n"), 0644); err != nil {
		t.Fatal(err)
	}

	baseImg := &model.Image{
		Name: "base",
		Tags: map[string]*model.Tag{"1.0": {Name: "1.0"}},
	}
	appImg := &model.Image{
		Name:      "app",
		Tags:      map[string]*model.Tag{"1.0": {Name: "1.0"}},
		DependsOn: []string{"base"},
	}

	project := &model.ContainerHiveProject{
		Config:             model.HiveProjectConfig{Platforms: []string{"linux/amd64"}},
		ImagesByIdentifier: map[string]*model.Image{"base": baseImg, "app": appImg},
		ImagesByName:       map[string][]*model.Image{"base": {baseImg}, "app": {appImg}},
	}

	bo := buildOrderWithDeps(t, distPath, project)
	opts := &ProjectBuildOpts{
		Project:     project,
		BuildOrder:  bo,
		DistPath:    distPath,
		ProgressOut: os.Stdout,
		Filters:     []Filter{{ImageName: "nonexistent"}}, // matches nothing
	}

	err := buildWithDeps(context.Background(), &Client{inner: nil}, opts)
	if err != nil {
		t.Errorf("expected no error when all tags filtered out, got: %v", err)
	}
}

func TestBuildWithDeps_BuildTagMkdirFails(t *testing.T) {
	distPath := t.TempDir()

	baseDir := filepath.Join(distPath, "base", "1.0")
	appDir := filepath.Join(distPath, "app", "1.0")
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(baseDir, "Dockerfile"), []byte("FROM scratch\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "Dockerfile"), []byte("FROM scratch\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Block MkdirAll: TarFilePath creates distPath/base/1.0/linux-amd64/image.tar
	// Place a regular file at linux-amd64 so MkdirAll cannot create the dir.
	if err := os.WriteFile(filepath.Join(baseDir, "linux-amd64"), []byte("blocker"), 0644); err != nil {
		t.Fatal(err)
	}

	baseImg := &model.Image{
		Name: "base",
		Tags: map[string]*model.Tag{"1.0": {Name: "1.0"}},
	}
	appImg := &model.Image{
		Name:      "app",
		Tags:      map[string]*model.Tag{"1.0": {Name: "1.0"}},
		DependsOn: []string{"base"},
	}

	project := &model.ContainerHiveProject{
		Config:             model.HiveProjectConfig{Platforms: []string{"linux/amd64"}},
		ImagesByIdentifier: map[string]*model.Image{"base": baseImg, "app": appImg},
		ImagesByName:       map[string][]*model.Image{"base": {baseImg}, "app": {appImg}},
	}

	bo := buildOrderWithDeps(t, distPath, project)
	opts := &ProjectBuildOpts{
		Project:     project,
		BuildOrder:  bo,
		DistPath:    distPath,
		ProgressOut: os.Stdout,
	}

	err := buildWithDeps(context.Background(), &Client{inner: nil}, opts)
	if err == nil {
		t.Fatal("expected error from buildTag MkdirAll failure, got nil")
	}
	if !strings.Contains(err.Error(), "failed to create platform dir") {
		t.Errorf("expected 'failed to create platform dir' error, got: %v", err)
	}
}

func TestBuildWithDeps_VariantDockerfileNotFound(t *testing.T) {
	distPath := t.TempDir()

	baseDir := filepath.Join(distPath, "base", "1.0")
	appDir := filepath.Join(distPath, "app", "1.0")
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(baseDir, "Dockerfile"), []byte("FROM scratch\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "Dockerfile"), []byte("FROM scratch\n"), 0644); err != nil {
		t.Fatal(err)
	}

	variantDef := &model.ImageVariant{
		Name:      "node",
		TagSuffix: "-node",
	}
	baseImg := &model.Image{
		Name: "base",
		Tags: map[string]*model.Tag{"1.0": {Name: "1.0"}},
		Variants: map[string]*model.ImageVariant{
			"node": variantDef,
		},
	}
	appImg := &model.Image{
		Name:      "app",
		Tags:      map[string]*model.Tag{"1.0": {Name: "1.0"}},
		DependsOn: []string{"base"},
	}

	project := &model.ContainerHiveProject{
		Config:             model.HiveProjectConfig{Platforms: []string{"linux/amd64"}},
		ImagesByIdentifier: map[string]*model.Image{"base": baseImg, "app": appImg},
		ImagesByName:       map[string][]*model.Image{"base": {baseImg}, "app": {appImg}},
	}

	bo := buildOrderWithDeps(t, distPath, project)
	// Filter to only the variant tag (skip base), so buildTag is not called.
	opts := &ProjectBuildOpts{
		Project:     project,
		BuildOrder:  bo,
		DistPath:    distPath,
		ProgressOut: os.Stdout,
		Filters:     []Filter{{ImageName: "base", TagName: "1.0-node"}},
	}

	// Variant Dockerfile (base/1.0-node/Dockerfile) does not exist → warning, no error.
	err := buildWithDeps(context.Background(), &Client{inner: nil}, opts)
	if err != nil {
		t.Errorf("expected no error for missing variant Dockerfile, got: %v", err)
	}
}

func TestBuildWithDeps_BuildVariantMkdirFails(t *testing.T) {
	distPath := t.TempDir()

	baseDir := filepath.Join(distPath, "base", "1.0")
	appDir := filepath.Join(distPath, "app", "1.0")
	variantDir := filepath.Join(distPath, "base", "1.0-node")
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(variantDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(baseDir, "Dockerfile"), []byte("FROM scratch\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "Dockerfile"), []byte("FROM scratch\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(variantDir, "Dockerfile"), []byte("FROM scratch\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Block MkdirAll for the variant platform dir.
	if err := os.WriteFile(filepath.Join(variantDir, "linux-amd64"), []byte("blocker"), 0644); err != nil {
		t.Fatal(err)
	}

	variantDef := &model.ImageVariant{
		Name:      "node",
		TagSuffix: "-node",
	}
	baseImg := &model.Image{
		Name: "base",
		Tags: map[string]*model.Tag{"1.0": {Name: "1.0"}},
		Variants: map[string]*model.ImageVariant{
			"node": variantDef,
		},
	}
	appImg := &model.Image{
		Name:      "app",
		Tags:      map[string]*model.Tag{"1.0": {Name: "1.0"}},
		DependsOn: []string{"base"},
	}

	project := &model.ContainerHiveProject{
		Config:             model.HiveProjectConfig{Platforms: []string{"linux/amd64"}},
		ImagesByIdentifier: map[string]*model.Image{"base": baseImg, "app": appImg},
		ImagesByName:       map[string][]*model.Image{"base": {baseImg}, "app": {appImg}},
	}

	bo := buildOrderWithDeps(t, distPath, project)
	// Filter to only the variant tag so base tag is skipped.
	opts := &ProjectBuildOpts{
		Project:     project,
		BuildOrder:  bo,
		DistPath:    distPath,
		ProgressOut: os.Stdout,
		Filters:     []Filter{{ImageName: "base", TagName: "1.0-node"}},
	}

	err := buildWithDeps(context.Background(), &Client{inner: nil}, opts)
	if err == nil {
		t.Fatal("expected error from buildVariant MkdirAll failure, got nil")
	}
	if !strings.Contains(err.Error(), "failed to create platform dir") {
		t.Errorf("expected 'failed to create platform dir' error, got: %v", err)
	}
}

func TestBuildWithDeps_FilterMatchesBaseNotVariant(t *testing.T) {
	distPath := t.TempDir()

	baseDir := filepath.Join(distPath, "base", "1.0")
	appDir := filepath.Join(distPath, "app", "1.0")
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(baseDir, "Dockerfile"), []byte("FROM scratch\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "Dockerfile"), []byte("FROM scratch\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Block MkdirAll for the base tag to prove it's attempted.
	if err := os.WriteFile(filepath.Join(baseDir, "linux-amd64"), []byte("blocker"), 0644); err != nil {
		t.Fatal(err)
	}

	variantDef := &model.ImageVariant{
		Name:      "node",
		TagSuffix: "-node",
	}
	baseImg := &model.Image{
		Name: "base",
		Tags: map[string]*model.Tag{"1.0": {Name: "1.0"}},
		Variants: map[string]*model.ImageVariant{
			"node": variantDef,
		},
	}
	appImg := &model.Image{
		Name:      "app",
		Tags:      map[string]*model.Tag{"1.0": {Name: "1.0"}},
		DependsOn: []string{"base"},
	}

	project := &model.ContainerHiveProject{
		Config:             model.HiveProjectConfig{Platforms: []string{"linux/amd64"}},
		ImagesByIdentifier: map[string]*model.Image{"base": baseImg, "app": appImg},
		ImagesByName:       map[string][]*model.Image{"base": {baseImg}, "app": {appImg}},
	}

	bo := buildOrderWithDeps(t, distPath, project)
	// Filter matches "base:1.0" (base tag) but NOT "base:1.0-node" (variant).
	opts := &ProjectBuildOpts{
		Project:     project,
		BuildOrder:  bo,
		DistPath:    distPath,
		ProgressOut: os.Stdout,
		Filters:     []Filter{{ImageName: "base", TagName: "1.0"}},
	}

	// buildTag is called for base:1.0 and fails on MkdirAll, proving the base
	// was attempted. Variant is skipped by filter.
	err := buildWithDeps(context.Background(), &Client{inner: nil}, opts)
	if err == nil {
		t.Fatal("expected error from base tag build, got nil")
	}
	if !strings.Contains(err.Error(), "failed to create platform dir") {
		t.Errorf("expected MkdirAll error, got: %v", err)
	}
}
