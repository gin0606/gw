package git_test

import (
	"path/filepath"
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

func TestResolvesToCommit(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	repo.CreateTag("v1")
	repo.Commit("second commit marker")
	head := repo.RevParse("HEAD")

	tests := []struct {
		ref  string
		want bool
	}{
		{"main", true},
		{"origin/main", true},
		{"v1", true},
		{head, true},
		{":/second commit marker", true},
		{":/no such commit message", false},
		{"main@{99}", false},
		{"nonexistent", false},
		{"origin/nonexistent", false},
		{"HEAD^{tree}", false},
		{"--help", false},
	}
	for _, tt := range tests {
		t.Run(tt.ref, func(t *testing.T) {
			got, err := git.ResolvesToCommit(repo.Root, tt.ref)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("ResolvesToCommit(%q) = %v, want %v", tt.ref, got, tt.want)
			}
		})
	}
}

func TestResolvesToCommit_GitCannotRun(t *testing.T) {
	_, err := git.ResolvesToCommit(filepath.Join(t.TempDir(), "missing"), "main")
	if err == nil {
		t.Error("expected error when git cannot be started")
	}
}

func TestSymbolicFullName(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	head := repo.RevParse("HEAD")
	repo.UpdateRef("refs/tags/--force", head)

	tests := []struct {
		ref    string
		want   string
		wantOK bool
	}{
		{"main", "refs/heads/main", true},
		{"origin/main", "refs/remotes/origin/main", true},
		{"--force", "refs/tags/--force", true},
		{head, "", false},
		{"nonexistent", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.ref, func(t *testing.T) {
			got, ok, err := git.SymbolicFullName(repo.Root, tt.ref)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want || ok != tt.wantOK {
				t.Errorf("SymbolicFullName(%q) = (%q, %v), want (%q, %v)", tt.ref, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

func TestListLocalBranches(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	repo.CreateBranch("feature-a")
	repo.CreateBranch("feature-b")

	branches, err := git.ListLocalBranches(repo.Root)
	if err != nil {
		t.Fatal(err)
	}

	want := map[string]bool{"main": false, "feature-a": false, "feature-b": false}
	for _, b := range branches {
		if _, ok := want[b]; ok {
			want[b] = true
		}
	}
	for name, found := range want {
		if !found {
			t.Errorf("expected branch %q in list, got %v", name, branches)
		}
	}
}

func TestListLocalBranches_MainOnly(t *testing.T) {
	repo := testutil.NewTestRepo(t)

	branches, err := git.ListLocalBranches(repo.Root)
	if err != nil {
		t.Fatal(err)
	}

	found := false
	for _, b := range branches {
		if b == "main" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'main' in branches, got %v", branches)
	}
}

func TestListRefs(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	repo.CreateBranch("feature-x")
	repo.PushBranch("feature-x")
	repo.CreateTag("v1.0.0")

	refs, err := git.ListRefs(repo.Root)
	if err != nil {
		t.Fatal(err)
	}

	want := map[string]bool{
		"main":             false,
		"feature-x":        false,
		"origin/main":      false,
		"origin/feature-x": false,
		"v1.0.0":           false,
	}
	for _, r := range refs {
		if _, ok := want[r]; ok {
			want[r] = true
		}
	}
	for name, found := range want {
		if !found {
			t.Errorf("expected ref %q in list, got %v", name, refs)
		}
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

func TestListWorktrees_PathWithNewline(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	wtPath := repo.CreateWorktree("wt\nnewline", "feature/newline")

	worktrees, err := git.ListWorktrees(repo.Root)
	if err != nil {
		t.Fatal(err)
	}

	want := []git.Worktree{
		{Path: repo.Root, Branch: "main"},
		{Path: wtPath, Branch: "feature/newline"},
	}
	if len(worktrees) != len(want) {
		t.Fatalf("got %d worktrees %q, want %q", len(worktrees), worktrees, want)
	}
	for i := range want {
		if worktrees[i] != want[i] {
			t.Errorf("worktrees[%d] = %q, want %q", i, worktrees[i], want[i])
		}
	}
}

func TestListWorktrees_DetachedHead(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	branchPath := repo.CreateWorktree("wt-branch", "feature/branch")
	detachedPath := repo.CreateDetachedWorktree("wt-detached")

	worktrees, err := git.ListWorktrees(repo.Root)
	if err != nil {
		t.Fatal(err)
	}

	want := []git.Worktree{
		{Path: repo.Root, Branch: "main"},
		{Path: branchPath, Branch: "feature/branch"},
		{Path: detachedPath, Branch: ""},
	}
	if len(worktrees) != len(want) {
		t.Fatalf("got %d worktrees %q, want %q", len(worktrees), worktrees, want)
	}
	for i := range want {
		if worktrees[i] != want[i] {
			t.Errorf("worktrees[%d] = %q, want %q", i, worktrees[i], want[i])
		}
	}
}
