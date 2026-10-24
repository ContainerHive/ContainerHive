package discovery

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ContainerHive/ContainerHive/pkg/model"
)

func TestDiscoverChiselProject(t *testing.T) {
	project, err := DiscoverProject(context.Background(), "../testdata/chisel-project")
	if err != nil {
		t.Fatalf("DiscoverProject failed: %v", err)
	}

	img, ok := project.ImagesByIdentifier["hello"]
	if !ok {
		t.Fatalf("chisel image not discovered, got identifiers: %v", keys(project.ImagesByIdentifier))
	}

	if img.BuildBackend != model.BuildBackendChisel {
		t.Errorf("BuildBackend = %q, want %q", img.BuildBackend, model.BuildBackendChisel)
	}
	if filepath.Base(img.BuildEntryPointPath) != model.ChiselFileName {
		t.Errorf("BuildEntryPointPath = %q, want a %s", img.BuildEntryPointPath, model.ChiselFileName)
	}
	if len(img.Tags) != 1 || img.Tags["1.0.0"] == nil {
		t.Errorf("Tags = %v, want a single 1.0.0 tag", img.Tags)
	}
	if img.RunConfig != nil {
		t.Errorf("RunConfig = %v, want nil without a config block", img.RunConfig)
	}
}

func TestResolveBuildEntrypointChisel(t *testing.T) {
	setup := func(t *testing.T, files map[string]string) string {
		t.Helper()
		dir := t.TempDir()
		for name, content := range files {
			if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
				t.Fatal(err)
			}
		}
		return dir
	}

	const cutSpec = "release: ubuntu-24.04\nslices:\n  - base-files_base\n"

	t.Run("explicit chisel backend requires a chisel.yaml", func(t *testing.T) {
		dir := setup(t, map[string]string{"image.yml": "description: x\ntags:\n  - name: 1.0\n"})
		_, err := resolveBuildEntrypoint(model.BuildBackendChisel, dir)
		if err == nil {
			t.Fatal("expected an error without a chisel.yaml, got nil")
		}
		if !strings.Contains(err.Error(), model.ChiselFileName) {
			t.Errorf("error does not mention %s: %v", model.ChiselFileName, err)
		}
	})

	t.Run("explicit chisel backend resolves the cut spec", func(t *testing.T) {
		dir := setup(t, map[string]string{"chisel.yaml": cutSpec})
		entrypoint, err := resolveBuildEntrypoint(model.BuildBackendChisel, dir)
		if err != nil {
			t.Fatalf("resolveBuildEntrypoint failed: %v", err)
		}
		if entrypoint.Backend != model.BuildBackendChisel {
			t.Errorf("backend = %q, want chisel", entrypoint.Backend)
		}
	})

	t.Run("a sole chisel.yaml auto-detects the chisel backend", func(t *testing.T) {
		dir := setup(t, map[string]string{"chisel.yaml": cutSpec})
		entrypoint, err := resolveBuildEntrypoint(model.BuildBackendAuto, dir)
		if err != nil {
			t.Fatalf("resolveBuildEntrypoint failed: %v", err)
		}
		if entrypoint.Backend != model.BuildBackendChisel {
			t.Errorf("backend = %q, want chisel", entrypoint.Backend)
		}
	})

	t.Run("chisel.yaml next to a Dockerfile is ambiguous and fails", func(t *testing.T) {
		dir := setup(t, map[string]string{
			"chisel.yaml": cutSpec,
			"Dockerfile":  "FROM scratch\n",
		})
		_, err := resolveBuildEntrypoint(model.BuildBackendAuto, dir)
		if err == nil {
			t.Fatal("expected an ambiguity error, got nil")
		}
		if !strings.Contains(err.Error(), "ambiguous") {
			t.Errorf("error does not report the ambiguity: %v", err)
		}
	})

	t.Run("chisel.yaml next to a flake is ambiguous and fails", func(t *testing.T) {
		dir := setup(t, map[string]string{
			"chisel.yaml": cutSpec,
			"flake.nix":   "{ }\n",
		})
		_, err := resolveBuildEntrypoint(model.BuildBackendAuto, dir)
		if err == nil {
			t.Fatal("expected an ambiguity error, got nil")
		}
		if !strings.Contains(err.Error(), "ambiguous") {
			t.Errorf("error does not report the ambiguity: %v", err)
		}
	})

	t.Run("unknown backend names all options", func(t *testing.T) {
		_, err := resolveBuildEntrypoint("bogus", t.TempDir())
		if err == nil || !strings.Contains(err.Error(), "chisel") {
			t.Errorf("expected an unknown backend error naming chisel, got: %v", err)
		}
	})
}
