package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/ContainerHive/ContainerHive/pkg/build"
	"github.com/ContainerHive/ContainerHive/pkg/changes"
)

func TestValidateChangesFormat(t *testing.T) {
	tests := []struct {
		format  string
		wantErr bool
	}{
		{format: "text"},
		{format: "json"},
		{format: "yaml", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			err := validateChangesFormat(tt.format)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateChangesFormat(%q) error = %v, wantErr %v", tt.format, err, tt.wantErr)
			}
		})
	}
}

func TestReadChangedFiles(t *testing.T) {
	got, err := readChangedFiles(strings.NewReader("images/a/Dockerfile\n\n  README.md  \n"), "-")
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"images/a/Dockerfile", "README.md"}; !slices.Equal(got, want) {
		t.Errorf("readChangedFiles() = %v, want %v", got, want)
	}

	if _, err := readChangedFiles(nil, filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Error("readChangedFiles() on missing file error = nil, want error")
	}
}

// writeChangesProject creates base -> python.
func writeChangesProject(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"hive.yml":                 "platforms: [linux/amd64]\n",
		"images/base/image.yml":    "tags:\n  - name: \"1\"\n",
		"images/base/Dockerfile":   "FROM ubuntu:24.04\n",
		"images/python/image.yml":  "tags:\n  - name: \"3\"\n",
		"images/python/Dockerfile": "FROM __hive__/base:1\n",
	}
	for name, content := range files {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestChangesCommand(t *testing.T) {
	root := writeChangesProject(t)

	tests := []struct {
		name    string
		args    []string
		stdin   string
		want    string
		wantErr bool
	}{
		{name: "cascade", args: []string{"--files-from", "-"}, stdin: "images/base/Dockerfile\n", want: "base\npython\n"},
		{name: "leaf", args: []string{"--files-from", "-"}, stdin: "images/python/Dockerfile\n", want: "python\n"},
		{name: "excluded", args: []string{"--files-from", "-"}, stdin: "README.md\n", want: ""},
		{name: "needs base or files", args: nil, wantErr: true},
		{name: "not both", args: []string{"--base", "main", "--files-from", "-"}, wantErr: true},
		{name: "bad unmatched", args: []string{"--files-from", "-", "--unmatched", "some"}, wantErr: true},
		{name: "bad format", args: []string{"--files-from", "-", "--format", "yaml"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			app := NewApp()
			app.Writer = &out
			app.Reader = strings.NewReader(tt.stdin)

			args := append([]string{"ch", "--project", root, "changes"}, tt.args...)
			err := app.Run(t.Context(), args)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Run() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && out.String() != tt.want {
				t.Errorf("output = %q, want %q", out.String(), tt.want)
			}
		})
	}
}

func TestChangesCommandWrite(t *testing.T) {
	root := writeChangesProject(t)

	var out bytes.Buffer
	app := NewApp()
	app.Writer = &out
	app.Reader = strings.NewReader("images/python/Dockerfile\n")
	args := []string{"ch", "--project", root, "changes", "--files-from", "-", "--format", "json", "--write"}
	if err := app.Run(t.Context(), args); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `"affected": [`) {
		t.Errorf("json output = %q, want an affected list", out.String())
	}

	set, err := changes.Load(filepath.Join(root, "dist"))
	if err != nil {
		t.Fatal(err)
	}
	if set == nil || !slices.Equal(set.Affected, []string{"python"}) {
		t.Errorf("stored set = %+v, want python affected", set)
	}
}

func TestLimitToChanges(t *testing.T) {
	affected := &changes.Set{Affected: []string{"pre-commit", "python"}}

	tests := []struct {
		name     string
		set      *changes.Set
		filters  []build.Filter
		want     []build.Filter
		wantSkip bool
	}{
		{name: "no change set keeps filters", set: nil, filters: []build.Filter{{ImageName: "base"}}, want: []build.Filter{{ImageName: "base"}}},
		{name: "no change set keeps empty filters", set: nil, filters: nil, want: nil},
		{name: "empty filters become affected images", set: affected, filters: nil, want: []build.Filter{{ImageName: "pre-commit"}, {ImageName: "python"}}},
		{name: "unaffected filter dropped", set: affected, filters: []build.Filter{{ImageName: "base"}, {ImageName: "python", TagName: "3"}}, want: []build.Filter{{ImageName: "python", TagName: "3"}}},
		{name: "tag-only filter expands", set: affected, filters: []build.Filter{{TagName: "1"}}, want: []build.Filter{{ImageName: "pre-commit", TagName: "1"}, {ImageName: "python", TagName: "1"}}},
		{name: "nothing affected skips", set: &changes.Set{Affected: []string{}}, filters: nil, wantSkip: true},
		{name: "only unaffected filters skip", set: affected, filters: []build.Filter{{ImageName: "base"}}, wantSkip: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dist := t.TempDir()
			if tt.set != nil {
				if err := changes.Write(dist, tt.set); err != nil {
					t.Fatal(err)
				}
			}
			got, _, skip, err := limitToChanges(dist, tt.filters)
			if err != nil {
				t.Fatalf("limitToChanges() error = %v", err)
			}
			if skip != tt.wantSkip {
				t.Errorf("skip = %v, want %v", skip, tt.wantSkip)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("filters = %v, want %v", got, tt.want)
			}
		})
	}
}
