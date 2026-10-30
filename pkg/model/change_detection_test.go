package model

import (
	"slices"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestChangeDetectionConfigPatterns(t *testing.T) {
	tests := []struct {
		name       string
		yaml       string
		wantExcl   []string
		wantGlobal []string
	}{
		{
			name:       "unset uses defaults",
			yaml:       "{}",
			wantExcl:   DefaultChangeDetectionExclude,
			wantGlobal: DefaultChangeDetectionTreatAsGlobal,
		},
		{
			name:       "set list replaces default",
			yaml:       "change_detection:\n  exclude: [docs/**]\n",
			wantExcl:   []string{"docs/**"},
			wantGlobal: DefaultChangeDetectionTreatAsGlobal,
		},
		{
			name:       "empty list disables default",
			yaml:       "change_detection:\n  treat_as_global: []\n",
			wantExcl:   DefaultChangeDetectionExclude,
			wantGlobal: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cfg HiveProjectConfig
			if err := yaml.Unmarshal([]byte(tt.yaml), &cfg); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if got := cfg.ChangeDetection.ExcludePatterns(); !slices.Equal(got, tt.wantExcl) {
				t.Errorf("ExcludePatterns() = %v, want %v", got, tt.wantExcl)
			}
			if got := cfg.ChangeDetection.TreatAsGlobalPatterns(); !slices.Equal(got, tt.wantGlobal) {
				t.Errorf("TreatAsGlobalPatterns() = %v, want %v", got, tt.wantGlobal)
			}
		})
	}
}
