package changes

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/ContainerHive/ContainerHive/pkg/discovery"
	"github.com/ContainerHive/ContainerHive/pkg/model"
)

// writeProject creates base -> python -> pre-commit plus an unrelated node
// image.
func writeProject(t *testing.T, hiveYml string) *model.ContainerHiveProject {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"hive.yml":                     hiveYml,
		"images/base/image.yml":        "tags:\n  - name: \"1\"\n",
		"images/base/Dockerfile":       "FROM ubuntu:24.04\n",
		"images/python/image.yml":      "tags:\n  - name: \"3.13\"\n",
		"images/python/Dockerfile":     "FROM __hive__/base:1\n",
		"images/pre-commit/image.yml":  "tags:\n  - name: \"4\"\ndepends_on:\n  - python\n",
		"images/pre-commit/Dockerfile": "FROM ubuntu:24.04\n",
		"images/node/image.yml":        "tags:\n  - name: \"22\"\n",
		"images/node/Dockerfile":       "FROM ubuntu:24.04\n",
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
	project, err := discovery.DiscoverProject(t.Context(), root, discovery.WithSkipTagRanges())
	if err != nil {
		t.Fatalf("DiscoverProject() error = %v", err)
	}
	return project
}

func TestDetect(t *testing.T) {
	project := writeProject(t, "platforms: [linux/amd64]\n")

	tests := []struct {
		name         string
		files        []string
		unmatched    string
		wantGlobal   bool
		wantChanged  []string
		wantAffected []string
	}{
		{name: "leaf", files: []string{"images/pre-commit/Dockerfile"}, wantChanged: []string{"pre-commit"}, wantAffected: []string{"pre-commit"}},
		{name: "middle", files: []string{"images/python/image.yml"}, wantChanged: []string{"python"}, wantAffected: []string{"pre-commit", "python"}},
		{name: "root cascades", files: []string{"images/base/Dockerfile"}, wantChanged: []string{"base"}, wantAffected: []string{"base", "pre-commit", "python"}},
		{name: "default exclude", files: []string{"README.md", "renovate.json"}, wantChanged: []string{}, wantAffected: []string{}},
		{name: "default global", files: []string{".pre-commit-config.yaml"}, wantGlobal: true, wantChanged: []string{}, wantAffected: []string{"base", "node", "pre-commit", "python"}},
		{name: "hive.yml is global", files: []string{"hive.yml"}, wantGlobal: true, wantChanged: []string{}, wantAffected: []string{"base", "node", "pre-commit", "python"}},
		{name: "unmatched rebuilds all", files: []string{"scripts/x.sh"}, wantGlobal: true, wantChanged: []string{}, wantAffected: []string{"base", "node", "pre-commit", "python"}},
		{name: "unmatched ignored", files: []string{"scripts/x.sh"}, unmatched: UnmatchedIgnore, wantChanged: []string{}, wantAffected: []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			set, err := Detect(project, tt.files, Options{Unmatched: tt.unmatched})
			if err != nil {
				t.Fatalf("Detect() error = %v", err)
			}
			if set.Global != tt.wantGlobal {
				t.Errorf("Global = %v, want %v", set.Global, tt.wantGlobal)
			}
			if !slices.Equal(set.Changed, tt.wantChanged) {
				t.Errorf("Changed = %v, want %v", set.Changed, tt.wantChanged)
			}
			if !slices.Equal(set.Affected, tt.wantAffected) {
				t.Errorf("Affected = %v, want %v", set.Affected, tt.wantAffected)
			}
		})
	}
}

func TestDetectConfigOverride(t *testing.T) {
	project := writeProject(t, "change_detection:\n  exclude: [docs/**]\n  treat_as_global: [ci/**]\n")

	set, err := Detect(project, []string{"README.md"}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !set.Global {
		t.Error("README.md is no longer excluded once exclude is overridden, want global")
	}

	set, err = Detect(project, []string{"docs/index.md", ".pre-commit-config.yaml"}, Options{Unmatched: UnmatchedIgnore})
	if err != nil {
		t.Fatal(err)
	}
	if set.Global {
		t.Error("overridden lists must drop the defaults, got global")
	}

	set, err = Detect(project, []string{"ci/pipeline.yml"}, Options{Unmatched: UnmatchedIgnore})
	if err != nil {
		t.Fatal(err)
	}
	if !set.Global {
		t.Error("ci/pipeline.yml matches treat_as_global, want global")
	}
}

func TestImagesLinkPlainDirectory(t *testing.T) {
	project := writeProject(t, "{}\n")
	if got := ImagesLink(project); got != "" {
		t.Errorf("ImagesLink() = %q, want none for a plain images directory", got)
	}
}

func TestDetectInvalidUnmatched(t *testing.T) {
	project := writeProject(t, "{}\n")
	if _, err := Detect(project, nil, Options{Unmatched: "maybe"}); err == nil {
		t.Error("Detect() error = nil, want invalid unmatched error")
	}
}

func TestTriggerDirs(t *testing.T) {
	project := writeProject(t, "{}\n")
	got, err := TriggerDirs(project)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string][]string{
		"base":       {"images/base"},
		"python":     {"images/base", "images/python"},
		"pre-commit": {"images/base", "images/pre-commit", "images/python"},
		"node":       {"images/node"},
	}
	for name, dirs := range want {
		if !slices.Equal(got[name], dirs) {
			t.Errorf("TriggerDirs()[%q] = %v, want %v", name, got[name], dirs)
		}
	}
}

func TestSetIsAffected(t *testing.T) {
	var none *Set
	if !none.IsAffected("base") {
		t.Error("nil set must affect every image")
	}
	set := &Set{Affected: []string{"python"}}
	if !set.IsAffected("python") || set.IsAffected("base") {
		t.Errorf("IsAffected() mismatch for %v", set.Affected)
	}
}

func TestWriteLoad(t *testing.T) {
	dist := filepath.Join(t.TempDir(), "dist")

	loaded, err := Load(dist)
	if err != nil || loaded != nil {
		t.Fatalf("Load() on missing file = %v, %v, want nil, nil", loaded, err)
	}

	set := &Set{Base: "abc", Changed: []string{"python"}, Affected: []string{"pre-commit", "python"}}
	if err := Write(dist, set); err != nil {
		t.Fatal(err)
	}
	loaded, err = Load(dist)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Base != set.Base || !slices.Equal(loaded.Affected, set.Affected) || !slices.Equal(loaded.Changed, set.Changed) {
		t.Errorf("Load() = %+v, want %+v", loaded, set)
	}
}

func TestProjectRelative(t *testing.T) {
	repo := t.TempDir()
	project := filepath.Join(repo, "test-project", "gitlab")
	if err := os.MkdirAll(project, 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := projectRelative(repo, project, []string{"README.md", "test-project/gitlab/images/a/Dockerfile", "test-project/github/hive.yml"})
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"../../README.md", "images/a/Dockerfile", "../github/hive.yml"}; !slices.Equal(got, want) {
		t.Errorf("projectRelative() = %v, want %v", got, want)
	}

	got, err = projectRelative(repo, repo, []string{"README.md"})
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"README.md"}; !slices.Equal(got, want) {
		t.Errorf("projectRelative() at repo root = %v, want %v", got, want)
	}
}

func TestDetectFromGitFallsBackToEverything(t *testing.T) {
	project := writeProject(t, "{}\n")

	set, err := DetectFromGit(project, "main", Options{})
	if err != nil {
		t.Fatalf("DetectFromGit() error = %v", err)
	}
	if !set.Global || set.Base != "main" {
		t.Errorf("DetectFromGit() outside a repository = %+v, want global for base main", set)
	}
	if want := []string{"base", "node", "pre-commit", "python"}; !slices.Equal(set.Affected, want) {
		t.Errorf("Affected = %v, want %v", set.Affected, want)
	}
}

func TestDetectFromGitStrict(t *testing.T) {
	project := writeProject(t, "{}\n")
	if _, err := DetectFromGit(project, "main", Options{Strict: true}); err == nil {
		t.Error("DetectFromGit() with Strict outside a repository error = nil, want error")
	}
}

// TestDetectSymlinkedImages covers projects whose images directory is a
// symlink to a shared directory, as git reports the shared path.
func TestDetectSymlinkedImages(t *testing.T) {
	repo := t.TempDir()
	files := map[string]string{
		"shared/images/base/image.yml":    "tags:\n  - name: \"1\"\n",
		"shared/images/base/Dockerfile":   "FROM ubuntu:24.04\n",
		"shared/images/python/image.yml":  "tags:\n  - name: \"3\"\n",
		"shared/images/python/Dockerfile": "FROM __hive__/base:1\n",
		"gitlab/hive.yml":                 "{}\n",
	}
	for name, content := range files {
		path := filepath.Join(repo, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink("../shared/images", filepath.Join(repo, "gitlab", "images")); err != nil {
		t.Fatal(err)
	}
	project, err := discovery.DiscoverProject(t.Context(), filepath.Join(repo, "gitlab"), discovery.WithSkipTagRanges())
	if err != nil {
		t.Fatal(err)
	}

	if got, want := ImageDirs(project)["base"], []string{"../shared/images/base"}; !slices.Equal(got, want) {
		t.Errorf("ImageDirs()[base] = %v, want %v", got, want)
	}

	if got := ImagesLink(project); got != "images" {
		t.Errorf("ImagesLink() = %q, want images", got)
	}
	retargeted, err := Detect(project, []string{"images"}, Options{Unmatched: UnmatchedIgnore})
	if err != nil {
		t.Fatal(err)
	}
	if !retargeted.Global {
		t.Errorf("retargeting the images symlink must rebuild everything, got %+v", retargeted)
	}

	repoFiles := []string{"shared/images/base/Dockerfile", "github/hive.yml"}
	projectFiles, err := projectRelative(repo, project.RootDir, repoFiles)
	if err != nil {
		t.Fatal(err)
	}
	set, err := Detect(project, projectFiles, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if set.Global {
		t.Errorf("a file of another project must not rebuild everything, got %+v", set)
	}
	if want := []string{"base", "python"}; !slices.Equal(set.Affected, want) {
		t.Errorf("Affected = %v, want %v", set.Affected, want)
	}
}
