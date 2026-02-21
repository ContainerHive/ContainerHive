package main

import (
	"testing"
)

func TestParseFilters(t *testing.T) {
	t.Run("empty args returns empty filters", func(t *testing.T) {
		filters := parseFilters([]string{})
		if len(filters) != 0 {
			t.Errorf("expected 0 filters, got %d", len(filters))
		}
	})

	t.Run("image name only", func(t *testing.T) {
		filters := parseFilters([]string{"ubuntu"})
		if len(filters) != 1 {
			t.Fatalf("expected 1 filter, got %d", len(filters))
		}
		if filters[0].ImageName != "ubuntu" {
			t.Errorf("expected ImageName ubuntu, got %q", filters[0].ImageName)
		}
		if filters[0].TagName != "" {
			t.Errorf("expected empty TagName, got %q", filters[0].TagName)
		}
	})

	t.Run("image name with tag", func(t *testing.T) {
		filters := parseFilters([]string{"ubuntu:24.04"})
		if len(filters) != 1 {
			t.Fatalf("expected 1 filter, got %d", len(filters))
		}
		if filters[0].ImageName != "ubuntu" {
			t.Errorf("expected ImageName ubuntu, got %q", filters[0].ImageName)
		}
		if filters[0].TagName != "24.04" {
			t.Errorf("expected TagName 24.04, got %q", filters[0].TagName)
		}
	})

	t.Run("multiple filters", func(t *testing.T) {
		filters := parseFilters([]string{"ubuntu:24.04", "dotnet"})
		if len(filters) != 2 {
			t.Fatalf("expected 2 filters, got %d", len(filters))
		}
		if filters[0].ImageName != "ubuntu" || filters[0].TagName != "24.04" {
			t.Errorf("filter[0] mismatch: %+v", filters[0])
		}
		if filters[1].ImageName != "dotnet" || filters[1].TagName != "" {
			t.Errorf("filter[1] mismatch: %+v", filters[1])
		}
	})
}
