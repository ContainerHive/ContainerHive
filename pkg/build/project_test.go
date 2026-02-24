package build

import "testing"

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
