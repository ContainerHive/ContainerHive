package source

import (
	"context"
	"fmt"
	"os"

	"github.com/ContainerHive/ContainerHive/pkg/model"
)

// githubAPIBase is overridden by tests to point at an httptest server.
var githubAPIBase = "https://api.github.com"

// maxGitHubPages caps pagination so a repository with thousands of
// releases/tags can't turn one Fetch into an unbounded crawl; selection
// only ever wants the newest entries, which sort first.
const maxGitHubPages = 3

const githubPageSize = 100

// GitHubSource lists a GitHub repository's releases (default) or tags.
type GitHubSource struct{}

func (GitHubSource) Name() string { return "github" }

func (GitHubSource) Validate(cfg *model.SourceConfig) error {
	if cfg.Repo == "" {
		return fmt.Errorf("source type %q requires repo", "github")
	}
	if cfg.Kind != "" && cfg.Kind != "releases" && cfg.Kind != "tags" {
		return fmt.Errorf("source type %q: kind must be \"releases\" or \"tags\", got %q", "github", cfg.Kind)
	}
	if cfg.URL != "" || cfg.Transform != "" || cfg.Image != "" {
		return fmt.Errorf("source type %q does not accept url/transform/image", "github")
	}
	return nil
}

func (GitHubSource) Descriptor(cfg *model.SourceConfig) string {
	kind := cfg.Kind
	if kind == "" {
		kind = "releases"
	}
	return fmt.Sprintf("github (%s, %s)", cfg.Repo, kind)
}

func (GitHubSource) CacheKeyParts(cfg *model.SourceConfig) []string {
	return []string{"github", cfg.Repo, cfg.Kind}
}

type githubRelease struct {
	TagName    string `json:"tag_name"`
	Name       string `json:"name"`
	Draft      bool   `json:"draft"`
	Prerelease bool   `json:"prerelease"`
	Published  string `json:"published_at"`
}

type githubTag struct {
	Name   string `json:"name"`
	Commit struct {
		SHA string `json:"sha"`
	} `json:"commit"`
}

func (s GitHubSource) Fetch(ctx context.Context, cfg *model.SourceConfig) ([]Version, error) {
	kind := cfg.Kind
	if kind == "" {
		kind = "releases"
	}

	headers := authHeaders(cfg)
	var versions []Version
	for page := 1; page <= maxGitHubPages; page++ {
		url := fmt.Sprintf("%s/repos/%s/%s?per_page=%d&page=%d", githubAPIBase, cfg.Repo, kind, githubPageSize, page)

		var got []Version
		var err error
		if kind == "tags" {
			got, err = fetchGitHubTags(ctx, url, headers)
		} else {
			got, err = fetchGitHubReleases(ctx, url, headers)
		}
		if err != nil {
			return nil, fmt.Errorf("github source %q: %w", cfg.Repo, err)
		}
		versions = append(versions, got...)
		if len(got) < githubPageSize {
			break
		}
	}
	return versions, nil
}

func fetchGitHubReleases(ctx context.Context, url string, headers map[string]string) ([]Version, error) {
	var releases []githubRelease
	if err := getJSON(ctx, url, headers, &releases); err != nil {
		return nil, err
	}
	versions := make([]Version, 0, len(releases))
	for _, r := range releases {
		if r.Draft {
			continue
		}
		versions = append(versions, Version{
			Raw:        r.TagName,
			Prerelease: r.Prerelease,
			Extra: map[string]string{
				"name":         r.Name,
				"published_at": r.Published,
			},
		})
	}
	return versions, nil
}

func fetchGitHubTags(ctx context.Context, url string, headers map[string]string) ([]Version, error) {
	var tags []githubTag
	if err := getJSON(ctx, url, headers, &tags); err != nil {
		return nil, err
	}
	versions := make([]Version, len(tags))
	for i, t := range tags {
		versions[i] = Version{Raw: t.Name, Extra: map[string]string{"commit_sha": t.Commit.SHA}}
	}
	return versions, nil
}

// authHeaders builds the Authorization header from token_env, falling back
// to GITHUB_TOKEN then GH_TOKEN. Anonymous access is allowed (rate-limited
// to 60 requests/hour); the token is never read from image.yml directly.
func authHeaders(cfg *model.SourceConfig) map[string]string {
	token := ""
	if cfg.TokenEnv != "" {
		token = os.Getenv(cfg.TokenEnv)
	}
	if token == "" {
		token = os.Getenv("GITHUB_TOKEN")
	}
	if token == "" {
		token = os.Getenv("GH_TOKEN")
	}
	if token == "" {
		return nil
	}
	return map[string]string{"Authorization": "Bearer " + token}
}
