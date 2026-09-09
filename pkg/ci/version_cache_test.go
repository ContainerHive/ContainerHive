package ci

import (
	"strings"
	"testing"

	"github.com/ContainerHive/ContainerHive/pkg/model"
)

func singleImageProjectWithTagRange() *model.ContainerHiveProject {
	img := &model.Image{
		Name: "app",
		Tags: map[string]*model.Tag{"1.0": {Name: "1.0"}},
		TagRanges: []*model.TagRange{
			{TagName: "{{.major}}", Source: &model.SourceConfig{Type: "registry", Image: "library/app"}},
		},
	}
	return &model.ContainerHiveProject{
		Config:       model.HiveProjectConfig{Platforms: []string{"linux/amd64"}},
		ImagesByName: map[string][]*model.Image{"app": {img}},
	}
}

func TestBuildCIContext_HasTagRanges(t *testing.T) {
	t.Run("false without tag_ranges", func(t *testing.T) {
		ctx, err := BuildCIContext(singleImageProjectForTemplate(), false)
		if err != nil {
			t.Fatal(err)
		}
		if ctx.HasTagRanges {
			t.Error("expected HasTagRanges to be false for a project without tag_ranges")
		}
		if ctx.TagRangeUnits != 0 {
			t.Errorf("expected TagRangeUnits to be 0, got %d", ctx.TagRangeUnits)
		}
	})

	t.Run("true with tag_ranges", func(t *testing.T) {
		ctx, err := BuildCIContext(singleImageProjectWithTagRange(), false)
		if err != nil {
			t.Fatal(err)
		}
		if !ctx.HasTagRanges {
			t.Error("expected HasTagRanges to be true for a project with tag_ranges")
		}
		if ctx.TagRangeUnits == 0 {
			t.Error("expected TagRangeUnits to be non-zero")
		}
	})
}

func TestCIContext_VersionCacheDir(t *testing.T) {
	t.Run("no project path", func(t *testing.T) {
		ctx := &CIContext{TemplateOptions: map[string]string{"ci_version_cache_dir": ".ch-cache"}}
		if got := ctx.VersionCacheDir(); got != ".ch-cache" {
			t.Errorf("got %q, want .ch-cache", got)
		}
	})
	t.Run("with project path", func(t *testing.T) {
		ctx := &CIContext{
			ProjectPath:     "subdir",
			TemplateOptions: map[string]string{"ci_version_cache_dir": ".ch-cache"},
		}
		if got := ctx.VersionCacheDir(); got != "subdir/.ch-cache" {
			t.Errorf("got %q, want subdir/.ch-cache", got)
		}
	})
}

func TestGithubTemplate_NoVersionCacheStepsWithoutTagRanges(t *testing.T) {
	ctx, err := BuildCIContext(singleImageProjectForTemplate(), false)
	if err != nil {
		t.Fatal(err)
	}
	out, err := Generate("github", ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	rendered := string(out)
	if strings.Contains(rendered, "Restore version cache") {
		t.Errorf("expected no version cache step for a project without tag_ranges, got:\n%s", rendered)
	}
	if strings.Contains(rendered, "CONTAINER_HIVE_CACHE_DIR") {
		t.Errorf("expected no CONTAINER_HIVE_CACHE_DIR for a project without tag_ranges, got:\n%s", rendered)
	}
}

func TestGithubTemplate_VersionCacheStepsWithTagRanges(t *testing.T) {
	ctx, err := BuildCIContext(singleImageProjectWithTagRange(), false)
	if err != nil {
		t.Fatal(err)
	}
	out, err := Generate("github", ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	rendered := string(out)

	if strings.Count(rendered, "Restore version cache") != 2 {
		t.Errorf("expected 2 version cache restore steps (generate + lint), got:\n%s", rendered)
	}
	if !strings.Contains(rendered, "actions/cache@"+actionsCacheVersionForTest()) {
		t.Errorf("expected the pinned actions/cache version, got:\n%s", rendered)
	}
	if !strings.Contains(rendered, `path: .ch-cache`) {
		t.Errorf("expected the default version cache dir, got:\n%s", rendered)
	}
	if !strings.Contains(rendered, `CONTAINER_HIVE_CACHE_DIR="$GITHUB_WORKSPACE/.ch-cache"`) {
		t.Errorf("expected CONTAINER_HIVE_CACHE_DIR set relative to $GITHUB_WORKSPACE, got:\n%s", rendered)
	}
	if !strings.Contains(rendered, "tag_ranges resolved to") {
		t.Errorf("expected the resolved-unit-count header comment, got:\n%s", rendered)
	}
}

func TestGithubTemplate_VersionCacheDisabledByOption(t *testing.T) {
	project := singleImageProjectWithTagRange()
	project.Config.TemplateOptions = map[string]string{"ci_version_cache": "false"}
	ctx, err := BuildCIContext(project, false)
	if err != nil {
		t.Fatal(err)
	}
	out, err := Generate("github", ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	rendered := string(out)
	if strings.Contains(rendered, "Restore version cache") {
		t.Errorf("expected no version cache step when ci_version_cache is false, got:\n%s", rendered)
	}
}

func TestGitlabTemplate_NoVersionCacheWithoutTagRanges(t *testing.T) {
	ctx, err := BuildCIContext(singleImageProjectForTemplate(), false)
	if err != nil {
		t.Fatal(err)
	}
	out, err := Generate("gitlab", ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(out), "ch-versions") {
		t.Errorf("expected no version cache block for a project without tag_ranges, got:\n%s", out)
	}
}

func TestGitlabTemplate_VersionCacheWithTagRanges(t *testing.T) {
	ctx, err := BuildCIContext(singleImageProjectWithTagRange(), false)
	if err != nil {
		t.Fatal(err)
	}
	out, err := Generate("gitlab", ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	rendered := string(out)

	if strings.Count(rendered, "key: ch-versions") != 2 {
		t.Errorf("expected 2 cache blocks (generate + lint), got:\n%s", rendered)
	}
	if !strings.Contains(rendered, "CONTAINER_HIVE_CACHE_DIR: .ch-cache") {
		t.Errorf("expected CONTAINER_HIVE_CACHE_DIR variable, got:\n%s", rendered)
	}
	if !strings.Contains(rendered, "policy: pull-push") {
		t.Errorf("expected the generate job's cache policy to be pull-push, got:\n%s", rendered)
	}
	if !strings.Contains(rendered, "policy: pull") {
		t.Errorf("expected the lint job's cache policy to be pull-only, got:\n%s", rendered)
	}
	if !strings.Contains(rendered, "tag_ranges resolved to") {
		t.Errorf("expected the resolved-unit-count header comment, got:\n%s", rendered)
	}
}

// actionsCacheVersionForTest avoids importing internal/actions just to
// avoid a magic string mismatch if the pinned version is bumped by renovate;
// tests should track behavior, not the exact pinned tag, so this only
// checks the option resolves to *some* non-empty version.
func actionsCacheVersionForTest() string {
	ctx, err := BuildCIContext(singleImageProjectWithTagRange(), false)
	if err != nil {
		panic(err)
	}
	return ctx.TemplateOptions["actions_cache_version"]
}
