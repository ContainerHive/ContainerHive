package model

import (
	"strings"
	"testing"
)

func TestParseChiselCutSpec(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		check   func(t *testing.T, spec *ChiselCutSpec)
		wantErr string
	}{
		{
			name: "branch release with slices",
			yaml: `
release: ubuntu-24.04
slices:
  - base-files_base
  - libc6_libs
`,
			check: func(t *testing.T, spec *ChiselCutSpec) {
				if spec.Release.Name != "ubuntu-24.04" {
					t.Errorf("release name = %q, want ubuntu-24.04", spec.Release.Name)
				}
				if spec.Release.IsGit() {
					t.Error("branch release must not report git")
				}
				if len(spec.Slices) != 2 || spec.Slices[0] != "base-files_base" {
					t.Errorf("slices = %v, want [base-files_base libc6_libs]", spec.Slices)
				}
			},
		},
		{
			name: "git release with ref",
			yaml: `
release:
  git: https://example.com/chisel-releases.git
  ref: "1.0"
slices:
  - hello_bins
`,
			check: func(t *testing.T, spec *ChiselCutSpec) {
				if !spec.Release.IsGit() {
					t.Error("git release must report git")
				}
				if spec.Release.Git != "https://example.com/chisel-releases.git" {
					t.Errorf("git = %q", spec.Release.Git)
				}
				if spec.Release.GitRef != "1.0" {
					t.Errorf("git ref = %q, want 1.0", spec.Release.GitRef)
				}
			},
		},
		{
			name: "git release without ref",
			yaml: `
release:
  git: https://example.com/chisel-releases.git
slices:
  - hello_bins
`,
			check: func(t *testing.T, spec *ChiselCutSpec) {
				if spec.Release.GitRef != "" {
					t.Errorf("git ref = %q, want empty", spec.Release.GitRef)
				}
			},
		},
		{
			name: "release must not use both forms together",
			yaml: "release: ubuntu-24.04\nslices:\n  - hello_bins\n",
			check: func(t *testing.T, spec *ChiselCutSpec) {
				spec.Release.Git = "https://example.com/r.git"
				err := spec.Validate()
				if err == nil || !strings.Contains(err.Error(), "must set either") {
					t.Errorf("expected both-forms error, got: %v", err)
				}
			},
		},
		{
			name: "ignore conditions",
			yaml: `
release: ubuntu-20.04
slices:
  - base-files_base
ignore: [unmaintained]
`,
			check: func(t *testing.T, spec *ChiselCutSpec) {
				if len(spec.Ignore) != 1 || spec.Ignore[0] != "unmaintained" {
					t.Errorf("ignore = %v, want [unmaintained]", spec.Ignore)
				}
			},
		},
		{
			name:    "missing release fails",
			yaml:    "slices:\n  - hello_bins\n",
			wantErr: "chisel release is required",
		},
		{
			name:    "empty slices fail",
			yaml:    "release: ubuntu-24.04\nslices: []\n",
			wantErr: "chisel slices are required",
		},
		{
			name:    "unknown ignore value fails",
			yaml:    "release: ubuntu-24.04\nslices:\n  - hello_bins\nignore: [broken]\n",
			wantErr: "unknown chisel ignore condition",
		},
		{
			name:    "unknown release field fails",
			yaml:    "release:\n  git: https://example.com/r.git\n  tag: v1\nslices:\n  - hello_bins\n",
			wantErr: "unknown chisel release field",
		},
		{
			name:    "unknown top-level field fails",
			yaml:    "release: ubuntu-24.04\nslices:\n  - hello_bins\narch: amd64\n",
			wantErr: "invalid chisel.yaml cut spec",
		},
		{
			name:    "broken yaml fails",
			yaml:    ":::\n",
			wantErr: "invalid chisel.yaml cut spec",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			spec, err := ParseChiselCutSpec([]byte(tc.yaml))
			if tc.wantErr != "" {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Errorf("error %q does not contain %q", err.Error(), tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			tc.check(t, spec)
		})
	}
}
