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
	got, branch, err := resolve.Resolve(repo.Root, bd, "feature/foo")
	if err != nil {
		t.Fatal(err)
	}
	if got != wtPath {
		t.Errorf("got %q, want %q", got, wtPath)
	}
	if branch != "feature/foo" {
		t.Errorf("branch = %q, want %q", branch, "feature/foo")
	}
}

func TestResolve_BranchNameDoesNotMatch(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	repoName := filepath.Base(repo.Root)

	// Create worktree with a directory name that doesn't match sanitize("my-branch")
	repo.CreateWorktree(repoName+"-worktrees/custom-dir", "my-branch")

	bd := baseDir(repo.Root)
	_, _, err := resolve.Resolve(repo.Root, bd, "my-branch")
	if err == nil {
		t.Error("expected error: branch name alone should not resolve when directory name differs")
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

	_, _, err := resolve.Resolve(repo.Root, bd, "fake-wt")
	if err == nil {
		t.Error("expected error for directory that exists but is not a worktree")
	}
}

func TestResolve_NotFound(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	bd := baseDir(repo.Root)

	_, _, err := resolve.Resolve(repo.Root, bd, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent identifier")
	}
}

func TestResolve_InvalidIdentifier(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	bd := baseDir(repo.Root)

	_, _, err := resolve.Resolve(repo.Root, bd, "/")
	if err == nil {
		t.Error("expected error for identifier that fails sanitization")
	}
}

