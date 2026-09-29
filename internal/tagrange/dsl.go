package tagrange

import (
	"github.com/ContainerHive/ContainerHive/pkg/model"
)

// latestOnly is the default LevelSelect for an omitted select level: keep
// only the single highest value. An omitted level never defaults to "all",
// so a config that forgets a level can't silently explode the tag count.
var latestOnly = &model.LevelSelect{Last: 1}

// selectVersions applies the hierarchical select DSL to a descending-sorted
// candidate list: keep majors per select.Major, then within each kept major
// keep minors per select.Minor, then within each kept minor keep patches per
// select.Patch. The result preserves the descending order.
func selectVersions(sortedDescending []candidate, sel *model.SelectConfig) []candidate {
	majorSel, minorSel, patchSel := latestOnly, latestOnly, latestOnly
	if sel != nil {
		if sel.Major != nil {
			majorSel = sel.Major
		}
		if sel.Minor != nil {
			minorSel = sel.Minor
		}
		if sel.Patch != nil {
			patchSel = sel.Patch
		}
	}

	var result []candidate
	for _, majorGroup := range groupByLevel(sortedDescending, func(c candidate) int64 { return int64(c.ver.Major()) }, majorSel) {
		for _, minorGroup := range groupByLevel(majorGroup, func(c candidate) int64 { return int64(c.ver.Minor()) }, minorSel) {
			result = append(result, keepLevel(minorGroup, patchSel)...)
		}
	}
	return result
}

// groupByLevel splits a descending-sorted candidate list into consecutive
// runs sharing the same key (e.g. all candidates with major 24), in the
// order those keys first appear, then keeps only the runs selected by sel.
func groupByLevel(sortedDescending []candidate, key func(candidate) int64, sel *model.LevelSelect) [][]candidate {
	var groups [][]candidate
	var groupKeys []int64
	for _, c := range sortedDescending {
		k := key(c)
		if n := len(groups); n > 0 && groupKeys[n-1] == k {
			groups[n-1] = append(groups[n-1], c)
			continue
		}
		groups = append(groups, []candidate{c})
		groupKeys = append(groupKeys, k)
	}

	if sel.All {
		return groups
	}
	last := sel.Last
	if last <= 0 {
		last = 1
	}
	if last > len(groups) {
		last = len(groups)
	}
	return groups[:last]
}

// keepLevel selects candidates at a leaf level (patch): either all of them,
// or the N highest, honoring sortedDescending's existing order.
func keepLevel(sortedDescending []candidate, sel *model.LevelSelect) []candidate {
	if sel.All {
		return sortedDescending
	}
	last := sel.Last
	if last <= 0 {
		last = 1
	}
	if last > len(sortedDescending) {
		last = len(sortedDescending)
	}
	return sortedDescending[:last]
}
