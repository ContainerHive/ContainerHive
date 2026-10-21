package rendering

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCopyFlakeLock(t *testing.T) {
	dir := t.TempDir()
	target := t.TempDir()

	t.Run("missing flake.lock is not an error", func(t *testing.T) {
		entrypoint := filepath.Join(dir, "flake.nix")
		if err := copyFlakeLock(target, entrypoint); err != nil {
			t.Fatalf("copyFlakeLock failed: %v", err)
		}
		if _, err := os.Stat(filepath.Join(target, "flake.lock")); !os.IsNotExist(err) {
			t.Errorf("flake.lock copied without a source, stat error: %v", err)
		}
	})

	t.Run("copies flake.lock", func(t *testing.T) {
		lockSrc := filepath.Join(dir, "flake.lock")
		want := "{\"version\":7}\n"
		if err := os.WriteFile(lockSrc, []byte(want), 0o600); err != nil {
			t.Fatal(err)
		}

		if err := copyFlakeLock(target, lockSrc); err != nil {
			t.Fatalf("copyFlakeLock failed: %v", err)
		}

		got, err := os.ReadFile(filepath.Join(target, "flake.lock"))
		if err != nil {
			t.Fatalf("flake.lock missing after copy: %v", err)
		}
		if string(got) != want {
			t.Errorf("flake.lock content = %q, want %q", got, want)
		}
	})
}
