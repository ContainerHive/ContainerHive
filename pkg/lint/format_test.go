package lint

import (
	"testing"
)

func TestParseFormats(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		opts, err := ParseFormats(nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(opts) != 1 || opts[0].Name != "terminal" {
			t.Errorf("expected [terminal], got %+v", opts)
		}
	})

	t.Run("empty slice default", func(t *testing.T) {
		opts, err := ParseFormats([]string{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(opts) != 1 || opts[0].Name != "terminal" {
			t.Errorf("expected [terminal], got %+v", opts)
		}
	})

	t.Run("github-actions", func(t *testing.T) {
		opts, err := ParseFormats([]string{"github-actions"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(opts) != 1 || opts[0].Name != "github-actions" {
			t.Errorf("unexpected options: %+v", opts)
		}
	})

	t.Run("codeclimate with path", func(t *testing.T) {
		opts, err := ParseFormats([]string{"codeclimate=report.json"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(opts) != 1 {
			t.Fatalf("expected 1 option, got %d", len(opts))
		}
		if opts[0].Name != "codeclimate" || opts[0].Path != "report.json" {
			t.Errorf("unexpected options: %+v", opts)
		}
	})

	t.Run("multiple formats", func(t *testing.T) {
		opts, err := ParseFormats([]string{"terminal", "github-actions", "codeclimate=out.json"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(opts) != 3 {
			t.Fatalf("expected 3 options, got %d", len(opts))
		}
		if opts[0].Name != "terminal" || opts[1].Name != "github-actions" || opts[2].Name != "codeclimate" || opts[2].Path != "out.json" {
			t.Errorf("unexpected options: %+v", opts)
		}
	})

	t.Run("unknown format", func(t *testing.T) {
		_, err := ParseFormats([]string{"unknown-format"})
		if err == nil {
			t.Fatal("expected error for unknown format")
		}
	})

	t.Run("codeclimate without path", func(t *testing.T) {
		_, err := ParseFormats([]string{"codeclimate"})
		if err == nil {
			t.Fatal("expected error for codeclimate without path")
		}
	})

	t.Run("terminal with path", func(t *testing.T) {
		_, err := ParseFormats([]string{"terminal=out.txt"})
		if err == nil {
			t.Fatal("expected error for terminal with path")
		}
	})

	t.Run("github-actions with path", func(t *testing.T) {
		_, err := ParseFormats([]string{"github-actions=ann.txt"})
		if err == nil {
			t.Fatal("expected error for github-actions with path")
		}
	})

	t.Run("duplicate format", func(t *testing.T) {
		_, err := ParseFormats([]string{"terminal", "terminal"})
		if err == nil {
			t.Fatal("expected error for duplicate format")
		}
	})

	t.Run("empty name", func(t *testing.T) {
		_, err := ParseFormats([]string{"=report.json"})
		if err == nil {
			t.Fatal("expected error for empty format name")
		}
	})
}

func TestFormatPrototypeHasPath(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{name: "terminal", want: false},
		{name: "github-actions", want: false},
		{name: "codeclimate", want: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			proto, ok := formatPrototypes[tc.name]
			if !ok {
				t.Fatalf("missing prototype for %q", tc.name)
			}
			if got := proto.HasPath(); got != tc.want {
				t.Errorf("HasPath() = %v, want %v", got, tc.want)
			}
		})
	}
}
