package git_test

import (
	"testing"

	"github.com/gin0606/gw/internal/git"
	"github.com/gin0606/gw/internal/testutil"
)

func TestRepoRoot_MainRepo(t *testing.T) {
	repo := testutil.NewTestRepo(t)

	root, err := git.RepoRoot(repo.Root)
	if err != nil {
		t.Fatal(err)
	}
	if root != repo.Root {
		t.Errorf("got %q, want %q", root, repo.Root)
	}
}

func TestRepoRoot_Worktree(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	wtPath := repo.CreateWorktree("test-wt", "test-branch")

	root, err := git.RepoRoot(wtPath)
	if err != nil {
		t.Fatal(err)
	}
	if root != repo.Root {
		t.Errorf("got %q, want %q", root, repo.Root)
	}
}

func TestRepoRoot_OutsideGitRepo(t *testing.T) {
	tmpDir := t.TempDir()
	_, err := git.RepoRoot(tmpDir)
	if err == nil {
		t.Error("expected error for non-git directory")
	}
}

func TestDefaultBranch_Set(t *testing.T) {
	repo := testutil.NewTestRepo(t)

	branch, err := git.DefaultBranch(repo.Root)
	if err != nil {
		t.Fatal(err)
	}
	if branch != "main" {
		t.Errorf("got %q, want %q", branch, "main")
	}
}

func TestDefaultBranch_NotSet(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	repo.DeleteOriginHead()

	_, err := git.DefaultBranch(repo.Root)
	if err == nil {
		t.Error("expected error when origin/HEAD is not set")
	}
}

func TestBranchExists_Exists(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	repo.CreateBranch("feature-test")

	exists, err := git.BranchExists(repo.Root, "feature-test")
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Error("expected branch to exist")
	}
}

func TestBranchExists_NotExists(t *testing.T) {
	repo := testutil.NewTestRepo(t)

	exists, err := git.BranchExists(repo.Root, "nonexistent")
	if err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Error("expected branch not to exist")
	}
}

func TestRemoteRefExists_Exists(t *testing.T) {
	repo := testutil.NewTestRepo(t)

	exists, err := git.RemoteRefExists(repo.Root, "origin/main")
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Error("expected remote ref to exist")
	}
}

func TestRemoteRefExists_NotExists(t *testing.T) {
	repo := testutil.NewTestRepo(t)

	exists, err := git.RemoteRefExists(repo.Root, "origin/nonexistent")
	if err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Error("expected remote ref not to exist")
	}
}

func TestListWorktrees_MainOnly(t *testing.T) {
	repo := testutil.NewTestRepo(t)

	worktrees, err := git.ListWorktrees(repo.Root)
	if err != nil {
		t.Fatal(err)
	}

	if len(worktrees) != 1 {
		t.Fatalf("got %d worktrees, want 1", len(worktrees))
	}
	if worktrees[0].Path != repo.Root {
		t.Errorf("got path %q, want %q", worktrees[0].Path, repo.Root)
	}
	if worktrees[0].Branch != "main" {
		t.Errorf("got branch %q, want %q", worktrees[0].Branch, "main")
	}
}

func TestListWorktrees_WithWorktrees(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	wtPath := repo.CreateWorktreeInBaseDir("feature/test")

	worktrees, err := git.ListWorktrees(repo.Root)
	if err != nil {
		t.Fatal(err)
	}

	if len(worktrees) != 2 {
		t.Fatalf("got %d worktrees, want 2", len(worktrees))
	}

	var found bool
	for _, wt := range worktrees {
		if wt.Path == wtPath && wt.Branch == "feature/test" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("worktree with path %q and branch %q not found in %v", wtPath, "feature/test", worktrees)
	}
}
