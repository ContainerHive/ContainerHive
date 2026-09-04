package cache

import "testing"

func TestComputeContentHash(t *testing.T) {
	base := []byte("FROM scratch\n")

	t.Run("deterministic for identical inputs", func(t *testing.T) {
		a := ComputeContentHash(base, map[string]string{"FOO": "bar"}, "linux/amd64")
		b := ComputeContentHash(base, map[string]string{"FOO": "bar"}, "linux/amd64")
		if a != b {
			t.Errorf("expected identical hashes, got %q and %q", a, b)
		}
	})

	t.Run("insensitive to build arg map ordering", func(t *testing.T) {
		a := ComputeContentHash(base, map[string]string{"FOO": "bar", "BAZ": "qux"}, "linux/amd64")
		b := ComputeContentHash(base, map[string]string{"BAZ": "qux", "FOO": "bar"}, "linux/amd64")
		if a != b {
			t.Errorf("expected hash to be independent of map ordering, got %q and %q", a, b)
		}
	})

	tests := []struct {
		name       string
		dockerfile []byte
		buildArgs  map[string]string
		platform   string
	}{
		{"different dockerfile content", []byte("FROM scratch\nRUN echo hi\n"), map[string]string{"FOO": "bar"}, "linux/amd64"},
		{"different build arg value", base, map[string]string{"FOO": "baz"}, "linux/amd64"},
		{"different build arg key", base, map[string]string{"OTHER": "bar"}, "linux/amd64"},
		{"different platform", base, map[string]string{"FOO": "bar"}, "linux/arm64"},
	}

	baseline := ComputeContentHash(base, map[string]string{"FOO": "bar"}, "linux/amd64")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ComputeContentHash(tt.dockerfile, tt.buildArgs, tt.platform)
			if got == baseline {
				t.Errorf("expected hash to differ from baseline, got same value %q", got)
			}
		})
	}
}
