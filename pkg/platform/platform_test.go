package platform

import (
	"testing"
)

func TestSanitize(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"linux/amd64", "linux-amd64"},
		{"linux/arm64", "linux-arm64"},
		{"linux/arm/v7", "linux-arm-v7"},
		{"windows/amd64", "windows-amd64"},
		{"already-sanitized", "already-sanitized"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := Sanitize(tt.input)
			if got != tt.want {
				t.Errorf("Sanitize(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestResolve(t *testing.T) {
	t.Run("variant wins", func(t *testing.T) {
		got := Resolve(
			[]string{"linux/amd64"},
			[]string{"linux/arm64"},
			[]string{"linux/amd64", "linux/arm64"},
		)
		if len(got) != 2 || got[0] != "linux/amd64" || got[1] != "linux/arm64" {
			t.Errorf("expected variant platforms, got %v", got)
		}
	})

	t.Run("image wins over global", func(t *testing.T) {
		got := Resolve(
			[]string{"linux/amd64"},
			[]string{"linux/arm64"},
			nil,
		)
		if len(got) != 1 || got[0] != "linux/arm64" {
			t.Errorf("expected image platforms, got %v", got)
		}
	})

	t.Run("falls back to global", func(t *testing.T) {
		got := Resolve(
			[]string{"linux/amd64"},
			nil,
			nil,
		)
		if len(got) != 1 || got[0] != "linux/amd64" {
			t.Errorf("expected global platforms, got %v", got)
		}
	})

	t.Run("all empty returns nil", func(t *testing.T) {
		got := Resolve(nil, nil, nil)
		if got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})
}
