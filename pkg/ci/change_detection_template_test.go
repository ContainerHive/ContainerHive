package ci

import (
	"slices"
	"strings"
	"testing"

	"github.com/ContainerHive/ContainerHive/pkg/model"
)

// changeDetectionProject has app depending on base via depends_on.
func changeDetectionProject(options map[string]string) *model.ContainerHiveProject {
	return &model.ContainerHiveProject{
		Config: model.HiveProjectConfig{
			Platforms:       []string{"linux/amd64"},
			TemplateOptions: options,
			ChangeDetection: &model.ChangeDetectionConfig{TreatAsGlobal: []string{"hive.yml", "ci/**"}},
		},
		ImagesByName: map[string][]*model.Image{
			"base": {{Name: "base", Identifier: "base", Tags: map[string]*model.Tag{"1": {Name: "1"}}}},
			"app":  {{Name: "app", Identifier: "app", DependsOn: []string{"base"}, Tags: map[string]*model.Tag{"1": {Name: "1"}}}},
		},
	}
}

func renderChangeDetection(t *testing.T, provider string, options map[string]string, projectPath string) string {
	t.Helper()
	ctx, err := BuildCIContext(changeDetectionProject(options), false)
	if err != nil {
		t.Fatal(err)
	}
	ctx.ProjectPath = projectPath
	out, err := Generate(provider, ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}

func TestBuildCIContext_ChangePaths(t *testing.T) {
	ctx, err := BuildCIContext(changeDetectionProject(nil), false)
	if err != nil {
		t.Fatal(err)
	}
	if !ctx.ChangeDetection {
		t.Fatal("ChangeDetection = false, want enabled by default")
	}
	want := map[string][]string{
		"base": {"images/base/**/*", "hive.yml", "ci/**/*"},
		"app":  {"images/app/**/*", "images/base/**/*", "hive.yml", "ci/**/*"},
	}
	for _, img := range ctx.Images {
		if !slices.Equal(img.ChangePaths, want[img.Name]) {
			t.Errorf("ChangePaths[%s] = %v, want %v", img.Name, img.ChangePaths, want[img.Name])
		}
	}
}

func TestGitlabTemplate_ChangeDetection(t *testing.T) {
	out := renderChangeDetection(t, "gitlab", nil, "sub")

	for _, want := range []string{
		`compare_to: refs/heads/$CI_DEFAULT_BRANCH`,
		`- "sub/images/app/**/*"`,
		`- "sub/images/base/**/*"`,
		`- "sub/ci/**/*"`,
		"    - job: manifest-base\n      optional: true",
		"    - job: build-app-amd64\n      optional: true",
		`GIT_DEPTH: "0"`,
		`changes --base "refs/remotes/origin/$CI_DEFAULT_BRANCH" --unmatched ignore --strict --write`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("rendered pipeline is missing %q", want)
		}
	}

	// base must not rebuild for an app change.
	base := out[strings.Index(out, "build-base-amd64:"):strings.Index(out, "manifest-base:")]
	if strings.Contains(base, "images/app/") {
		t.Errorf("build-base-amd64 rules include the app directory:\n%s", base)
	}
}

func TestTemplates_ChangeDetectionDisabled(t *testing.T) {
	for _, provider := range []string{"gitlab"} {
		t.Run(provider, func(t *testing.T) {
			out := renderChangeDetection(t, provider, map[string]string{"ci_change_detection": "false"}, "")
			for _, unwanted := range []string{"compare_to", "optional: true", "GIT_DEPTH", " changes --base"} {
				if strings.Contains(out, unwanted) {
					t.Errorf("disabled change detection still renders %q", unwanted)
				}
			}
		})
	}
}
