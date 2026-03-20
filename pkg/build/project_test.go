package build

import (
	"testing"
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
	address string
	local   bool
}

func (r *mockRegistry) Address() string { return r.address }
func (r *mockRegistry) IsLocal() bool   { return r.local }

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
