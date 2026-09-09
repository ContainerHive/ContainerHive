package model

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func unmarshalLevelSelect(t *testing.T, yamlStr string) (LevelSelect, error) {
	t.Helper()
	var wrapper struct {
		Sel LevelSelect `yaml:"sel"`
	}
	err := yaml.Unmarshal([]byte(yamlStr), &wrapper)
	return wrapper.Sel, err
}

func TestLevelSelect_UnmarshalYAML(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		want    LevelSelect
		wantErr bool
	}{
		{"latest keyword", "sel: latest", LevelSelect{Last: 1}, false},
		{"all keyword", "sel: all", LevelSelect{All: true}, false},
		{"bare integer", "sel: 3", LevelSelect{Last: 3}, false},
		{"mapping last", "sel: {last: 3}", LevelSelect{Last: 3}, false},
		{"mapping all true", "sel: {all: true}", LevelSelect{All: true}, false},
		{"mapping all false", "sel: {all: false}", LevelSelect{}, false},
		{"unknown scalar", "sel: newest", LevelSelect{}, true},
		{"zero integer", "sel: 0", LevelSelect{}, true},
		{"negative integer", "sel: -1", LevelSelect{}, true},
		{"unknown mapping key", "sel: {lst: 3}", LevelSelect{}, true},
		{"last and all both set", "sel: {last: 2, all: true}", LevelSelect{}, true},
		{"non-numeric last", `sel: {last: "abc"}`, LevelSelect{}, true},
		{"non-boolean all", `sel: {all: "yes"}`, LevelSelect{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := unmarshalLevelSelect(t, tt.yaml)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected an error, got none (result: %+v)", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestSelectConfig_UnmarshalYAML_MixedForms(t *testing.T) {
	// Exercises the issue's own example: mixed scalar and mapping forms in
	// one select block.
	var cfg struct {
		Select SelectConfig `yaml:"select"`
	}
	yamlStr := `
select:
  major: { last: 3 }
  minor: { last: 3 }
  patch: latest
`
	if err := yaml.Unmarshal([]byte(yamlStr), &cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Select.Major == nil || cfg.Select.Major.Last != 3 {
		t.Errorf("major: got %+v, want Last=3", cfg.Select.Major)
	}
	if cfg.Select.Minor == nil || cfg.Select.Minor.Last != 3 {
		t.Errorf("minor: got %+v, want Last=3", cfg.Select.Minor)
	}
	if cfg.Select.Patch == nil || cfg.Select.Patch.Last != 1 {
		t.Errorf("patch: got %+v, want Last=1 (from \"latest\")", cfg.Select.Patch)
	}
}
