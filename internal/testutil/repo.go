package testutil

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin0606/gw/internal/pathutil"
)

// TestRepo represents a temporary git repository for testing.
type TestRepo struct {
	Root     string // Main repository root (symlink-resolved)
	BareRoot string // Bare repository path (origin)
	t        *testing.T
}

// NewTestRepo creates a temporary git repository with a bare origin remote.
// The repository has an initial commit on "main" and origin/HEAD is set.
func NewTestRepo(t *testing.T) *TestRepo {
	t.Helper()

	dir := t.TempDir()
	// Resolve symlinks for consistent path comparison (macOS /var -> /private/var)
	dir, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}

	bareDir := filepath.Join(dir, "origin.git")
	gitCmd(t, "", "init", "--bare", "-b", "main", bareDir)

	repoDir := filepath.Join(dir, "repo")
	gitCmd(t, "", "init", "-b", "main", repoDir)
	gitCmd(t, repoDir, "remote", "add", "origin", bareDir)

	if err := os.WriteFile(filepath.Join(repoDir, ".gitkeep"), []byte{}, 0644); err != nil {
		t.Fatal(err)
	}
	gitCmd(t, repoDir, "add", ".")
	gitCmd(t, repoDir, "commit", "-m", "initial")
	gitCmd(t, repoDir, "push", "-u", "origin", "main")
	gitCmd(t, repoDir, "remote", "set-head", "origin", "--auto")

	return &TestRepo{
		Root:     repoDir,
		BareRoot: bareDir,
		t:        t,
	}
}

// CreateBranch creates a new local branch at the current HEAD.
func (r *TestRepo) CreateBranch(name string) {
	r.t.Helper()
	gitCmd(r.t, r.Root, "branch", name)
}

// PushBranch pushes a branch to origin.
func (r *TestRepo) PushBranch(name string) {
	r.t.Helper()
	gitCmd(r.t, r.Root, "push", "origin", name)
}

// CreateTag creates a lightweight tag at the current HEAD.
func (r *TestRepo) CreateTag(name string) {
	r.t.Helper()
	gitCmd(r.t, r.Root, "tag", name)
}

// DeleteOriginHead removes origin/HEAD symbolic ref.
func (r *TestRepo) DeleteOriginHead() {
	r.t.Helper()
	gitCmd(r.t, r.Root, "remote", "set-head", "origin", "--delete")
}

// DeleteRemoteRef deletes a remote tracking ref (e.g., "origin/main").
func (r *TestRepo) DeleteRemoteRef(ref string) {
	r.t.Helper()
	gitCmd(r.t, r.Root, "update-ref", "-d", "refs/remotes/"+ref)
}

// CreateWorktree creates a git worktree with a new branch and returns its absolute path.
func (r *TestRepo) CreateWorktree(name, branch string) string {
	r.t.Helper()
	wtPath := filepath.Join(filepath.Dir(r.Root), name)
	gitCmd(r.t, r.Root, "worktree", "add", wtPath, "-b", branch)
	return wtPath
}

// WriteConfig writes .gw/config with the given TOML content.
func (r *TestRepo) WriteConfig(content string) {
	r.t.Helper()
	configDir := filepath.Join(r.Root, ".gw")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		r.t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config"), []byte(content), 0644); err != nil {
		r.t.Fatal(err)
	}
}

// WriteHook creates a hook script in .gw/hooks/ with execute permission.
func (r *TestRepo) WriteHook(name, content string) {
	r.t.Helper()
	hookDir := filepath.Join(r.Root, ".gw", "hooks")
	if err := os.MkdirAll(hookDir, 0755); err != nil {
		r.t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(hookDir, name), []byte(content), 0755); err != nil {
		r.t.Fatal(err)
	}
}

// WriteHookNoExec creates a hook script without execute permission.
func (r *TestRepo) WriteHookNoExec(name, content string) {
	r.t.Helper()
	hookDir := filepath.Join(r.Root, ".gw", "hooks")
	if err := os.MkdirAll(hookDir, 0755); err != nil {
		r.t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(hookDir, name), []byte(content), 0644); err != nil {
		r.t.Fatal(err)
	}
}

// CreateWorktreeInBaseDir creates a worktree in the default base directory (<repo-name>-worktrees/<sanitized-branch>).
func (r *TestRepo) CreateWorktreeInBaseDir(branch string) string {
	r.t.Helper()
	repoName := filepath.Base(r.Root)
	baseDir := filepath.Join(filepath.Dir(r.Root), repoName+"-worktrees")
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		r.t.Fatal(err)
	}
	wtPath, err := pathutil.ComputePath(baseDir, branch)
	if err != nil {
		r.t.Fatal(err)
	}
	gitCmd(r.t, r.Root, "worktree", "add", wtPath, "-b", branch)
	return wtPath
}

func gitCmd(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Test",
		"GIT_AUTHOR_EMAIL=test@test.com",
		"GIT_COMMITTER_NAME=Test",
		"GIT_COMMITTER_EMAIL=test@test.com",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v (dir=%s): %v\n%s", args, dir, err, string(out))
	}
	return strings.TrimSpace(string(out))
}
