package resolve_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gin0606/gw/internal/resolve"
	"github.com/gin0606/gw/internal/testutil"
)

func baseDir(repoRoot string) string {
	repoName := filepath.Base(repoRoot)
	return filepath.Join(filepath.Dir(repoRoot), repoName+"-worktrees")
}

func TestResolve_SanitizedNameMatch(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	wtPath := repo.CreateWorktreeInBaseDir("feature/foo")

	bd := baseDir(repo.Root)
	got, err := resolve.Resolve(repo.Root, bd, "feature/foo")
	if err != nil {
		t.Fatal(err)
	}
	if got != wtPath {
		t.Errorf("got %q, want %q", got, wtPath)
	}
}

func TestResolve_BranchNameScan(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	repoName := filepath.Base(repo.Root)

	// Create worktree with a directory name that doesn't match sanitize("my-branch")
	wtPath := repo.CreateWorktree(repoName+"-worktrees/custom-dir", "my-branch")

	bd := baseDir(repo.Root)
	got, err := resolve.Resolve(repo.Root, bd, "my-branch")
	if err != nil {
		t.Fatal(err)
	}
	if got != wtPath {
		t.Errorf("got %q, want %q", got, wtPath)
	}
}

func TestResolve_SanitizedNameMatchPriority(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	repoName := filepath.Base(repo.Root)
	bd := baseDir(repo.Root)

	// Create a worktree whose sanitized name matches the identifier
	wtPath1 := repo.CreateWorktreeInBaseDir("feature/bar")

	// Create another worktree in base dir whose branch name is "feature/bar" but different dir name
	// This tests that sanitized name path match takes priority over branch name scan
	wtPath2 := repo.CreateWorktree(repoName+"-worktrees/other-dir", "feature/bar-alias")
	_ = wtPath2

	// "feature/bar" should match via sanitized name path match, not branch scan
	got, err := resolve.Resolve(repo.Root, bd, "feature/bar")
	if err != nil {
		t.Fatal(err)
	}
	if got != wtPath1 {
		t.Errorf("got %q, want %q (sanitized name match should take priority)", got, wtPath1)
	}
}

func TestResolve_DirectoryExistsButNotInWorktreeList(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	bd := baseDir(repo.Root)

	// Create directory but not as a git worktree
	dirPath := filepath.Join(bd, "fake-wt")
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		t.Fatal(err)
	}

	_, err := resolve.Resolve(repo.Root, bd, "fake-wt")
	if err == nil {
		t.Error("expected error for directory that exists but is not a worktree")
	}
}

func TestResolve_NotFound(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	bd := baseDir(repo.Root)

	_, err := resolve.Resolve(repo.Root, bd, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent identifier")
	}
}

func TestResolve_BranchScanExcludesOutsideBaseDir(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	bd := baseDir(repo.Root)

	// The main repo has branch "main" but is outside baseDir.
	// "gw go main" should NOT resolve to the main repo.
	_, err := resolve.Resolve(repo.Root, bd, "main")
	if err == nil {
		t.Error("expected error: main repo worktree should not match because it is outside baseDir")
	}
}
