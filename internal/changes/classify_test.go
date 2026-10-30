package changes

import (
	"slices"
	"testing"
)

func TestClassify(t *testing.T) {
	imageDirs := map[string][]string{
		"base":       {"images/base"},
		"python":     {"images/python"},
		"tools/lint": {"images/tools/lint"},
		"dotnet":     {"images/dotnet/8", "images/dotnet/9"},
		"node":       {"../shared/images/node"},
	}
	exclude := []string{"README.md", "styles/**", ".pre-commit-config.yaml"}
	global := []string{".pre-commit-config.yaml", "hive.yml"}

	tests := []struct {
		name        string
		files       []string
		unmatched   Unmatched
		wantGlobal  bool
		wantFile    string
		wantChanged []string
	}{
		{name: "image dir", files: []string{"images/python/Dockerfile"}, wantChanged: []string{"python"}},
		{name: "image dir with dot prefix", files: []string{"./images/base/image.yml"}, wantChanged: []string{"base"}},
		{name: "nested image name", files: []string{"images/tools/lint/Dockerfile"}, wantChanged: []string{"tools/lint"}},
		{name: "one name many dirs", files: []string{"images/dotnet/9/Dockerfile"}, wantChanged: []string{"dotnet"}},
		{name: "dir prefix is not a parent", files: []string{"images/python-ci/Dockerfile"}, unmatched: UnmatchedIgnore},
		{name: "excluded root file", files: []string{"README.md"}},
		{name: "excluded folder", files: []string{"styles/report.css"}},
		{name: "global beats exclude", files: []string{".pre-commit-config.yaml"}, wantGlobal: true, wantFile: ".pre-commit-config.yaml"},
		{name: "global config", files: []string{"images/base/Dockerfile", "hive.yml"}, wantGlobal: true, wantFile: "hive.yml", wantChanged: []string{"base"}},
		{name: "unmatched rebuilds all by default", files: []string{"scripts/release.sh"}, wantGlobal: true, wantFile: "scripts/release.sh"},
		{name: "unmatched ignored", files: []string{"scripts/release.sh"}, unmatched: UnmatchedIgnore},
		{name: "blank lines skipped", files: []string{"", "  "}},
		{name: "outside the project ignored", files: []string{"../other/hive.yml"}},
		{name: "image dir outside the project", files: []string{"../shared/images/node/Dockerfile"}, wantChanged: []string{"node"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			unmatched := tt.unmatched
			if unmatched == "" {
				unmatched = UnmatchedAll
			}
			got, err := Classify(tt.files, Rules{
				Exclude:       exclude,
				TreatAsGlobal: global,
				ImageDirs:     imageDirs,
				Unmatched:     unmatched,
			})
			if err != nil {
				t.Fatalf("Classify() error = %v", err)
			}
			if got.Global != tt.wantGlobal {
				t.Errorf("Global = %v, want %v", got.Global, tt.wantGlobal)
			}
			if got.GlobalFile != tt.wantFile {
				t.Errorf("GlobalFile = %q, want %q", got.GlobalFile, tt.wantFile)
			}
			if !slices.Equal(got.Changed, tt.wantChanged) {
				t.Errorf("Changed = %v, want %v", got.Changed, tt.wantChanged)
			}
		})
	}
}

func TestClassifyInvalidPattern(t *testing.T) {
	_, err := Classify([]string{"a"}, Rules{Exclude: []string{"[a"}})
	if err == nil {
		t.Fatal("Classify() error = nil, want invalid pattern error")
	}
}

func TestParseUnmatched(t *testing.T) {
	for _, value := range []string{"all", "ignore"} {
		if _, err := ParseUnmatched(value); err != nil {
			t.Errorf("ParseUnmatched(%q) error = %v", value, err)
		}
	}
	if _, err := ParseUnmatched("some"); err == nil {
		t.Error("ParseUnmatched(\"some\") error = nil, want error")
	}
}

func TestExpand(t *testing.T) {
	// pre-commit depends on python, python depends on base.
	dependents := map[string][]string{
		"base":   {"python"},
		"python": {"pre-commit"},
	}
	next := func(name string) []string { return dependents[name] }

	tests := []struct {
		name  string
		start []string
		want  []string
	}{
		{name: "leaf", start: []string{"pre-commit"}, want: []string{"pre-commit"}},
		{name: "middle", start: []string{"python"}, want: []string{"pre-commit", "python"}},
		{name: "root cascades", start: []string{"base"}, want: []string{"base", "pre-commit", "python"}},
		{name: "empty", start: nil, want: []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Expand(tt.start, next); !slices.Equal(got, tt.want) {
				t.Errorf("Expand() = %v, want %v", got, tt.want)
			}
		})
	}
}
