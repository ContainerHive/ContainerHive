package build

import "testing"

func TestMatchesFilters_Empty(t *testing.T) {
	if !matchesFilters(nil, "any", "any", false) {
		t.Error("empty filters should match everything")
	}
	if !matchesFilters([]Filter{}, "any", "any", false) {
		t.Error("empty filter slice should match everything")
	}
}

func TestMatchesFilters_ImageName(t *testing.T) {
	filters := []Filter{{ImageName: "app", IncludeVariants: true}}
	if !matchesFilters(filters, "app", "latest", false) {
		t.Error("should match app")
	}
	if matchesFilters(filters, "other", "latest", false) {
		t.Error("should not match other")
	}
}

func TestMatchesFilters_TagName(t *testing.T) {
	filters := []Filter{{TagName: "1.0", IncludeVariants: true}}
	if !matchesFilters(filters, "any", "1.0", false) {
		t.Error("should match tag 1.0")
	}
	if matchesFilters(filters, "any", "2.0", false) {
		t.Error("should not match tag 2.0")
	}
}

func TestMatchesFilters_Variants(t *testing.T) {
	withVariants := []Filter{{ImageName: "app", IncludeVariants: true}}
	withoutVariants := []Filter{{ImageName: "app", IncludeVariants: false}}

	if !matchesFilters(withVariants, "app", "1.0-slim", true) {
		t.Error("should match variant when IncludeVariants is true")
	}
	if matchesFilters(withoutVariants, "app", "1.0-slim", true) {
		t.Error("should not match variant when IncludeVariants is false")
	}
}

func TestMatchesFilters_Combined(t *testing.T) {
	filters := []Filter{{ImageName: "app", TagName: "1.0", IncludeVariants: true}}
	if !matchesFilters(filters, "app", "1.0", false) {
		t.Error("should match exact image+tag")
	}
	if matchesFilters(filters, "app", "2.0", false) {
		t.Error("should not match different tag")
	}
	if matchesFilters(filters, "other", "1.0", false) {
		t.Error("should not match different image")
	}
}
