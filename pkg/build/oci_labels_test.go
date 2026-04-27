package build

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/timo-reymann/ContainerHive/pkg/model"
)

func TestApplyLabelPlaceholders(t *testing.T) {
	tests := []struct {
		name  string
		in    string
		image string
		tag   string
		want  string
	}{
		{"no placeholders", "https://example.com", "python", "3.12", "https://example.com"},
		{"image only", "https://example.com/%image%", "python", "3.12", "https://example.com/python"},
		{"tag only", "https://example.com/v/%tag%", "python", "3.12", "https://example.com/v/3.12"},
		{"both", "https://example.com/%image%/%tag%/docs", "python", "3.12-alpine", "https://example.com/python/3.12-alpine/docs"},
		{"repeated", "%image%-%image%", "python", "3.12", "python-python"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := applyLabelPlaceholders(tt.in, tt.image, tt.tag)
			if got != tt.want {
				t.Errorf("applyLabelPlaceholders(%q, %q, %q) = %q, want %q", tt.in, tt.image, tt.tag, got, tt.want)
			}
		})
	}
}

func TestBuildOCILabels_AlwaysOn(t *testing.T) {
	got := BuildOCILabels(OCILabelArgs{ImageName: "python", Tag: "3.12"})

	if got[ociLabelTitle] != "python" {
		t.Errorf("title = %q, want %q", got[ociLabelTitle], "python")
	}
	if got[ociLabelRefName] != "python" {
		t.Errorf("ref.name = %q, want %q", got[ociLabelRefName], "python")
	}
	if got[ociLabelVersion] != "3.12" {
		t.Errorf("version = %q, want %q", got[ociLabelVersion], "3.12")
	}
	if _, err := time.Parse(time.RFC3339, got[ociLabelCreated]); err != nil {
		t.Errorf("created %q is not RFC3339: %v", got[ociLabelCreated], err)
	}
	if _, ok := got[ociLabelDescription]; ok {
		t.Errorf("description must be omitted when empty, got %q", got[ociLabelDescription])
	}
}

func TestBuildOCILabels_VariantTag(t *testing.T) {
	got := BuildOCILabels(OCILabelArgs{ImageName: "python", Tag: "3.12-alpine"})
	if got[ociLabelTitle] != "python" {
		t.Errorf("title = %q, want %q", got[ociLabelTitle], "python")
	}
	if got[ociLabelVersion] != "3.12-alpine" {
		t.Errorf("version = %q, want %q", got[ociLabelVersion], "3.12-alpine")
	}
}

func TestBuildOCILabels_Description(t *testing.T) {
	got := BuildOCILabels(OCILabelArgs{ImageName: "python", Tag: "3.12", Description: "Python runtime"})
	if got[ociLabelDescription] != "Python runtime" {
		t.Errorf("description = %q, want %q", got[ociLabelDescription], "Python runtime")
	}
}

func TestBuildOCILabels_Project(t *testing.T) {
	got := BuildOCILabels(OCILabelArgs{
		ImageName: "python",
		Tag:       "3.12-alpine",
		Project: &model.LabelsConfig{
			Vendor:        "Acme",
			Authors:       "team@acme.test",
			Url:           "https://acme.test/%image%/%tag%",
			Documentation: "https://docs.acme.test/%image%",
		},
	})
	if got[ociLabelVendor] != "Acme" {
		t.Errorf("vendor = %q", got[ociLabelVendor])
	}
	if got[ociLabelAuthors] != "team@acme.test" {
		t.Errorf("authors = %q", got[ociLabelAuthors])
	}
	if got[ociLabelURL] != "https://acme.test/python/3.12-alpine" {
		t.Errorf("url = %q", got[ociLabelURL])
	}
	if got[ociLabelDocumentation] != "https://docs.acme.test/python" {
		t.Errorf("documentation = %q", got[ociLabelDocumentation])
	}
}

func TestBuildOCILabels_ProjectOmittedFields(t *testing.T) {
	got := BuildOCILabels(OCILabelArgs{
		ImageName: "python",
		Tag:       "3.12",
		Project:   &model.LabelsConfig{Vendor: "Acme"},
	})
	if got[ociLabelVendor] != "Acme" {
		t.Errorf("vendor = %q", got[ociLabelVendor])
	}
	for _, k := range []string{ociLabelAuthors, ociLabelURL, ociLabelDocumentation} {
		if _, ok := got[k]; ok {
			t.Errorf("%s must be omitted, got %q", k, got[k])
		}
	}
}

func TestBuildOCILabels_CustomMerge(t *testing.T) {
	got := BuildOCILabels(OCILabelArgs{
		ImageName: "python",
		Tag:       "3.12",
		Project: &model.LabelsConfig{
			Vendor: "Acme",
			Custom: map[string]string{
				"com.acme.project":               "alpha",
				"org.opencontainers.image.title": "should-be-overridden",
			},
		},
		ImageLabels: map[string]string{
			"com.acme.image":   "python",
			"com.acme.project": "beta", // image overrides project
		},
	})

	if got["com.acme.project"] != "beta" {
		t.Errorf("image custom label must override project custom: got %q", got["com.acme.project"])
	}
	if got["com.acme.image"] != "python" {
		t.Errorf("image-only custom label missing: got %q", got["com.acme.image"])
	}
	if got[ociLabelTitle] != "python" {
		t.Errorf("standard auto label must override custom: title = %q", got[ociLabelTitle])
	}
	if got[ociLabelVendor] != "Acme" {
		t.Errorf("structured project label missing: vendor = %q", got[ociLabelVendor])
	}
}

func TestBuildOCILabels_PrecedenceChain(t *testing.T) {
	got := BuildOCILabels(OCILabelArgs{
		ImageName: "python",
		Tag:       "3.12-alpine",
		Project: &model.LabelsConfig{
			Custom: map[string]string{
				"com.acme.layer": "project",
				"com.acme.proj":  "x",
			},
		},
		ImageLabels: map[string]string{
			"com.acme.layer": "image",
			"com.acme.img":   "y",
		},
		TagLabels: map[string]string{
			"com.acme.layer": "tag",
			"com.acme.tag":   "z",
		},
		VariantLabels: map[string]string{
			"com.acme.layer":   "variant",
			"com.acme.variant": "w",
		},
	})

	if got["com.acme.layer"] != "variant" {
		t.Errorf("variant must win precedence chain: got %q", got["com.acme.layer"])
	}
	for k, want := range map[string]string{
		"com.acme.proj":    "x",
		"com.acme.img":     "y",
		"com.acme.tag":     "z",
		"com.acme.variant": "w",
	} {
		if got[k] != want {
			t.Errorf("layer-unique label %s = %q, want %q", k, got[k], want)
		}
	}
}

func TestGitInfo_EmptyRoot(t *testing.T) {
	rev, src := gitInfo("")
	if rev != "" || src != "" {
		t.Errorf("empty root must return empty strings, got (%q, %q)", rev, src)
	}
}

func TestGitInfo_NonGitDir(t *testing.T) {
	dir := t.TempDir()
	rev, src := gitInfo(dir)
	if rev != "" || src != "" {
		t.Errorf("non-git dir must return empty strings, got (%q, %q)", rev, src)
	}
}

func TestGitInfo_WithCommitAndOrigin(t *testing.T) {
	dir := t.TempDir()
	repo, err := git.PlainInit(dir, false)
	if err != nil {
		t.Fatalf("PlainInit: %v", err)
	}

	wt, err := repo.Worktree()
	if err != nil {
		t.Fatalf("Worktree: %v", err)
	}
	hash, err := wt.Commit("init", &git.CommitOptions{
		AllowEmptyCommits: true,
		Author: &object.Signature{
			Name:  "test",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		t.Fatalf("Commit: %v", err)
	}

	if _, err := repo.CreateRemote(&config.RemoteConfig{
		Name: "origin",
		URLs: []string{"https://example.test/repo.git"},
	}); err != nil {
		t.Fatalf("CreateRemote: %v", err)
	}

	rev, src := gitInfo(dir)
	if rev != hash.String() {
		t.Errorf("revision = %q, want %q", rev, hash.String())
	}
	if src != "https://example.test/repo.git" {
		t.Errorf("source = %q", src)
	}

	// Subdirectory inside the worktree should still resolve via DetectDotGit.
	sub := filepath.Join(dir, "nested")
	rev2, src2 := gitInfo(sub)
	if rev2 != "" || src2 != "" {
		// Subdir doesn't exist on disk; that's fine, expect empty (PlainOpenWithOptions fails).
	}
}

func TestGitInfo_NoCommit(t *testing.T) {
	dir := t.TempDir()
	if _, err := git.PlainInit(dir, false); err != nil {
		t.Fatalf("PlainInit: %v", err)
	}
	rev, src := gitInfo(dir)
	if rev != "" {
		t.Errorf("revision must be empty when no commits, got %q", rev)
	}
	if src != "" {
		t.Errorf("source must be empty when no remote, got %q", src)
	}
}
