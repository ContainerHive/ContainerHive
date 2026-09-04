package build

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ContainerHive/ContainerHive/pkg/model"
	"github.com/ContainerHive/ContainerHive/pkg/shard"
)

// captureSlog swaps in a text handler writing to a buffer for the duration
// of the test, restoring the previous default logger on cleanup.
func captureSlog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })
	return &buf
}

func singleImageProject(withVariant bool) *model.ContainerHiveProject {
	imageDef := &model.Image{
		Identifier: "myimg",
		Name:       "myimg",
		Tags:       map[string]*model.Tag{"1.0": {Name: "1.0"}},
	}
	if withVariant {
		imageDef.Variants = map[string]*model.ImageVariant{"node": {Name: "node", TagSuffix: "-node"}}
	}
	return &model.ContainerHiveProject{
		Config:             model.HiveProjectConfig{Platforms: []string{"linux/amd64"}},
		ImagesByIdentifier: map[string]*model.Image{"myimg": imageDef},
		ImagesByName:       map[string][]*model.Image{"myimg": {imageDef}},
	}
}

// TestBuildWithoutDeps_ShardSkipsUnownedBaseTag proves the base-tag gate is
// actually consulted: with ownsBase stubbed to always deny, no build is
// attempted at all, so there is no "Dockerfile not found" error - the tag
// was skipped, not attempted and swallowed.
func TestBuildWithoutDeps_ShardSkipsUnownedBaseTag(t *testing.T) {
	opts := &ProjectBuildOpts{
		Project:     singleImageProject(false),
		DistPath:    t.TempDir(),
		ProgressOut: os.Stdout,
		baseOwns:    func(string, string) bool { return false },
	}

	if err := buildWithoutDeps(context.Background(), &Client{inner: nil}, opts); err != nil {
		t.Errorf("expected the unowned base tag to be skipped silently, got: %v", err)
	}
}

// TestBuildWithoutDeps_ShardBuildsOwnedBaseTag is the inverse: ownsBase
// stubbed to always allow must reach the actual build attempt, which fails
// on the missing Dockerfile - proving the tag was not skipped.
func TestBuildWithoutDeps_ShardBuildsOwnedBaseTag(t *testing.T) {
	opts := &ProjectBuildOpts{
		Project:     singleImageProject(false),
		DistPath:    t.TempDir(),
		ProgressOut: os.Stdout,
		baseOwns:    func(string, string) bool { return true },
	}

	err := buildWithoutDeps(context.Background(), &Client{inner: nil}, opts)
	if err == nil || !strings.Contains(err.Error(), "Dockerfile not found") {
		t.Errorf("expected a Dockerfile-not-found error proving the base tag was attempted, got: %v", err)
	}
}

// TestBuildWithoutDeps_BaseGateReceivesIdentifierAndTagName confirms the base
// gate is called with the image's Identifier (not Name) and the exact base
// tag name - the same key pkg/shard indexes on.
func TestBuildWithoutDeps_BaseGateReceivesIdentifierAndTagName(t *testing.T) {
	var gotIdentifier, gotTag string
	opts := &ProjectBuildOpts{
		Project:     singleImageProject(false),
		DistPath:    t.TempDir(),
		ProgressOut: os.Stdout,
		baseOwns: func(identifier, tagName string) bool {
			gotIdentifier, gotTag = identifier, tagName
			return false
		},
	}

	if err := buildWithoutDeps(context.Background(), &Client{inner: nil}, opts); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotIdentifier != "myimg" || gotTag != "1.0" {
		t.Errorf("baseOwns called with (%q, %q), want (%q, %q)", gotIdentifier, gotTag, "myimg", "1.0")
	}
}

// TestBuildWithoutDeps_ShardSkipsUnownedVariantTag mirrors the base-tag skip
// test for the variant gate. The base tag is also denied so only the variant
// path is under test.
func TestBuildWithoutDeps_ShardSkipsUnownedVariantTag(t *testing.T) {
	opts := &ProjectBuildOpts{
		Project:     singleImageProject(true),
		DistPath:    t.TempDir(),
		ProgressOut: os.Stdout,
		baseOwns:    func(string, string) bool { return false },
		variantOwns: func(string, string) bool { return false },
	}

	if err := buildWithoutDeps(context.Background(), &Client{inner: nil}, opts); err != nil {
		t.Errorf("expected the unowned variant tag to be skipped silently, got: %v", err)
	}
}

// TestBuildWithoutDeps_ShardBuildsBaseForOwnedVariant is the key invariant of
// NewBaseTagSharder's wiring: a shard that owns only the variant, not the
// base tag by exact match, must still attempt to build the base - because
// the variant's Dockerfile is FROM that base tag locally. Denying via
// baseOwns here would be wrong if the wiring didn't route base-tag ownership
// through the base gate independently of variant ownership; this test
// exercises the gates as pkg/shard.NewBaseTagSharder actually composes them.
func TestBuildWithoutDeps_ShardBuildsBaseForOwnedVariant(t *testing.T) {
	project := singleImageProject(true)
	// A 2-shard split where shard 2 owns the variant but not the base by
	// exact match, to prove ownsBase still reports true for the base
	// because of the owned variant.
	twoShards := shard.Shard{Current: 2, Max: 2}
	baseOwns := shard.NewBaseTagSharder(project, twoShards)
	exactOwns := shard.NewTagSharder(project, twoShards)

	if exactOwns("myimg", "1.0") {
		t.Fatal("test setup invalid: shard 2 should not own the base tag by exact match")
	}
	if !exactOwns("myimg", "1.0-node") {
		t.Fatal("test setup invalid: shard 2 should own the variant tag")
	}
	if !baseOwns("myimg", "1.0") {
		t.Fatal("test setup invalid: NewBaseTagSharder should report the base as owned via the variant")
	}

	opts := &ProjectBuildOpts{
		Project:     project,
		DistPath:    t.TempDir(),
		ProgressOut: os.Stdout,
		baseOwns:    baseOwns,
		variantOwns: exactOwns,
	}

	err := buildWithoutDeps(context.Background(), &Client{inner: nil}, opts)
	if err == nil || !strings.Contains(err.Error(), "Dockerfile not found") || !strings.Contains(err.Error(), "1.0") {
		t.Errorf("expected the base tag to be attempted (Dockerfile not found for 1.0), got: %v", err)
	}
}

// TestBuildProject_EmptyShardWarnsAndSucceeds is the "nothing to do" case: a
// shard configured beyond the number of available units must warn and
// return nil rather than silently doing nothing or erroring.
func TestBuildProject_EmptyShardWarnsAndSucceeds(t *testing.T) {
	buf := captureSlog(t)

	opts := &ProjectBuildOpts{
		Project:     singleImageProject(false),
		DistPath:    t.TempDir(),
		ProgressOut: os.Stdout,
		BuildOrder:  emptyBuildOrder(t),
		Shard:       shard.Shard{Current: 5, Max: 5}, // only 1 unit exists; shard 5 owns nothing
	}

	if err := BuildProject(context.Background(), &Client{inner: nil}, opts); err != nil {
		t.Fatalf("expected an empty shard to succeed with a warning, got: %v", err)
	}
	if !strings.Contains(buf.String(), "Nothing to do for this shard") {
		t.Errorf("expected a \"nothing to do\" warning, got log output: %s", buf.String())
	}
}

// TestBuildProject_ShardWithDependenciesWarns proves the sharding-plus-deps
// warning fires when both are true, so operators see it in the job log
// rather than silently relying on the registry fallback.
func TestBuildProject_ShardWithDependenciesWarns(t *testing.T) {
	buf := captureSlog(t)

	distPath := t.TempDir()
	if err := os.MkdirAll(filepath.Join(distPath, "app"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(distPath, "base"), 0755); err != nil {
		t.Fatal(err)
	}

	appImg := &model.Image{Identifier: "app", Name: "app", Tags: map[string]*model.Tag{}, DependsOn: []string{"base"}}
	baseImg := &model.Image{Identifier: "base", Name: "base", Tags: map[string]*model.Tag{"1.0": {Name: "1.0"}}}
	project := &model.ContainerHiveProject{
		Config: model.HiveProjectConfig{Platforms: []string{"linux/amd64"}},
		ImagesByIdentifier: map[string]*model.Image{
			"app": appImg, "base": baseImg,
		},
		ImagesByName: map[string][]*model.Image{
			"app": {appImg}, "base": {baseImg},
		},
	}
	bo := buildOrderWithDeps(t, distPath, project)

	opts := &ProjectBuildOpts{
		Project:     project,
		DistPath:    distPath,
		ProgressOut: os.Stdout,
		BuildOrder:  bo,
		Shard:       shard.Shard{Current: 1, Max: 2},
	}

	// The build itself will fail fast (missing Dockerfile) - only the
	// warning is under test here.
	_ = BuildProject(context.Background(), &Client{inner: nil}, opts)

	if !strings.Contains(buf.String(), "inter-image dependencies") {
		t.Errorf("expected a warning about sharding combined with inter-image dependencies, got: %s", buf.String())
	}
}
