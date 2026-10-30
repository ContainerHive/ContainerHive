package changes

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func commitFile(t *testing.T, worktree *git.Worktree, root, name, content string) plumbing.Hash {
	t.Helper()
	full := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := worktree.Add(name); err != nil {
		t.Fatal(err)
	}
	hash, err := worktree.Commit("change "+name, &git.CommitOptions{
		Author: &object.Signature{Name: "test", Email: "test@example.com", When: time.Now()},
	})
	if err != nil {
		t.Fatal(err)
	}
	return hash
}

func TestDiffFiles(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping git integration test in short mode")
	}

	root := t.TempDir()
	repo, err := git.PlainInit(root, false)
	if err != nil {
		t.Fatal(err)
	}
	worktree, err := repo.Worktree()
	if err != nil {
		t.Fatal(err)
	}

	base := commitFile(t, worktree, root, "images/base/Dockerfile", "FROM scratch\n")
	commitFile(t, worktree, root, "images/python/Dockerfile", "FROM __hive__/base:1\n")
	commitFile(t, worktree, root, "README.md", "# test\n")

	diff, err := DiffFiles(filepath.Join(root, "images"), base.String())
	if err != nil {
		t.Fatalf("DiffFiles() error = %v", err)
	}
	want := []string{"README.md", "images/python/Dockerfile"}
	if !slices.Equal(diff.Files, want) {
		t.Errorf("Files = %v, want %v", diff.Files, want)
	}
	wantRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	if gotRoot, _ := filepath.EvalSymlinks(diff.RepoRoot); gotRoot != wantRoot {
		t.Errorf("RepoRoot = %q, want %q", gotRoot, wantRoot)
	}

	if _, err := DiffFiles(root, "does-not-exist"); err == nil {
		t.Error("DiffFiles() with unknown base error = nil, want error")
	}
}
