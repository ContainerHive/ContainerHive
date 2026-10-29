package rendering

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ContainerHive/ContainerHive/pkg/model"
)

func TestCopyChiselSlices(t *testing.T) {
	dir := t.TempDir()
	entrypoint := filepath.Join(dir, model.ChiselFileName)
	if err := os.WriteFile(entrypoint, []byte("release: ubuntu-24.04\nslices:\n  - base-files_base\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	t.Run("missing slices dir is not an error", func(t *testing.T) {
		target := t.TempDir()
		if err := copyChiselSlices(target, entrypoint); err != nil {
			t.Fatalf("copyChiselSlices failed: %v", err)
		}
		if _, err := os.Stat(filepath.Join(target, "slices")); !os.IsNotExist(err) {
			t.Errorf("slices copied without a source, stat error: %v", err)
		}
	})

	t.Run("copies slices dir", func(t *testing.T) {
		if err := os.MkdirAll(filepath.Join(dir, "slices"), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "slices", "hello.yaml"), []byte("package: hello\n"), 0o600); err != nil {
			t.Fatal(err)
		}

		target := t.TempDir()
		if err := copyChiselSlices(target, entrypoint); err != nil {
			t.Fatalf("copyChiselSlices failed: %v", err)
		}
		got, err := os.ReadFile(filepath.Join(target, "slices", "hello.yaml"))
		if err != nil {
			t.Fatalf("slices/hello.yaml missing after copy: %v", err)
		}
		if string(got) != "package: hello\n" {
			t.Errorf("slices/hello.yaml content = %q", got)
		}
	})

	t.Run("ignores slices dir next to a Dockerfile", func(t *testing.T) {
		dockerDir := t.TempDir()
		dockerfile := filepath.Join(dockerDir, "Dockerfile")
		if err := os.WriteFile(dockerfile, []byte("FROM scratch\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Join(dockerDir, "slices"), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dockerDir, "slices", "hello.yaml"), []byte("package: hello\n"), 0o600); err != nil {
			t.Fatal(err)
		}

		target := t.TempDir()
		if err := copyChiselSlices(target, dockerfile); err != nil {
			t.Fatalf("copyChiselSlices failed: %v", err)
		}
		if _, err := os.Stat(filepath.Join(target, "slices")); !os.IsNotExist(err) {
			t.Errorf("slices copied for a Dockerfile entrypoint, stat error: %v", err)
		}
	})
}
