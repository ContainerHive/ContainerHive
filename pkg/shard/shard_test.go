package shard

import (
	"testing"

	"github.com/ContainerHive/ContainerHive/pkg/model"
)

// fixtureProject builds a project with:
//   - "alpha" (2 identifiers sharing the name "alpha", like variant subfolders):
//     alpha-a has tags "1.0", "2.0" with a "-node" variant.
//     alpha-b has tag "1.0" with no variants.
//   - "beta" has a single tag "1.0" with no variants.
func fixtureProject() *model.ContainerHiveProject {
	alphaA := &model.Image{
		Identifier: "alpha-a",
		Name:       "alpha",
		Tags: map[string]*model.Tag{
			"1.0": {Name: "1.0"},
			"2.0": {Name: "2.0"},
		},
		Variants: map[string]*model.ImageVariant{
			"node": {Name: "node", TagSuffix: "-node"},
		},
	}
	alphaB := &model.Image{
		Identifier: "alpha-b",
		Name:       "alpha",
		Tags: map[string]*model.Tag{
			"1.0": {Name: "1.0"},
		},
	}
	beta := &model.Image{
		Identifier: "beta",
		Name:       "beta",
		Tags: map[string]*model.Tag{
			"1.0": {Name: "1.0"},
		},
	}
	return &model.ContainerHiveProject{
		ImagesByIdentifier: map[string]*model.Image{
			"alpha-a": alphaA,
			"alpha-b": alphaB,
			"beta":    beta,
		},
		ImagesByName: map[string][]*model.Image{
			"alpha": {alphaA, alphaB},
			"beta":  {beta},
		},
	}
}

func TestShardOwns(t *testing.T) {
	tests := []struct {
		name    string
		max     int
		current int
		index   int
		want    bool
	}{
		{"disabled matches everything", 1, 1, 0, true},
		{"disabled matches everything, large index", 1, 1, 99, true},
		{"3 shards, index 0 -> shard 1", 3, 1, 0, true},
		{"3 shards, index 0 -> not shard 2", 3, 2, 0, false},
		{"3 shards, index 3 -> shard 1 again", 3, 1, 3, true},
		{"odd count: 7 items, 3 shards, index 6 -> shard 1", 3, 1, 6, true},
		{"odd count: 7 items, 3 shards, index 6 -> not shard 3", 3, 3, 6, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Shard{Current: tt.current, Max: tt.max}
			if got := s.Owns(tt.index); got != tt.want {
				t.Errorf("Owns(%d) = %v, want %v", tt.index, got, tt.want)
			}
		})
	}
}

func TestShardOwnsPartitionsExactly(t *testing.T) {
	// Property: for any Max, every index is owned by exactly one shard.
	for _, max := range []int{1, 2, 3, 5, 7} {
		for index := 0; index < 20; index++ {
			owners := 0
			for current := 1; current <= max; current++ {
				s := Shard{Current: current, Max: max}
				if s.Owns(index) {
					owners++
				}
			}
			if owners != 1 {
				t.Errorf("max=%d index=%d: owned by %d shards, want exactly 1", max, index, owners)
			}
		}
	}
}

func TestShardValidate(t *testing.T) {
	tests := []struct {
		name    string
		s       Shard
		wantErr bool
	}{
		{"valid", Shard{Current: 1, Max: 1}, false},
		{"valid middle", Shard{Current: 2, Max: 3}, false},
		{"max less than 1", Shard{Current: 1, Max: 0}, true},
		{"current less than 1", Shard{Current: 0, Max: 3}, true},
		{"current greater than max", Shard{Current: 4, Max: 3}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.s.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTagIndexCanonicalOrder(t *testing.T) {
	project := fixtureProject()
	refs := TagIndex(project)

	want := []TagRef{
		{Identifier: "alpha-a", TagName: "1.0"},
		{Identifier: "alpha-a", TagName: "1.0-node"},
		{Identifier: "alpha-a", TagName: "2.0"},
		{Identifier: "alpha-a", TagName: "2.0-node"},
		{Identifier: "alpha-b", TagName: "1.0"},
		{Identifier: "beta", TagName: "1.0"},
	}

	if len(refs) != len(want) {
		t.Fatalf("TagIndex() returned %d refs, want %d: %v", len(refs), len(want), refs)
	}
	for i := range want {
		if refs[i] != want[i] {
			t.Errorf("refs[%d] = %+v, want %+v", i, refs[i], want[i])
		}
	}
}

func TestTagIndexDeterministic(t *testing.T) {
	project := fixtureProject()
	first := TagIndex(project)
	second := TagIndex(project)

	if len(first) != len(second) {
		t.Fatalf("lengths differ: %d vs %d", len(first), len(second))
	}
	for i := range first {
		if first[i] != second[i] {
			t.Errorf("index %d differs across calls: %+v vs %+v", i, first[i], second[i])
		}
	}
}

func TestTagIndexDistinguishesSameNameDifferentIdentifier(t *testing.T) {
	project := fixtureProject()
	refs := TagIndex(project)

	identifiers := map[string]bool{}
	for _, ref := range refs {
		if ref.Identifier == "alpha-a" || ref.Identifier == "alpha-b" {
			identifiers[ref.Identifier] = true
		}
	}
	if len(identifiers) != 2 {
		t.Errorf("expected both alpha-a and alpha-b to appear as distinct identifiers, got %v", identifiers)
	}
}

func TestNewTagSharderDisjointAndComplete(t *testing.T) {
	project := fixtureProject()
	all := TagIndex(project)

	const maxShards = 3
	owned := map[TagRef]int{} // ref -> number of shards claiming it

	for current := 1; current <= maxShards; current++ {
		s := Shard{Current: current, Max: maxShards}
		owns := NewTagSharder(project, s)
		for _, ref := range all {
			if owns(ref.Identifier, ref.TagName) {
				owned[ref]++
			}
		}
	}

	for _, ref := range all {
		if owned[ref] != 1 {
			t.Errorf("ref %+v owned by %d shards, want exactly 1", ref, owned[ref])
		}
	}
}

func TestNewTagSharderDisabledMatchesEverything(t *testing.T) {
	project := fixtureProject()
	owns := NewTagSharder(project, Shard{Current: 1, Max: 1})

	for _, ref := range TagIndex(project) {
		if !owns(ref.Identifier, ref.TagName) {
			t.Errorf("disabled sharder should own %+v", ref)
		}
	}
	// Match-all when disabled mirrors utils.MatchesFilter's "empty filter
	// matches everything" semantics - it does not check existence either.
	if !owns("nonexistent", "0.0") {
		t.Error("disabled sharder should match-all, even for a tag that doesn't exist")
	}
}

func TestNewBaseTagSharderSupersetProperty(t *testing.T) {
	project := fixtureProject()
	const maxShards = 3

	// For every base tag with variants, the set of shards that must build it
	// is exactly the union of {shard owning the base} and {shards owning any
	// of its variants}.
	for current := 1; current <= maxShards; current++ {
		s := Shard{Current: current, Max: maxShards}
		ownsExact := NewTagSharder(project, s)
		ownsBase := NewBaseTagSharder(project, s)

		// alpha-a/1.0 has variant alpha-a/1.0-node.
		wantBuildsBase := ownsExact("alpha-a", "1.0") || ownsExact("alpha-a", "1.0-node")
		if ownsBase("alpha-a", "1.0") != wantBuildsBase {
			t.Errorf("shard %d: ownsBase(alpha-a,1.0) = %v, want %v", current, ownsBase("alpha-a", "1.0"), wantBuildsBase)
		}

		// Every variant's owning shard must also build that variant's base.
		if ownsExact("alpha-a", "1.0-node") && !ownsBase("alpha-a", "1.0") {
			t.Errorf("shard %d owns variant 1.0-node but not its base 1.0", current)
		}
	}
}

func TestNewBaseTagSharderNoVariantsBehavesLikeExact(t *testing.T) {
	project := fixtureProject()
	const maxShards = 3

	for current := 1; current <= maxShards; current++ {
		s := Shard{Current: current, Max: maxShards}
		ownsExact := NewTagSharder(project, s)
		ownsBase := NewBaseTagSharder(project, s)

		if ownsExact("beta", "1.0") != ownsBase("beta", "1.0") {
			t.Errorf("shard %d: image with no variants should have identical exact/base ownership", current)
		}
	}
}

func TestCrossCommandAlignment(t *testing.T) {
	project := fixtureProject()
	const maxShards = 4

	for current := 1; current <= maxShards; current++ {
		s := Shard{Current: current, Max: maxShards}
		a := NewTagSharder(project, s)
		b := NewTagSharder(project, s)
		for _, ref := range TagIndex(project) {
			if a(ref.Identifier, ref.TagName) != b(ref.Identifier, ref.TagName) {
				t.Errorf("shard %d: two independently constructed sharders disagree on %+v", current, ref)
			}
		}
	}
}

func TestOwnedUnits(t *testing.T) {
	project := fixtureProject()
	all := TagIndex(project)

	t.Run("disabled returns everything", func(t *testing.T) {
		owned := OwnedUnits(project, Shard{Current: 1, Max: 1})
		if len(owned) != len(all) {
			t.Errorf("got %d units, want %d", len(owned), len(all))
		}
	})

	t.Run("union across shards equals full index, no overlap", func(t *testing.T) {
		const maxShards = 3
		seen := map[TagRef]int{}
		for current := 1; current <= maxShards; current++ {
			owned := OwnedUnits(project, Shard{Current: current, Max: maxShards})
			for _, ref := range owned {
				seen[ref]++
			}
		}
		if len(seen) != len(all) {
			t.Errorf("union has %d distinct units, want %d", len(seen), len(all))
		}
		for ref, count := range seen {
			if count != 1 {
				t.Errorf("unit %+v seen in %d shards, want 1", ref, count)
			}
		}
	})

	t.Run("over-provisioned shard is empty", func(t *testing.T) {
		// More shards than units: the last shard(s) own nothing.
		owned := OwnedUnits(project, Shard{Current: len(all) + 1, Max: len(all) + 5})
		if len(owned) != 0 {
			t.Errorf("expected empty shard, got %v", owned)
		}
	})

	t.Run("single unit with many shards concentrates in shard 1", func(t *testing.T) {
		single := &model.ContainerHiveProject{
			ImagesByIdentifier: map[string]*model.Image{
				"solo": {Identifier: "solo", Name: "solo", Tags: map[string]*model.Tag{"1.0": {Name: "1.0"}}},
			},
		}
		owned := OwnedUnits(single, Shard{Current: 1, Max: 10})
		if len(owned) != 1 {
			t.Fatalf("shard 1 should own the single unit, got %v", owned)
		}
		for current := 2; current <= 10; current++ {
			owned := OwnedUnits(single, Shard{Current: current, Max: 10})
			if len(owned) != 0 {
				t.Errorf("shard %d should be empty, got %v", current, owned)
			}
		}
	})
}

func TestUnitCountByName(t *testing.T) {
	project := fixtureProject()
	counts := UnitCountByName(project)

	// alpha-a: 2 tags × (1+1 variant) = 4 units, alpha-b: 1 tag × 1 = 1 unit → total 5
	if counts["alpha"] != 5 {
		t.Errorf("expected alpha to have 5 units, got %d", counts["alpha"])
	}
	// beta: 1 tag × 1 = 1 unit
	if counts["beta"] != 1 {
		t.Errorf("expected beta to have 1 unit, got %d", counts["beta"])
	}
}

func TestUnitCountByName_ConsistencyWithTagIndex(t *testing.T) {
	project := fixtureProject()
	counts := UnitCountByName(project)
	index := TagIndex(project)

	// Sum of per-name counts must equal total units from TagIndex.
	total := 0
	for _, c := range counts {
		total += c
	}
	if total != len(index) {
		t.Errorf("UnitCountByName sum (%d) != TagIndex length (%d)", total, len(index))
	}

	// Verify per-name breakdown matches TagIndex grouping.
	perName := make(map[string]int)
	for _, ref := range index {
		perName[project.ImagesByIdentifier[ref.Identifier].Name]++
	}
	for name, expected := range perName {
		if counts[name] != expected {
			t.Errorf("name %q: UnitCountByName=%d, TagIndex grouping=%d", name, counts[name], expected)
		}
	}
}

func TestUnitCountByName_NoVariants(t *testing.T) {
	project := &model.ContainerHiveProject{
		ImagesByIdentifier: map[string]*model.Image{
			"app": {Identifier: "app", Name: "app", Tags: map[string]*model.Tag{"1.0": {}, "2.0": {}, "3.0": {}}},
		},
		ImagesByName: map[string][]*model.Image{
			"app": {{Name: "app", Tags: map[string]*model.Tag{"1.0": {}, "2.0": {}, "3.0": {}}}},
		},
	}
	counts := UnitCountByName(project)
	if counts["app"] != 3 {
		t.Errorf("expected 3 units, got %d", counts["app"])
	}
}
