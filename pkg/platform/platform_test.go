package platform

import (
	"testing"
)

func TestNormalize(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"linux/amd64", "linux/amd64"},
		{"linux/arm64", "linux/arm64"},
		{"linux/arm/v7", "linux/arm/v7"},
		{"windows/amd64", "windows/amd64"},
		{"amd64", "linux/amd64"},
		{"arm64", "linux/arm64"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := Normalize(tt.input)
			if got != tt.want {
				t.Errorf("Normalize(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

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

func TestShort(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"linux/amd64", "amd64"},
		{"linux/arm64", "arm64"},
		{"amd64", "amd64"},
		{"linux/arm/v7", "linux/arm/v7"},
		{"linux/arm64/v8", "linux/arm64/v8"},
		{"windows/amd64", "windows/amd64"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := Short(tt.input)
			if got != tt.want {
				t.Errorf("Short(%q) = %q, want %q", tt.input, got, tt.want)
			}
			if Normalize(got) != Normalize(tt.input) {
				t.Errorf("Normalize(Short(%q)) = %q, want %q", tt.input, Normalize(got), Normalize(tt.input))
			}
		})
	}
}

func TestArch(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"linux/amd64", "amd64"},
		{"arm64", "arm64"},
		{"linux/arm64/v8", "arm64"},
		{"linux/arm/v7", "arm"},
		{"windows/amd64", "amd64"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := Arch(tt.input)
			if got != tt.want {
				t.Errorf("Arch(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestUnsupported(t *testing.T) {
	tests := []struct {
		name       string
		requested  []string
		configured [][]string
		want       []string
	}{
		{"all supported", []string{"amd64", "linux/arm64"}, [][]string{{"linux/amd64"}, {"arm64"}}, nil},
		{"none requested", nil, [][]string{{"linux/amd64"}}, nil},
		{"typo", []string{"arm"}, [][]string{{"linux/arm/v7"}}, []string{"arm"}},
		{"only in variant", []string{"riscv64"}, [][]string{{"linux/amd64"}, nil, {"linux/riscv64"}}, nil},
		{"keeps requested order", []string{"s390x", "amd64", "ppc64le"}, [][]string{{"linux/amd64"}}, []string{"s390x", "ppc64le"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Unsupported(tt.requested, tt.configured...)
			if len(got) != len(tt.want) {
				t.Fatalf("Unsupported() = %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("Unsupported() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestResolve(t *testing.T) {
	t.Run("override selects within variant platforms", func(t *testing.T) {
		got := Resolve(
			[]string{"linux/arm64"},
			[]string{"linux/amd64", "linux/arm64"},
			[]string{"linux/amd64", "linux/arm64"},
			[]string{"linux/amd64", "linux/arm64"},
		)
		if len(got) != 1 || got[0] != "linux/arm64" {
			t.Errorf("expected [linux/arm64], got %v", got)
		}
	})

	t.Run("override not supported by variant yields empty", func(t *testing.T) {
		got := Resolve(
			[]string{"linux/arm64"},
			[]string{"linux/amd64", "linux/arm64"},
			nil,
			[]string{"linux/amd64"},
		)
		if len(got) != 0 {
			t.Errorf("expected empty slice, got %v", got)
		}
	})

	t.Run("override not supported by image yields empty", func(t *testing.T) {
		got := Resolve(
			[]string{"linux/arm64"},
			[]string{"linux/amd64", "linux/arm64"},
			[]string{"linux/amd64"},
			nil,
		)
		if len(got) != 0 {
			t.Errorf("expected empty slice, got %v", got)
		}
	})

	t.Run("override not configured anywhere yields empty", func(t *testing.T) {
		got := Resolve(
			[]string{"linux/riscv64"},
			[]string{"linux/amd64"},
			nil,
			nil,
		)
		if len(got) != 0 {
			t.Errorf("expected empty slice, got %v", got)
		}
	})

	t.Run("override keeps configured order and normalizes", func(t *testing.T) {
		got := Resolve(
			[]string{"arm64", "amd64"},
			[]string{"amd64", "linux/arm64", "linux/riscv64"},
			nil,
			nil,
		)
		if len(got) != 2 || got[0] != "linux/amd64" || got[1] != "linux/arm64" {
			t.Errorf("expected [linux/amd64 linux/arm64], got %v", got)
		}
	})

	t.Run("variant wins", func(t *testing.T) {
		got := Resolve(
			nil,
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
			nil,
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
			nil,
			[]string{"linux/amd64"},
			nil,
			nil,
		)
		if len(got) != 1 || got[0] != "linux/amd64" {
			t.Errorf("expected global platforms, got %v", got)
		}
	})

	t.Run("all empty returns empty slice", func(t *testing.T) {
		got := Resolve(nil, nil, nil, nil)
		if len(got) != 0 {
			t.Errorf("expected empty slice, got %v", got)
		}
	})

	t.Run("drops duplicates across short and full forms", func(t *testing.T) {
		got := Resolve(
			nil,
			[]string{"amd64", "linux/amd64", "arm64"},
			nil,
			nil,
		)
		if len(got) != 2 || got[0] != "linux/amd64" || got[1] != "linux/arm64" {
			t.Errorf("expected [linux/amd64 linux/arm64], got %v", got)
		}
	})

	t.Run("override does not duplicate mixed forms", func(t *testing.T) {
		got := Resolve(
			[]string{"amd64"},
			[]string{"amd64", "linux/amd64"},
			nil,
			nil,
		)
		if len(got) != 1 || got[0] != "linux/amd64" {
			t.Errorf("expected [linux/amd64], got %v", got)
		}
	})

	t.Run("normalizes short platform names", func(t *testing.T) {
		got := Resolve(
			nil,
			[]string{"amd64", "arm64"},
			nil,
			nil,
		)
		if len(got) != 2 || got[0] != "linux/amd64" || got[1] != "linux/arm64" {
			t.Errorf("expected normalized platforms, got %v", got)
		}
	})
}
