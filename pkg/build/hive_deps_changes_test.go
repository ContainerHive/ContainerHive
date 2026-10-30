package build

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ContainerHive/ContainerHive/pkg/changes"
)

func TestResolveHiveDeps_ChangeDetection(t *testing.T) {
	tests := []struct {
		name    string
		changes *changes.Set
		want    string
	}{
		{name: "no change set keeps build ID", changes: nil, want: "docker-image://ghcr.io/myorg/base:1.0-build.42"},
		{name: "affected dependency keeps build ID", changes: &changes.Set{Affected: []string{"base", "app"}}, want: "docker-image://ghcr.io/myorg/base:1.0-build.42"},
		{name: "skipped dependency uses production tag", changes: &changes.Set{Affected: []string{"app"}}, want: "docker-image://ghcr.io/myorg/base:1.0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			dockerfile := filepath.Join(dir, "Dockerfile")
			if err := os.WriteFile(dockerfile, []byte("FROM __hive__/base:1.0\n"), 0o644); err != nil {
				t.Fatal(err)
			}

			result, err := ResolveHiveDeps(HiveDepsOpts{
				DockerfilePath:  dockerfile,
				DistPath:        dir,
				PlatformStr:     "linux/amd64",
				RegistryAddress: "ghcr.io/myorg",
				BuildID:         "42",
				Changes:         tt.changes,
			})
			if err != nil {
				t.Fatalf("ResolveHiveDeps() error = %v", err)
			}
			defer result.Cleanup()
			if got := result.NamedContexts["context:hive-dep/base:1.0"]; got != tt.want {
				t.Errorf("named context = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestResolveParentImage_ChangeDetection(t *testing.T) {
	result, err := ResolveParentImage("__hive__/base:1.0", HiveDepsOpts{
		DistPath:        t.TempDir(),
		PlatformStr:     "linux/amd64",
		RegistryAddress: "ghcr.io/myorg",
		BuildID:         "42",
		Changes:         &changes.Set{Affected: []string{"app"}},
	})
	if err != nil {
		t.Fatalf("ResolveParentImage() error = %v", err)
	}
	defer result.Deps.Cleanup()
	if result.Parent.RegistryRef != "ghcr.io/myorg/base:1.0" {
		t.Errorf("RegistryRef = %q, want ghcr.io/myorg/base:1.0", result.Parent.RegistryRef)
	}
}
