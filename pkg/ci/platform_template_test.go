package ci

import (
	"strings"
	"testing"
)

func multiPlatformTemplateOutput(t *testing.T, provider string, artifacts bool) string {
	t.Helper()

	project := singleImageProjectForTemplate()
	project.Config.Platforms = []string{"linux/amd64", "linux/arm64"}
	ctx, err := BuildCIContext(project, artifacts)
	if err != nil {
		t.Fatal(err)
	}

	output, err := Generate(provider, ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	return string(output)
}

func TestGitlabTemplate_TestJobsPerPlatform(t *testing.T) {
	rendered := multiPlatformTemplateOutput(t, "gitlab", false)

	for _, expected := range []string{
		".test-amd64:\n  extends: .test",
		".test-arm64:\n  extends: .test",
		"test-app-amd64:\n  extends: .test-amd64",
		"test-app-arm64:\n  extends: .test-arm64",
		"PLATFORM: amd64",
		"PLATFORM: arm64",
		"- test-app-amd64",
		"- test-app-arm64",
		"ch test ${IMAGE_NAME} --platform ${PLATFORM} --build-id",
	} {
		if !strings.Contains(rendered, expected) {
			t.Errorf("expected GitLab output to contain %q, got:\n%s", expected, rendered)
		}
	}

	if strings.Contains(rendered, "test-app:\n") {
		t.Errorf("expected architecture-specific GitLab test jobs, got generic test job:\n%s", rendered)
	}
}

func TestGithubTemplate_TestJobsPerPlatform(t *testing.T) {
	rendered := multiPlatformTemplateOutput(t, "github", false)

	for _, expected := range []string{
		"test-app-amd64:\n    runs-on: ubuntu-24.04\n    needs:\n      - build-app-amd64",
		"test-app-arm64:\n    runs-on: ubuntu-24.04-arm\n    needs:\n      - build-app-arm64",
		"test app --platform amd64 --build-id",
		"test app --platform arm64 --build-id",
		"Container Structure Tests - app (amd64)",
		"Container Structure Tests - app (arm64)",
		"- test-app-amd64",
		"- test-app-arm64",
	} {
		if !strings.Contains(rendered, expected) {
			t.Errorf("expected GitHub output to contain %q, got:\n%s", expected, rendered)
		}
	}

	if strings.Contains(rendered, "test-app:\n") {
		t.Errorf("expected architecture-specific GitHub test jobs, got generic test job:\n%s", rendered)
	}
}

func TestGithubTemplate_TestJobsDownloadMatchingPlatformArtifacts(t *testing.T) {
	rendered := multiPlatformTemplateOutput(t, "github", true)
	start := strings.SplitN(rendered, "  test-app-amd64:", 2)
	if len(start) != 2 {
		t.Fatalf("expected amd64 test job, got:\n%s", rendered)
	}
	jobs := strings.SplitN(start[1], "  test-app-arm64:", 2)
	if len(jobs) != 2 {
		t.Fatalf("expected separate amd64 and arm64 test jobs, got:\n%s", rendered)
	}
	arm64Job := strings.SplitN(jobs[1], "  report:", 2)[0]

	if !strings.Contains(jobs[0], "name: app-amd64") {
		t.Errorf("expected amd64 test job to download amd64 artifacts, got:\n%s", jobs[0])
	}
	if strings.Contains(jobs[0], "name: app-arm64") {
		t.Errorf("amd64 test job should not download arm64 artifacts, got:\n%s", jobs[0])
	}
	if !strings.Contains(arm64Job, "name: app-arm64") {
		t.Errorf("expected arm64 test job to download arm64 artifacts, got:\n%s", arm64Job)
	}
	if strings.Contains(arm64Job, "name: app-amd64") {
		t.Errorf("arm64 test job should not download amd64 artifacts, got:\n%s", arm64Job)
	}
}
