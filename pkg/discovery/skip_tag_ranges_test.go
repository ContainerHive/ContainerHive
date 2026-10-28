package discovery

import (
	"context"
	"testing"
)

func TestDiscoverProject_WithSkipTagRanges(t *testing.T) {
	root := writeProjectWithTagRanges(t, tagRangeYML)

	// The resolver would add the 1.2 tag; skipping must not consult it.
	project, err := DiscoverProject(context.Background(), root,
		WithTagRangeResolver(fixtureResolver("1.2.0")),
		WithSkipTagRanges(),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	img := project.ImagesByIdentifier["app"]
	if img == nil {
		t.Fatal("expected image \"app\" to be discovered")
	}
	if _, ok := img.Tags["static"]; !ok {
		t.Error("expected the static tag to survive")
	}
	if _, ok := img.Tags["1.2"]; ok {
		t.Errorf("expected no generated tags, got tags: %v", tagNames(img.Tags))
	}
}
