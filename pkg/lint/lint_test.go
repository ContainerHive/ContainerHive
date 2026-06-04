package lint

import (
	"strings"
	"testing"

	"github.com/ContainerHive/ContainerHive/pkg/model"
	gohadolint "github.com/timo-reymann/go-hadolint"
)

func gohadolintFinding(code, level string, line, column int, message string) gohadolint.Finding {
	return gohadolint.Finding{
		Code:    code,
		Level:   level,
		Line:    line,
		Column:  column,
		Message: message,
		File:    "Dockerfile",
	}
}

func TestSubstituteHiveParent(t *testing.T) {
	in := []byte("FROM __hive_parent__\nCOPY --from=__hive_parent__ /a /a\n")
	out := SubstituteHiveParent(in, "__hive__/myimg:1.0")
	want := "FROM __hive__/myimg:1.0\nCOPY --from=__hive__/myimg:1.0 /a /a\n"
	if string(out) != want {
		t.Errorf("got %q, want %q", out, want)
	}

	unchanged := []byte("FROM ubuntu:24.04\n")
	if got := SubstituteHiveParent(unchanged, "__hive__/x:1"); string(got) != string(unchanged) {
		t.Errorf("content without placeholder must be unchanged, got %q", got)
	}

	if got := SubstituteHiveParent(in, ""); string(got) != string(in) {
		t.Errorf("empty parentRef must leave content unchanged, got %q", got)
	}
}

func TestPickReferenceTag(t *testing.T) {
	if got := PickReferenceTag(nil); got != "hive-parent" {
		t.Errorf("empty tags: got %q, want hive-parent", got)
	}
	tags := map[string]*model.Tag{
		"1.27": {Name: "1.27"},
		"1.25": {Name: "1.25"},
		"1.26": {Name: "1.26"},
	}
	if got := PickReferenceTag(tags); got != "1.25" {
		t.Errorf("lexicographically first tag: got %q, want 1.25", got)
	}
}

func TestFormatLevel(t *testing.T) {
	cases := []struct {
		level    string
		color    bool
		want     string
		wantTags bool
	}{
		{level: "error", color: false, want: "ERROR"},
		{level: "warning", color: false, want: "WARNING"},
		{level: "error", color: true, want: ansiBrightRed + "ERROR" + ansiReset, wantTags: true},
		{level: "warning", color: true, want: ansiBrightYellow + "WARNING" + ansiReset, wantTags: true},
		{level: "info", color: true, want: ansiBrightCyan + "INFO" + ansiReset, wantTags: true},
		{level: "style", color: true, want: ansiFaint + "STYLE" + ansiReset, wantTags: true},
		// Unknown levels stay plain even with color enabled so we don't emit
		// a stray escape sequence for a future hadolint severity.
		{level: "wat", color: true, want: "WAT"},
	}
	for _, tc := range cases {
		t.Run(tc.level, func(t *testing.T) {
			got := FormatLevel(tc.level, tc.color)
			if got != tc.want {
				t.Errorf("FormatLevel(%q, %v) = %q, want %q", tc.level, tc.color, got, tc.want)
			}
			hasEsc := strings.Contains(got, "\x1b[")
			if hasEsc != tc.wantTags {
				t.Errorf("FormatLevel(%q, %v): escape presence = %v, want %v", tc.level, tc.color, hasEsc, tc.wantTags)
			}
		})
	}
}

func TestResolveConfig(t *testing.T) {
	ignoredSeed := []string{"DL3000"}
	cases := []struct {
		name       string
		project    *model.LintConfig
		cliThresh  string
		wantThresh string
	}{
		{name: "no config no flag", wantThresh: "error"},
		{name: "cli flag only", cliThresh: "warning", wantThresh: "warning"},
		{name: "project only", project: &model.LintConfig{FailureThreshold: "info"}, wantThresh: "info"},
		{name: "cli overrides project", project: &model.LintConfig{FailureThreshold: "info"}, cliThresh: "warning", wantThresh: "warning"},
		{name: "project no threshold", project: &model.LintConfig{Ignored: ignoredSeed}, wantThresh: "error"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ResolveConfig(tc.project, tc.cliThresh)
			if got == nil {
				t.Fatalf("expected non-nil config")
			}
			if got.FailureThreshold != tc.wantThresh {
				t.Errorf("FailureThreshold = %q, want %q", got.FailureThreshold, tc.wantThresh)
			}
			if tc.project != nil && tc.project.FailureThreshold == "info" && got == tc.project {
				t.Errorf("ResolveConfig returned the same pointer; expected a copy")
			}
		})
	}
}
