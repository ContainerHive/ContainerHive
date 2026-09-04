package ci

import (
	"strings"
	"testing"

	"github.com/ContainerHive/ContainerHive/pkg/model"
)

func singleImageProjectForTemplate() *model.ContainerHiveProject {
	return &model.ContainerHiveProject{
		Config: model.HiveProjectConfig{
			Platforms: []string{"linux/amd64"},
		},
		ImagesByName: map[string][]*model.Image{
			"app": {{Name: "app", Tags: map[string]*model.Tag{"1.0": {Name: "1.0"}}}},
		},
	}
}

// TestGitlabTemplate_NoParallelKeyAtDefault confirms the default
// ci_build_shards/ci_test_shards of "1" renders no `parallel:` key at all -
// not `parallel: 1`, which GitLab may reject and which needlessly renames
// the job to "... 1/1".
func TestGitlabTemplate_NoParallelKeyAtDefault(t *testing.T) {
	project := singleImageProjectForTemplate()
	ctx, err := BuildCIContext(project, false)
	if err != nil {
		t.Fatal(err)
	}

	out, err := Generate("gitlab", ctx, "")
	if err != nil {
		t.Fatal(err)
	}

	if strings.Contains(string(out), "parallel:") {
		t.Errorf("expected no parallel: key at the default shard count of 1, got:\n%s", out)
	}
}

// TestGitlabTemplate_ParallelKeyWhenShardsSet confirms build/test jobs get a
// parallel: key equal to the configured shard count when it does not exceed
// the image's shard unit count.
func TestGitlabTemplate_ParallelKeyWhenShardsSet(t *testing.T) {
	// Need ≥5 units so the requested values (3 build, 5 test) are not capped.
	project := &model.ContainerHiveProject{
		Config: model.HiveProjectConfig{
			Platforms: []string{"linux/amd64"},
			TemplateOptions: map[string]string{
				"ci_build_shards": "3",
				"ci_test_shards":  "5",
			},
		},
		ImagesByName: map[string][]*model.Image{
			"app": {{
				Name: "app",
				Tags: map[string]*model.Tag{
					"1.0": {Name: "1.0"}, "2.0": {Name: "2.0"}, "3.0": {Name: "3.0"},
					"4.0": {Name: "4.0"}, "5.0": {Name: "5.0"},
				},
			}},
		},
	}
	ctx, err := BuildCIContext(project, false)
	if err != nil {
		t.Fatal(err)
	}

	out, err := Generate("gitlab", ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	rendered := string(out)

	if !strings.Contains(rendered, "parallel: 3") {
		t.Errorf("expected build job to have parallel: 3, got:\n%s", rendered)
	}
	if !strings.Contains(rendered, "parallel: 5") {
		t.Errorf("expected test job to have parallel: 5, got:\n%s", rendered)
	}
}

// TestGitlabTemplate_ParallelCappedAtUnitCount confirms that when the
// configured shard count exceeds the image's shard unit count, parallel is
// capped at the unit count instead of emitting an over-provisioned value.
func TestGitlabTemplate_ParallelCappedAtUnitCount(t *testing.T) {
	// 1 tag, 0 variants → 1 unit → parallel should not appear at all.
	project := &model.ContainerHiveProject{
		Config: model.HiveProjectConfig{
			Platforms: []string{"linux/amd64"},
			TemplateOptions: map[string]string{
				"ci_build_shards": "6",
				"ci_test_shards":  "4",
			},
		},
		ImagesByName: map[string][]*model.Image{
			"app": {{Name: "app", Tags: map[string]*model.Tag{"1.0": {Name: "1.0"}}}},
		},
	}
	ctx, err := BuildCIContext(project, false)
	if err != nil {
		t.Fatal(err)
	}

	out, err := Generate("gitlab", ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	rendered := string(out)

	if strings.Contains(rendered, "parallel:") {
		t.Errorf("expected no parallel: key when unit count is 1, got:\n%s", rendered)
	}
}

// TestGitlabTemplate_ParallelCappedToUnitCount confirms that when the
// configured shard count exceeds the unit count but is not 1, parallel is
// capped to the unit count.
func TestGitlabTemplate_ParallelCappedToUnitCount(t *testing.T) {
	// 2 tags, 0 variants → 2 units → requested 6 should be capped to 2.
	project := &model.ContainerHiveProject{
		Config: model.HiveProjectConfig{
			Platforms: []string{"linux/amd64"},
			TemplateOptions: map[string]string{
				"ci_build_shards": "6",
				"ci_test_shards":  "5",
			},
		},
		ImagesByName: map[string][]*model.Image{
			"app": {{
				Name: "app",
				Tags: map[string]*model.Tag{"1.0": {Name: "1.0"}, "2.0": {Name: "2.0"}},
			}},
		},
	}
	ctx, err := BuildCIContext(project, false)
	if err != nil {
		t.Fatal(err)
	}

	out, err := Generate("gitlab", ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	rendered := string(out)

	if !strings.Contains(rendered, "parallel: 2") {
		t.Errorf("expected both build and test jobs to have parallel: 2, got:\n%s", rendered)
	}
}

// TestGitlabTemplate_NoCPEFlagAtDefault confirms the default
// ci_sbom_generate_cpes of "true" renders the sbom job without --no-cpe.
func TestGitlabTemplate_NoCPEFlagAtDefault(t *testing.T) {
	project := singleImageProjectForTemplate()
	ctx, err := BuildCIContext(project, false)
	if err != nil {
		t.Fatal(err)
	}

	out, err := Generate("gitlab", ctx, "")
	if err != nil {
		t.Fatal(err)
	}

	if strings.Contains(string(out), "--no-cpe") {
		t.Errorf("expected no --no-cpe flag at the default ci_sbom_generate_cpes, got:\n%s", out)
	}
}

// TestGitlabTemplate_NoCPEFlagWhenDisabled confirms setting
// ci_sbom_generate_cpes to "false" adds --no-cpe to the sbom job to keep
// generated SBOMs under GitLab's artifact size limits.
func TestGitlabTemplate_NoCPEFlagWhenDisabled(t *testing.T) {
	project := singleImageProjectForTemplate()
	project.Config.TemplateOptions = map[string]string{
		"ci_sbom_generate_cpes": "false",
	}
	ctx, err := BuildCIContext(project, false)
	if err != nil {
		t.Fatal(err)
	}

	out, err := Generate("gitlab", ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	rendered := string(out)

	if !strings.Contains(rendered, `sbom ${IMAGE_NAME} --platform ${PLATFORM} --build-id "$SNAPSHOT_ID" --no-cpe`) {
		t.Errorf("expected sbom job to have --no-cpe appended, got:\n%s", rendered)
	}
}
