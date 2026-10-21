package build

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHashSource(t *testing.T) {
	tests := []struct {
		name       string
		entrypoint string
		want       string
	}{
		{"flake includes flake.lock", "flake.nix", "entrylock"},
		{"dockerfile ignores flake.lock", "Dockerfile", "entry"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, tt.entrypoint)
			if err := os.WriteFile(path, []byte("entry"), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, "flake.lock"), []byte("lock"), 0o600); err != nil {
				t.Fatal(err)
			}

			got, err := hashSource(path)
			if err != nil {
				t.Fatalf("hashSource failed: %v", err)
			}
			if string(got) != tt.want {
				t.Errorf("hashSource() = %q, want %q", got, tt.want)
			}
		})
	}
}
