package ci

import (
	"strings"
	"testing"
)

func multiPlatformTemplateOutput(t *testing.T, provider string, artifacts bool) string {
	t.Helper()
	return platformTemplateOutput(t, provider, artifacts, []string{"linux/amd64", "linux/arm64"})
}

func platformTemplateOutput(t *testing.T, provider string, artifacts bool, platforms []string) string {
	t.Helper()

	project := singleImageProjectForTemplate()
	project.Config.Platforms = platforms
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

func TestGitlabTemplate_DindService(t *testing.T) {
	tests := []struct {
		name    string
		options map[string]string
		want    string
	}{
		{
			name: "default is the rootless helper image",
			want: "- name: " + DefaultDindImage + "\n      alias: dind\n      command: [\"dockerd\", \"--host=tcp://0.0.0.0:2375\"]",
		},
		{
			name:    "ci_dind_image overrides the image",
			options: map[string]string{"ci_dind_image": "docker:dind"},
			want:    "- name: docker:dind\n      alias: dind",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			project := singleImageProjectForTemplate()
			project.Config.TemplateOptions = tt.options
			ctx, err := BuildCIContext(project, false)
			if err != nil {
				t.Fatal(err)
			}
			output, err := Generate("gitlab", ctx, "")
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(output), tt.want) {
				t.Errorf("expected GitLab output to contain %q, got:\n%s", tt.want, output)
			}
		})
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

var multiComponentPlatforms = []string{"linux/arm/v7", "linux/arm64/v8", "windows/amd64"}

func TestGithubTemplate_MultiComponentPlatforms(t *testing.T) {
	rendered := platformTemplateOutput(t, "github", true, multiComponentPlatforms)

	for _, expected := range []string{
		"build-app-linux-arm-v7:\n    runs-on: ubuntu-24.04\n",
		"build-app-linux-arm64-v8:\n    runs-on: ubuntu-24.04-arm\n",
		"build-app-windows-amd64:\n    runs-on: ubuntu-24.04\n",
		"test-app-linux-arm-v7:\n    runs-on: ubuntu-24.04\n    needs:\n      - build-app-linux-arm-v7",
		"test-app-linux-arm64-v8:\n    runs-on: ubuntu-24.04-arm\n",
		"build app --platform linux/arm/v7 --build-id",
		"test app --platform linux/arm64/v8 --build-id",
		"name: app-linux-arm-v7\n",
		"path: dist/app/*/linux-arm-v7/",
		"path: dist/app/*/linux-arm64-v8/",
		"path: dist/app/*/windows-amd64/",
	} {
		if !strings.Contains(rendered, expected) {
			t.Errorf("expected GitHub output to contain %q, got:\n%s", expected, rendered)
		}
	}

	for _, unexpected := range []string{"-linux/", "-windows/", "linux-linux"} {
		if strings.Contains(rendered, unexpected) {
			t.Errorf("expected no raw platform in identifiers (%q), got:\n%s", unexpected, rendered)
		}
	}
}

func TestGitlabTemplate_MultiComponentPlatforms(t *testing.T) {
	rendered := platformTemplateOutput(t, "gitlab", false, multiComponentPlatforms)

	for _, expected := range []string{
		".build-linux-arm-v7:\n  extends: .build",
		".test-linux-arm-v7:\n  extends: .test",
		"build-app-linux-arm-v7:\n  extends: .build-linux-arm-v7",
		"test-app-linux-arm-v7:\n  extends: .test-linux-arm-v7",
		"- build-app-linux-arm-v7",
		"PLATFORM: linux/arm/v7",
		"PLATFORM: windows/amd64",
	} {
		if !strings.Contains(rendered, expected) {
			t.Errorf("expected GitLab output to contain %q, got:\n%s", expected, rendered)
		}
	}

	for _, unexpected := range []string{"-linux/", "-windows/"} {
		if strings.Contains(rendered, unexpected) {
			t.Errorf("expected no raw platform in identifiers (%q), got:\n%s", unexpected, rendered)
		}
	}
}
