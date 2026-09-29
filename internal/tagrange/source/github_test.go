package source

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ContainerHive/ContainerHive/pkg/model"
)

func withGitHubTestServer(t *testing.T, handler http.HandlerFunc) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	prev := githubAPIBase
	githubAPIBase = server.URL
	t.Cleanup(func() { githubAPIBase = prev })
}

func TestGitHubSource_Fetch_Releases(t *testing.T) {
	releases := []githubRelease{
		{TagName: "v1.2.0", Name: "1.2.0", Prerelease: false},
		{TagName: "v1.3.0-rc1", Name: "1.3.0-rc1", Prerelease: true},
		{TagName: "v1.1.0", Name: "1.1.0", Draft: true}, // must be dropped
	}
	withGitHubTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("page") != "1" {
			_ = json.NewEncoder(w).Encode([]githubRelease{})
			return
		}
		_ = json.NewEncoder(w).Encode(releases)
	})

	cfg := &model.SourceConfig{Type: "github", Repo: "example/repo"}
	versions, err := GitHubSource{}.Fetch(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(versions) != 2 {
		t.Fatalf("expected 2 releases (draft dropped), got %d: %+v", len(versions), versions)
	}
	byRaw := make(map[string]Version, len(versions))
	for _, v := range versions {
		byRaw[v.Raw] = v
	}
	if v, ok := byRaw["v1.3.0-rc1"]; !ok || !v.Prerelease {
		t.Errorf("expected v1.3.0-rc1 to be marked prerelease, got %+v", byRaw["v1.3.0-rc1"])
	}
	if v, ok := byRaw["v1.2.0"]; !ok || v.Prerelease {
		t.Errorf("expected v1.2.0 to not be prerelease, got %+v", v)
	}
}

func TestGitHubSource_Fetch_Tags(t *testing.T) {
	tags := []githubTag{{Name: "v1.0.0", Commit: struct {
		SHA string `json:"sha"`
	}{SHA: "abc123"}}}
	withGitHubTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("page") != "1" {
			_ = json.NewEncoder(w).Encode([]githubTag{})
			return
		}
		_ = json.NewEncoder(w).Encode(tags)
	})

	cfg := &model.SourceConfig{Type: "github", Repo: "example/repo", Kind: "tags"}
	versions, err := GitHubSource{}.Fetch(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(versions) != 1 || versions[0].Extra["commit_sha"] != "abc123" {
		t.Errorf("unexpected versions: %+v", versions)
	}
}

func TestGitHubSource_Fetch_Pagination(t *testing.T) {
	// Page 1 returns a full page (githubPageSize), so a second page must be
	// requested; page 2 returns a partial page, stopping pagination.
	fullPage := make([]githubRelease, githubPageSize)
	for i := range fullPage {
		fullPage[i] = githubRelease{TagName: "v0.0." + string(rune('a'+i%26))}
	}
	calls := 0
	withGitHubTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Query().Get("page") == "1" {
			_ = json.NewEncoder(w).Encode(fullPage)
			return
		}
		_ = json.NewEncoder(w).Encode([]githubRelease{{TagName: "v2.0.0"}})
	})

	cfg := &model.SourceConfig{Type: "github", Repo: "example/repo"}
	versions, err := GitHubSource{}.Fetch(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 2 {
		t.Errorf("expected exactly 2 page requests, got %d", calls)
	}
	if len(versions) != githubPageSize+1 {
		t.Errorf("expected %d versions, got %d", githubPageSize+1, len(versions))
	}
}

func TestGitHubSource_Fetch_RateLimitError(t *testing.T) {
	withGitHubTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-RateLimit-Remaining", "0")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"message":"API rate limit exceeded"}`))
	})

	cfg := &model.SourceConfig{Type: "github", Repo: "example/repo"}
	_, err := GitHubSource{}.Fetch(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected an error for a 403 response")
	}
}

func TestGitHubSource_Fetch_UsesTokenFromEnv(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "gh-secret")
	var gotAuth string
	withGitHubTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		_ = json.NewEncoder(w).Encode([]githubRelease{})
	})

	cfg := &model.SourceConfig{Type: "github", Repo: "example/repo"}
	if _, err := (GitHubSource{}).Fetch(context.Background(), cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotAuth != "Bearer gh-secret" {
		t.Errorf("Authorization = %q, want Bearer gh-secret", gotAuth)
	}
}

func TestGitHubSource_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *model.SourceConfig
		wantErr bool
	}{
		{"valid releases", &model.SourceConfig{Type: "github", Repo: "a/b"}, false},
		{"valid tags", &model.SourceConfig{Type: "github", Repo: "a/b", Kind: "tags"}, false},
		{"missing repo", &model.SourceConfig{Type: "github"}, true},
		{"bad kind", &model.SourceConfig{Type: "github", Repo: "a/b", Kind: "commits"}, true},
		{"stray url field", &model.SourceConfig{Type: "github", Repo: "a/b", URL: "https://x"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := GitHubSource{}.Validate(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
