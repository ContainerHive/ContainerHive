package test

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/ContainerHive/ContainerHive/pkg/model"
	"github.com/ContainerHive/ContainerHive/pkg/shard"
)

func captureSlog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })
	return &buf
}

func twoImageProject() *model.ContainerHiveProject {
	appImg := &model.Image{
		Identifier: "app",
		Name:       "app",
		Tags:       map[string]*model.Tag{"1.0": {Name: "1.0"}},
		Platforms:  []string{"linux/amd64"},
	}
	libImg := &model.Image{
		Identifier: "lib",
		Name:       "lib",
		Tags:       map[string]*model.Tag{"2.0": {Name: "2.0"}},
		Platforms:  []string{"linux/amd64"},
	}
	return &model.ContainerHiveProject{
		ImagesByIdentifier: map[string]*model.Image{"app": appImg, "lib": libImg},
		ImagesByName:       map[string][]*model.Image{"app": {appImg}, "lib": {libImg}},
		Config:             model.HiveProjectConfig{Platforms: []string{"linux/amd64"}},
	}
}

// TestRunProjectTests_EmptyShardWarnsAndSucceeds mirrors the build/sbom
// "nothing to do" behavior: a shard configured beyond the number of
// available units must warn and return zero counts with no error, rather
// than the pre-existing silent "0, 0, nil" that gave no indication why.
func TestRunProjectTests_EmptyShardWarnsAndSucceeds(t *testing.T) {
	buf := captureSlog(t)

	// Single-image project, shard 2 of 2: the one unit is owned by shard 1.
	project := &model.ContainerHiveProject{
		ImagesByIdentifier: map[string]*model.Image{
			"app": {Identifier: "app", Name: "app", Tags: map[string]*model.Tag{"1.0": {Name: "1.0"}}, Platforms: []string{"linux/amd64"}},
		},
		Config: model.HiveProjectConfig{Platforms: []string{"linux/amd64"}},
	}

	tested, failed, err := RunProjectTests(context.Background(), &Opts{
		DistPath: t.TempDir(),
		Project:  project,
		Shard:    shard.Shard{Current: 2, Max: 2},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tested != 0 || failed != 0 {
		t.Errorf("expected 0 tested and 0 failed, got tested=%d failed=%d", tested, failed)
	}
	if !strings.Contains(buf.String(), "Nothing to do for this shard") {
		t.Errorf("expected a \"nothing to do\" warning, got log output: %s", buf.String())
	}
}

// TestRunProjectTests_ShardSkipsUnownedImage proves the exact-tag gate in
// RunProjectTests, not just OwnedUnits, actually excludes a unit this shard
// does not own. runTestsForTag logs "No test definitions, skipping" as soon
// as it is entered for a tag, before touching the filesystem beyond
// CollectTestDefinitions - so that log line appears once per owned unit and
// not at all for a skipped one, distinguishing "ran and found nothing" from
// "never ran".
func TestRunProjectTests_ShardSkipsUnownedImage(t *testing.T) {
	project := twoImageProject()
	all := shard.TagIndex(project)
	if len(all) != 2 {
		t.Fatalf("test setup invalid: expected 2 units, got %d", len(all))
	}

	for current := 1; current <= 2; current++ {
		buf := captureSlog(t)
		s := shard.Shard{Current: current, Max: 2}
		owned := shard.OwnedUnits(project, s)
		if len(owned) != 1 {
			t.Fatalf("test setup invalid: shard %d should own exactly 1 unit, got %d", current, len(owned))
		}
		ownedImage := owned[0].Identifier

		if _, _, err := RunProjectTests(context.Background(), &Opts{
			DistPath: t.TempDir(),
			Project:  project,
			Shard:    s,
		}); err != nil {
			t.Fatalf("shard %d: unexpected error: %v", current, err)
		}

		for _, ref := range all {
			entered := strings.Contains(buf.String(), `image=`+ref.Identifier)
			if ref.Identifier == ownedImage && !entered {
				t.Errorf("shard %d: expected runTestsForTag to be entered for owned image %q, but no log line mentioned it: %s", current, ownedImage, buf.String())
			}
			if ref.Identifier != ownedImage && entered {
				t.Errorf("shard %d: image %q is not owned by this shard but was still processed: %s", current, ref.Identifier, buf.String())
			}
		}
	}
}

// TestRunProjectTests_DisabledShardBehavesAsBefore locks in that a
// zero-valued Shard (the default for every existing caller) is equivalent
// to no sharding at all - Opts{} without a Shard field must not change
// pre-existing behavior.
func TestRunProjectTests_DisabledShardBehavesAsBefore(t *testing.T) {
	project := twoImageProject()

	tested, failed, err := RunProjectTests(context.Background(), &Opts{
		DistPath: t.TempDir(),
		Project:  project,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tested != 0 || failed != 0 {
		t.Errorf("expected 0/0, got tested=%d failed=%d", tested, failed)
	}
}
