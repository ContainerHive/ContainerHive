package changes

import (
	"fmt"
	"slices"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// GitDiff holds the files changed between the merge base of a base revision
// and HEAD.
type GitDiff struct {
	// RepoRoot is the absolute worktree root of the repository.
	RepoRoot string
	// Files are slash-separated paths relative to RepoRoot, sorted.
	Files []string
}

// DiffFiles opens the repository containing dir and returns the files changed
// between the merge base of base and HEAD. Renamed files report both paths.
// Uncommitted changes are not part of the diff.
func DiffFiles(dir, base string) (*GitDiff, error) {
	repo, err := git.PlainOpenWithOptions(dir, &git.PlainOpenOptions{DetectDotGit: true})
	if err != nil {
		return nil, fmt.Errorf("opening git repository: %w", err)
	}
	worktree, err := repo.Worktree()
	if err != nil {
		return nil, fmt.Errorf("opening git worktree: %w", err)
	}

	head, err := commitAt(repo, plumbing.Revision(plumbing.HEAD))
	if err != nil {
		return nil, fmt.Errorf("resolving HEAD: %w", err)
	}
	baseCommit, err := commitAt(repo, plumbing.Revision(base))
	if err != nil {
		return nil, fmt.Errorf("resolving base %q: %w", base, err)
	}

	mergeBases, err := baseCommit.MergeBase(head)
	if err != nil {
		return nil, fmt.Errorf("finding merge base of %q and HEAD: %w", base, err)
	}
	if len(mergeBases) == 0 {
		return nil, fmt.Errorf("no merge base of %q and HEAD, is the clone shallow?", base)
	}

	files, err := changedFiles(mergeBases[0], head)
	if err != nil {
		return nil, err
	}
	return &GitDiff{RepoRoot: worktree.Filesystem.Root(), Files: files}, nil
}

func commitAt(repo *git.Repository, revision plumbing.Revision) (*object.Commit, error) {
	hash, err := repo.ResolveRevision(revision)
	if err != nil {
		return nil, err
	}
	return repo.CommitObject(*hash)
}

func changedFiles(from, to *object.Commit) ([]string, error) {
	fromTree, err := from.Tree()
	if err != nil {
		return nil, fmt.Errorf("reading tree of %s: %w", from.Hash, err)
	}
	toTree, err := to.Tree()
	if err != nil {
		return nil, fmt.Errorf("reading tree of %s: %w", to.Hash, err)
	}
	diff, err := object.DiffTree(fromTree, toTree)
	if err != nil {
		return nil, fmt.Errorf("diffing %s and %s: %w", from.Hash, to.Hash, err)
	}

	seen := make(map[string]bool)
	for _, change := range diff {
		for _, name := range []string{change.From.Name, change.To.Name} {
			if name != "" {
				seen[name] = true
			}
		}
	}
	files := make([]string, 0, len(seen))
	for name := range seen {
		files = append(files, name)
	}
	slices.Sort(files)
	return files, nil
}
