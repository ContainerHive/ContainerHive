package discovery

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/ContainerHive/ContainerHive/pkg/model"
)

func TestDiscoverNixProject(t *testing.T) {
	project, err := DiscoverProject(context.Background(), "../testdata/nix-project")
	if err != nil {
		t.Fatalf("DiscoverProject failed: %v", err)
	}

	imageName := project.RootDir + "/images/busybox"
	img, ok := project.ImagesByIdentifier["busybox"]
	if !ok {
		t.Fatalf("nix image not discovered, got identifiers: %v", keys(project.ImagesByIdentifier))
	}
	_ = imageName

	if img.BuildBackend != model.BuildBackendNix {
		t.Errorf("BuildBackend = %q, want %q", img.BuildBackend, model.BuildBackendNix)
	}
	if filepath.Base(img.BuildEntryPointPath) != "flake.nix" {
		t.Errorf("BuildEntryPointPath = %q, want a flake.nix", img.BuildEntryPointPath)
	}
	if len(img.Tags) != 1 || img.Tags["1.36.0"] == nil {
		t.Errorf("Tags = %v, want a single 1.36.0 tag", img.Tags)
	}
}

func keys(m map[string]*model.Image) []string {
	names := make([]string, 0, len(m))
	for name := range m {
		names = append(names, name)
	}
	return names
}
